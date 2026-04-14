package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SellerStatus represents the lifecycle state of a seller.
type SellerStatus string

const (
	SellerStatusPending   SellerStatus = "pending"
	SellerStatusApproved  SellerStatus = "approved"
	SellerStatusRejected  SellerStatus = "rejected"
	SellerStatusSuspended SellerStatus = "suspended"
)

// Seller represents a seller on the marketplace.
type Seller struct {
	ID                uuid.UUID       `json:"id"`
	Auth0OrgID        string          `json:"auth0_org_id"`
	Name              string          `json:"name"`
	Slug              string          `json:"slug"`
	Status            SellerStatus    `json:"status"`
	StripeAccountID   string          `json:"stripe_account_id,omitempty"`
	CommissionRateBPS int             `json:"commission_rate_bps"`
	Settings          json.RawMessage `json:"settings,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// SellerUserRole represents the role of a user within a seller organization.
type SellerUserRole string

const (
	SellerUserRoleOwner  SellerUserRole = "owner"
	SellerUserRoleAdmin  SellerUserRole = "admin"
	SellerUserRoleMember SellerUserRole = "member"
)

// SellerUser represents a user belonging to a seller organization.
type SellerUser struct {
	ID          uuid.UUID      `json:"id"`
	SellerID    uuid.UUID      `json:"seller_id"`
	Auth0UserID string         `json:"auth0_user_id"`
	Role        SellerUserRole `json:"role"`
	CreatedAt   time.Time      `json:"created_at"`
}
