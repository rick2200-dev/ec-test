package httputil_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
)

func TestJSON_StatusAndBody(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.JSON(w, http.StatusCreated, map[string]string{"key": "value"})

	res := w.Result()
	if res.StatusCode != http.StatusCreated {
		t.Errorf("expected %d, got %d", http.StatusCreated, res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["key"] != "value" {
		t.Errorf("expected body key=value, got %v", body)
	}
}

func TestJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.JSON(w, http.StatusNoContent, nil)

	res := w.Result()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("expected %d, got %d", http.StatusNoContent, res.StatusCode)
	}
	b, _ := io.ReadAll(res.Body)
	if len(b) != 0 {
		t.Errorf("expected empty body for nil data, got %q", string(b))
	}
}

func TestError_AppError(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.Error(w, apperrors.NotFound("thing not found"))

	res := w.Result()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected %d, got %d", http.StatusNotFound, res.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["error"] != "thing not found" {
		t.Errorf("unexpected error message: %q", body["error"])
	}
}

func TestError_AppErrorWithCode(t *testing.T) {
	// Errors carrying a semantic Code should surface that code in the JSON
	// body alongside the message, so clients can switch display patterns
	// on codes like "DUPLICATE_EMAIL" without parsing human text.
	w := httptest.NewRecorder()
	httputil.Error(w, apperrors.BadRequest("email already registered").WithCode("DUPLICATE_EMAIL"))

	res := w.Result()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, res.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["error"] != "email already registered" {
		t.Errorf("unexpected error message: %q", body["error"])
	}
	if body["code"] != "DUPLICATE_EMAIL" {
		t.Errorf("expected code=DUPLICATE_EMAIL, got %q", body["code"])
	}
}

func TestError_AppErrorNoCodeOmitted(t *testing.T) {
	// When no Code is attached, the `code` field must be absent (not an
	// empty string). This pins the omitempty behavior so legacy handlers
	// keep producing the exact same body they did before.
	w := httptest.NewRecorder()
	httputil.Error(w, apperrors.BadRequest("bad input"))

	res := w.Result()
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if _, present := body["code"]; present {
		t.Errorf("expected no `code` field for uncoded error, got body=%v", body)
	}
}

func TestError_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.Error(w, errors.New("some internal error"))

	res := w.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, res.StatusCode)
	}
}

func TestDecode_Valid(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	body := `{"name":"test"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var p payload
	if err := httputil.Decode(r, &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "test" {
		t.Errorf("expected name=test, got %q", p.Name)
	}
}

func TestDecode_NilBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	var p struct{}
	err := httputil.Decode(r, &p)
	if err == nil {
		t.Fatal("expected error for nil body")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Status != http.StatusBadRequest {
		t.Errorf("expected 400 AppError, got %v", err)
	}
}

func TestDecode_UnknownField(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"unknown":"field"}`))
	var p struct {
		Name string `json:"name"`
	}
	err := httputil.Decode(r, &p)
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}
