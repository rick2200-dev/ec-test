// Package tracing wires OpenTelemetry distributed tracing into every service.
//
// Each service calls Init early in main to install a global TracerProvider that
// exports OTLP/HTTP to the configured collector (Jaeger all-in-one in local
// docker-compose). The returned shutdown flushes pending spans on graceful
// termination. When OTEL_SDK_DISABLED=true, Init is a no-op so unit tests and
// CI do not require a running collector.
package tracing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config captures the inputs Init needs.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string
	Insecure       bool
	Sampler        string
	SamplerArg     string
	Disabled       bool
}

// LoadConfig reads OTEL_* environment variables with sensible defaults for
// local docker-compose.
func LoadConfig(defaultServiceName string) Config {
	endpoint := firstNonEmpty(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), "http://jaeger:4318")
	return Config{
		ServiceName:    firstNonEmpty(os.Getenv("OTEL_SERVICE_NAME"), defaultServiceName),
		ServiceVersion: firstNonEmpty(os.Getenv("SERVICE_VERSION"), "dev"),
		Environment:    firstNonEmpty(os.Getenv("DEPLOYMENT_ENVIRONMENT"), "local"),
		Endpoint:       endpoint,
		Insecure:       strings.HasPrefix(endpoint, "http://"),
		Sampler:        firstNonEmpty(os.Getenv("OTEL_TRACES_SAMPLER"), "parentbased_always_on"),
		SamplerArg:     os.Getenv("OTEL_TRACES_SAMPLER_ARG"),
		Disabled:       parseBool(os.Getenv("OTEL_SDK_DISABLED")),
	}
}

// Init installs the global TracerProvider and TextMapPropagator. The returned
// shutdown must be invoked on process exit.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if cfg.Disabled {
		// Leave the global noop provider in place; still install W3C
		// propagation so extraction/injection helpers behave consistently.
		otel.SetTextMapPropagator(compositePropagator())
		return func(context.Context) error { return nil }, nil
	}

	if cfg.ServiceName == "" {
		return nil, errors.New("tracing: ServiceName is required")
	}

	endpoint, err := parseOTLPEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse OTLP endpoint: %w", err)
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpointURL(endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
		semconv.DeploymentEnvironment(cfg.Environment),
	))
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(buildSampler(cfg.Sampler, cfg.SamplerArg)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(compositePropagator())

	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(ctx)
	}, nil
}

func compositePropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
}

func buildSampler(name, arg string) sdktrace.Sampler {
	switch strings.ToLower(name) {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	case "traceidratio":
		ratio := parseFloat(arg, 1.0)
		return sdktrace.TraceIDRatioBased(ratio)
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample())
	case "parentbased_traceidratio":
		ratio := parseFloat(arg, 1.0)
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	case "parentbased_always_on":
		fallthrough
	default:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}

// parseOTLPEndpoint accepts either a base URL ("http://jaeger:4318") or an
// explicit traces URL ("http://jaeger:4318/v1/traces") and returns the full
// traces URL expected by otlptracehttp.WithEndpointURL.
func parseOTLPEndpoint(raw string) (string, error) {
	raw = strings.TrimRight(raw, "/")
	if raw == "" {
		return "", errors.New("empty endpoint")
	}
	if strings.HasSuffix(raw, "/v1/traces") {
		return raw, nil
	}
	return raw + "/v1/traces", nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func parseBool(s string) bool {
	if s == "" {
		return false
	}
	b, _ := strconv.ParseBool(s)
	return b
}

func parseFloat(s string, def float64) float64 {
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}
