package domain

import (
	"time"

	"github.com/google/uuid"
)

// Buyer is a minimal profile record for a buyer-side end user. Identity is
// owned by Auth0 (Auth0UserID is the `sub` claim); this row exists so that
// profile fields (email, display name) are queryable locally without hitting
// the Auth0 Management API on every request.
//
// The row is upserted on the buyer's first successful login by the Next.js
// BFF (`onCallback`) via POST /internal/buyers/upsert. Downstream services
// continue to key off the Auth0 sub — the Buyer UUID here is not used as a
// foreign key elsewhere.
type Buyer struct {
	ID          uuid.UUID `json:"id"`
	Auth0Sub    string    `json:"auth0_sub"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
