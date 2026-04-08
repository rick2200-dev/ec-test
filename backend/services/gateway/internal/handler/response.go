package handler

import "net/http"

// writeRaw writes raw bytes (typically JSON from a downstream service)
// to the response with the given status code.
func writeRaw(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(body)
}
