package grpcclient

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	catalogv1 "github.com/Riku-KANO/ec-test/services/catalog/api/gen/go/catalog/v1"
	inventoryv1 "github.com/Riku-KANO/ec-test/services/inventory/api/gen/go/inventory/v1"
	orderv1 "github.com/Riku-KANO/ec-test/services/order/api/gen/go/order/v1"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/config"
)

// internalTokenCreds attaches a shared secret as x-internal-token gRPC
// metadata on every RPC call so the downstream service can authenticate
// the caller.
type internalTokenCreds struct {
	token string
}

func (c internalTokenCreds) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{"x-internal-token": c.token}, nil
}

func (c internalTokenCreds) RequireTransportSecurity() bool { return false }

// GRPCClients holds gRPC client connections and typed clients for downstream services.
type GRPCClients struct {
	catalogConn   *grpc.ClientConn
	inventoryConn *grpc.ClientConn
	orderConn     *grpc.ClientConn

	CatalogClient   catalogv1.CatalogServiceClient
	InventoryClient inventoryv1.InventoryServiceClient
	OrderClient     orderv1.OrderServiceClient
}

// NewGRPCClients creates gRPC connections to each downstream service.
func NewGRPCClients(cfg config.Config) (*GRPCClients, error) {
	if cfg.CatalogInternalToken == "" {
		return nil, fmt.Errorf("CATALOG_INTERNAL_TOKEN is required; catalog gRPC rejects unauthenticated calls")
	}

	baseOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	catalogOpts := append(append([]grpc.DialOption{}, baseOpts...),
		grpc.WithPerRPCCredentials(internalTokenCreds{token: cfg.CatalogInternalToken}),
	)
	catalogConn, err := grpc.NewClient(cfg.CatalogGRPCAddr, catalogOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to catalog gRPC: %w", err)
	}

	inventoryConn, err := grpc.NewClient(cfg.InventoryGRPCAddr, baseOpts...)
	if err != nil {
		_ = catalogConn.Close()
		return nil, fmt.Errorf("failed to connect to inventory gRPC: %w", err)
	}

	orderConn, err := grpc.NewClient(cfg.OrderGRPCAddr, baseOpts...)
	if err != nil {
		_ = catalogConn.Close()
		_ = inventoryConn.Close()
		return nil, fmt.Errorf("failed to connect to order gRPC: %w", err)
	}

	slog.Info("gRPC clients connected",
		"catalog", cfg.CatalogGRPCAddr,
		"inventory", cfg.InventoryGRPCAddr,
		"order", cfg.OrderGRPCAddr,
	)

	return &GRPCClients{
		catalogConn:     catalogConn,
		inventoryConn:   inventoryConn,
		orderConn:       orderConn,
		CatalogClient:   catalogv1.NewCatalogServiceClient(catalogConn),
		InventoryClient: inventoryv1.NewInventoryServiceClient(inventoryConn),
		OrderClient:     orderv1.NewOrderServiceClient(orderConn),
	}, nil
}

// Close closes all gRPC connections, logging any errors.
func (c *GRPCClients) Close() {
	if c.catalogConn != nil {
		if err := c.catalogConn.Close(); err != nil {
			slog.Warn("failed to close catalog gRPC connection", "error", err)
		}
	}
	if c.inventoryConn != nil {
		if err := c.inventoryConn.Close(); err != nil {
			slog.Warn("failed to close inventory gRPC connection", "error", err)
		}
	}
	if c.orderConn != nil {
		if err := c.orderConn.Close(); err != nil {
			slog.Warn("failed to close order gRPC connection", "error", err)
		}
	}
}
