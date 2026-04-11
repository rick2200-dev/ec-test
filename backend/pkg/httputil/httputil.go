package httputil

import (
	"encoding/json"
	"log/slog"
	"net/http"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
)

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		// Once WriteHeader has been called we cannot change the status,
		// so an Encode error here is logged-and-swallowed. The most
		// likely cause is a client disconnect mid-response.
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Warn("failed to encode JSON response", "error", err)
		}
	}
}

// Error writes an error JSON response.
//
// When err is an *apperrors.AppError, the full struct is marshaled so any
// application-defined Code (e.g. "DUPLICATE_EMAIL") reaches the client as
// `{"error": "...", "code": "..."}`. Callers that did not set a code get
// the legacy `{"error": "..."}` shape because Code is `omitempty`.
//
// Unwrapped (non-AppError) errors collapse to a generic 500 so internal
// details never leak to the caller — callers who need a specific status
// must produce an AppError.
func Error(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		JSON(w, appErr.Status, appErr)
		return
	}
	JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

// Decode reads and decodes a JSON request body into the target struct.
func Decode(r *http.Request, target any) error {
	if r.Body == nil {
		return apperrors.BadRequest("empty request body")
	}
	defer func() { _ = r.Body.Close() }()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return apperrors.BadRequest("invalid request body: " + err.Error())
	}
	return nil
}
