package httputil

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
)

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// Error writes an error JSON response.
func Error(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		JSON(w, appErr.Code, map[string]string{"error": appErr.Message})
		return
	}
	JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

// Decode reads and decodes a JSON request body into the target struct.
func Decode(r *http.Request, target any) error {
	if r.Body == nil {
		return apperrors.BadRequest("empty request body")
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return apperrors.BadRequest("invalid request body: " + err.Error())
	}
	return nil
}
