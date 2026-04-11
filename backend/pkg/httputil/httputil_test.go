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
	if !errors.As(err, &appErr) || appErr.Code != http.StatusBadRequest {
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
