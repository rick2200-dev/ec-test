package repository

import "github.com/google/uuid"

// parseUUID wraps uuid.Parse so scanners can handle nullable columns
// consistently across repos without open-coding the error message.
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
