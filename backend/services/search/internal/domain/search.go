package domain

import "github.com/google/uuid"

// SearchRequest represents a product search query.
type SearchRequest struct {
	Query      string    `json:"q"`
	TenantID   uuid.UUID `json:"tenant_id"`
	SellerID   *uuid.UUID `json:"seller_id,omitempty"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
	MinPrice   *float64  `json:"min_price,omitempty"`
	MaxPrice   *float64  `json:"max_price,omitempty"`
	Status     string    `json:"status,omitempty"`
	SortBy     string    `json:"sort_by,omitempty"`
	SortOrder  string    `json:"sort_order,omitempty"`
	Limit      int       `json:"limit"`
	Offset     int       `json:"offset"`
}

// SearchResult holds the response from a search query.
type SearchResult struct {
	Products         []ProductHit `json:"products"`
	PromotedProducts []ProductHit `json:"promoted_products,omitempty"`
	Total            int          `json:"total"`
	Facets           []Facet      `json:"facets"`
}

// ProductHit represents a single product in search results.
type ProductHit struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	SellerID      uuid.UUID `json:"seller_id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Description   string    `json:"description"`
	Status        string    `json:"status"`
	PriceAmount   float64   `json:"price_amount"`
	PriceCurrency string    `json:"price_currency"`
	SellerName    string    `json:"seller_name"`
	CategoryName  string    `json:"category_name"`
	Score         float64   `json:"score"`
	IsPromoted    bool      `json:"is_promoted"`
	PlanTier      int       `json:"plan_tier"`
}

// Facet represents a facet grouping in search results.
type Facet struct {
	Field  string       `json:"field"`
	Values []FacetValue `json:"values"`
}

// FacetValue represents a single value within a facet.
type FacetValue struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// ProductEvent represents a product event received from Pub/Sub.
type ProductEvent struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	SellerID    uuid.UUID `json:"seller_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
}
