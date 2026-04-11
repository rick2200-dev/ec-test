package errors_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
)

func TestNew(t *testing.T) {
	sentinel := errors.New("underlying")
	err := apperrors.New(http.StatusTeapot, "test message", sentinel)
	if err.Status != http.StatusTeapot {
		t.Errorf("expected status %d, got %d", http.StatusTeapot, err.Status)
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
	if err.Status != http.StatusNotFound {
		t.Errorf("expected %d, got %d", http.StatusNotFound, err.Status)
	}
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Error("expected ErrNotFound")
	}
}

func TestBadRequest(t *testing.T) {
	err := apperrors.BadRequest("bad input")
	if err.Status != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, err.Status)
	}
	if !errors.Is(err, apperrors.ErrBadRequest) {
		t.Error("expected ErrBadRequest")
	}
}

func TestUnauthorized(t *testing.T) {
	err := apperrors.Unauthorized("not authenticated")
	if err.Status != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, err.Status)
	}
	if !errors.Is(err, apperrors.ErrUnauthorized) {
		t.Error("expected ErrUnauthorized")
	}
}

func TestForbidden(t *testing.T) {
	err := apperrors.Forbidden("access denied")
	if err.Status != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, err.Status)
	}
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Error("expected ErrForbidden")
	}
}

func TestConflict(t *testing.T) {
	err := apperrors.Conflict("already exists")
	if err.Status != http.StatusConflict {
		t.Errorf("expected %d, got %d", http.StatusConflict, err.Status)
	}
	if !errors.Is(err, apperrors.ErrConflict) {
		t.Error("expected ErrConflict")
	}
}

func TestInternal(t *testing.T) {
	underlying := errors.New("db error")
	err := apperrors.Internal("something went wrong", underlying)
	if err.Status != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, err.Status)
	}
	if !errors.Is(err, underlying) {
		t.Error("expected to wrap underlying error")
	}
}

func TestInternal_NilCause(t *testing.T) {
	err := apperrors.Internal("something went wrong", nil)
	if err.Status != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, err.Status)
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

// TestWithCode covers the builder method used for fine-grained 400 / 409
// error display on the client. The Status is preserved; only Code changes.
func TestWithCode(t *testing.T) {
	err := apperrors.BadRequest("email already registered").WithCode("DUPLICATE_EMAIL")
	if err.Status != http.StatusBadRequest {
		t.Errorf("Status = %d, want 400", err.Status)
	}
	if err.Code != "DUPLICATE_EMAIL" {
		t.Errorf("Code = %q, want DUPLICATE_EMAIL", err.Code)
	}
	if !errors.Is(err, apperrors.ErrBadRequest) {
		t.Error("WithCode must not break error wrapping")
	}
}

// TestAppError_MarshalJSON pins the wire format. Two properties matter:
//  1. The message is serialized under the legacy key `error` so existing
//     clients keep working.
//  2. `code` is omitted entirely when unset (so legacy 400/404/... bodies
//     stay `{"error": "..."}`), and present when WithCode was called.
func TestAppError_MarshalJSON(t *testing.T) {
	t.Run("no code", func(t *testing.T) {
		err := apperrors.BadRequest("bad input")
		b, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Fatalf("marshal: %v", marshalErr)
		}
		want := `{"error":"bad input"}`
		if string(b) != want {
			t.Errorf("json = %s, want %s", b, want)
		}
	})

	t.Run("with code", func(t *testing.T) {
		err := apperrors.BadRequest("email already registered").WithCode("DUPLICATE_EMAIL")
		b, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Fatalf("marshal: %v", marshalErr)
		}
		want := `{"code":"DUPLICATE_EMAIL","error":"email already registered"}`
		if string(b) != want {
			t.Errorf("json = %s, want %s", b, want)
		}
	})
}
