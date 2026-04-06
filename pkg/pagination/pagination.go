package pagination

import (
	"net/http"
	"strconv"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Params holds pagination parameters.
type Params struct {
	Limit  int
	Offset int
}

// FromRequest extracts pagination parameters from an HTTP request.
func FromRequest(r *http.Request) Params {
	p := Params{Limit: DefaultLimit, Offset: 0}

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			p.Limit = v
		}
	}
	if p.Limit > MaxLimit {
		p.Limit = MaxLimit
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			p.Offset = v
		}
	}

	return p
}

// Response is a generic paginated response.
type Response[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
