package errors_test

import (
	"errors"
	"net/http"
	"testing"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
)

func TestNew(t *testing.T) {
	sentinel := errors.New("underlying")
	err := apperrors.New(http.StatusTeapot, "test message", sentinel)
	if err.Code != http.StatusTeapot {
		t.Errorf("expected code %d, got %d", http.StatusTeapot, err.Code)
	}
	if err.Message != "test message" {
		t.Errorf("expected message %q, got %q", "test message", err.Message)
	}
	if !errors.Is(err, sentinel) {
		t.Error("expected err to wrap sentinel")
	}
}

func TestNotFound(t *testing.T) {
	err := apperrors.NotFound("item not found")
	if err.Code != http.StatusNotFound {
		t.Errorf("expected %d, got %d", http.StatusNotFound, err.Code)
	}
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Error("expected ErrNotFound")
	}
}

func TestBadRequest(t *testing.T) {
	err := apperrors.BadRequest("bad input")
	if err.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, err.Code)
	}
	if !errors.Is(err, apperrors.ErrBadRequest) {
		t.Error("expected ErrBadRequest")
	}
}

func TestUnauthorized(t *testing.T) {
	err := apperrors.Unauthorized("not authenticated")
	if err.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, err.Code)
	}
	if !errors.Is(err, apperrors.ErrUnauthorized) {
		t.Error("expected ErrUnauthorized")
	}
}

func TestForbidden(t *testing.T) {
	err := apperrors.Forbidden("access denied")
	if err.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, err.Code)
	}
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Error("expected ErrForbidden")
	}
}

func TestConflict(t *testing.T) {
	err := apperrors.Conflict("already exists")
	if err.Code != http.StatusConflict {
		t.Errorf("expected %d, got %d", http.StatusConflict, err.Code)
	}
	if !errors.Is(err, apperrors.ErrConflict) {
		t.Error("expected ErrConflict")
	}
}

func TestInternal(t *testing.T) {
	underlying := errors.New("db error")
	err := apperrors.Internal("something went wrong", underlying)
	if err.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, err.Code)
	}
	if !errors.Is(err, underlying) {
		t.Error("expected to wrap underlying error")
	}
}

func TestInternal_NilCause(t *testing.T) {
	err := apperrors.Internal("something went wrong", nil)
	if err.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, err.Code)
	}
	if err.Error() != "something went wrong" {
		t.Errorf("unexpected error string: %q", err.Error())
	}
}

func TestAppError_Error(t *testing.T) {
	cause := errors.New("cause")
	err := apperrors.New(400, "msg", cause)
	want := "msg: cause"
	if err.Error() != want {
		t.Errorf("expected %q, got %q", want, err.Error())
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("cause")
	err := apperrors.New(400, "msg", cause)
	if !errors.Is(err, cause) {
		t.Error("Unwrap should expose the underlying cause")
	}
}
