package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/services/search/internal/domain"
)

// PostgresEngine implements SearchEngine using PostgreSQL full-text search.
type PostgresEngine struct {
	pool *pgxpool.Pool
}

// NewPostgresEngine creates a new PostgresEngine.
func NewPostgresEngine(pool *pgxpool.Pool) *PostgresEngine {
	return &PostgresEngine{pool: pool}
}

// Search executes a full-text search query against catalog_svc.products.
func (e *PostgresEngine) Search(ctx context.Context, req domain.SearchRequest) (*domain.SearchResult, error) {
	var (
		conditions []string
		args       []any
		argIdx     int
	)

	nextArg := func(val any) string {
		argIdx++
		args = append(args, val)
		return fmt.Sprintf("$%d", argIdx)
	}

	// Always filter by tenant
	conditions = append(conditions, fmt.Sprintf("p.tenant_id = %s", nextArg(req.TenantID)))

	// Full-text search condition
	hasQuery := strings.TrimSpace(req.Query) != ""
	if hasQuery {
		// Convert the user query into a tsquery with prefix matching
		terms := strings.Fields(req.Query)
		tsTerms := make([]string, len(terms))
		for i, t := range terms {
			tsTerms[i] = t + ":*"
		}
		tsQuery := strings.Join(tsTerms, " & ")
		conditions = append(conditions, fmt.Sprintf("p.search_vector @@ to_tsquery('simple', %s)", nextArg(tsQuery)))
	}

	// Optional filters
	if req.SellerID != nil {
		conditions = append(conditions, fmt.Sprintf("p.seller_id = %s", nextArg(*req.SellerID)))
	}
	if req.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("p.category_id = %s", nextArg(*req.CategoryID)))
	}
	if req.Status != "" {
		conditions = append(conditions, fmt.Sprintf("p.status = %s", nextArg(req.Status)))
	}
	if req.MinPrice != nil {
		conditions = append(conditions, fmt.Sprintf("s.price_amount >= %s", nextArg(*req.MinPrice)))
	}
	if req.MaxPrice != nil {
		conditions = append(conditions, fmt.Sprintf("s.price_amount <= %s", nextArg(*req.MaxPrice)))
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Build rank expression
	rankExpr := "0"
	if hasQuery {
		rankExpr = fmt.Sprintf("ts_rank(p.search_vector, to_tsquery('simple', $2))")
	}

	// Sort
	orderClause := buildOrderClause(req.SortBy, req.SortOrder, hasQuery)

	// Limit / offset
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	limitArg := nextArg(req.Limit)
	offsetArg := nextArg(req.Offset)

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT p.id)
		FROM catalog_svc.products p
		LEFT JOIN catalog_svc.skus s ON s.product_id = p.id AND s.tenant_id = p.tenant_id
		%s
	`, whereClause)

	var total int
	if err := e.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count search results: %w", err)
	}

	// Main search query
	searchQuery := fmt.Sprintf(`
		SELECT DISTINCT ON (p.id)
			p.id, p.tenant_id, p.seller_id, p.name, p.slug, COALESCE(p.description, ''),
			p.status,
			COALESCE(s.price_amount, 0), COALESCE(s.price_currency, 'JPY'),
			COALESCE(sel.company_name, ''), COALESCE(c.name, ''),
			%s AS rank
		FROM catalog_svc.products p
		LEFT JOIN catalog_svc.skus s ON s.product_id = p.id AND s.tenant_id = p.tenant_id
		LEFT JOIN auth_svc.sellers sel ON sel.id = p.seller_id AND sel.tenant_id = p.tenant_id
		LEFT JOIN catalog_svc.categories c ON c.id = p.category_id AND c.tenant_id = p.tenant_id
		%s
		%s
		LIMIT %s OFFSET %s
	`, rankExpr, whereClause, orderClause, limitArg, offsetArg)

	rows, err := e.pool.Query(ctx, searchQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("execute search query: %w", err)
	}
	defer rows.Close()

	var products []domain.ProductHit
	for rows.Next() {
		var h domain.ProductHit
		if err := rows.Scan(
			&h.ID, &h.TenantID, &h.SellerID, &h.Name, &h.Slug, &h.Description,
			&h.Status,
			&h.PriceAmount, &h.PriceCurrency,
			&h.SellerName, &h.CategoryName,
			&h.Score,
		); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		products = append(products, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}

	// Facets: categories
	facets, err := e.buildFacets(ctx, args[:1], req.TenantID) // reuse first arg (tenant_id)
	if err != nil {
		return nil, err
	}

	return &domain.SearchResult{
		Products: products,
		Total:    total,
		Facets:   facets,
	}, nil
}

// buildFacets generates category and price range facets for the tenant.
func (e *PostgresEngine) buildFacets(ctx context.Context, _ []any, tenantID uuid.UUID) ([]domain.Facet, error) {
	var facets []domain.Facet

	// Category facet
	catRows, err := e.pool.Query(ctx, `
		SELECT c.name, COUNT(p.id)
		FROM catalog_svc.products p
		JOIN catalog_svc.categories c ON c.id = p.category_id AND c.tenant_id = p.tenant_id
		WHERE p.tenant_id = $1 AND p.status = 'active'
		GROUP BY c.name
		ORDER BY COUNT(p.id) DESC
		LIMIT 20
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query category facets: %w", err)
	}
	defer catRows.Close()

	var catValues []domain.FacetValue
	for catRows.Next() {
		var fv domain.FacetValue
		if err := catRows.Scan(&fv.Value, &fv.Count); err != nil {
			return nil, fmt.Errorf("scan category facet: %w", err)
		}
		catValues = append(catValues, fv)
	}
	if len(catValues) > 0 {
		facets = append(facets, domain.Facet{Field: "category", Values: catValues})
	}

	// Price range facet
	priceRows, err := e.pool.Query(ctx, `
		SELECT
			CASE
				WHEN s.price_amount < 1000 THEN 'under_1000'
				WHEN s.price_amount < 5000 THEN '1000_5000'
				WHEN s.price_amount < 10000 THEN '5000_10000'
				WHEN s.price_amount < 50000 THEN '10000_50000'
				ELSE '50000_plus'
			END AS price_range,
			COUNT(DISTINCT p.id)
		FROM catalog_svc.products p
		JOIN catalog_svc.skus s ON s.product_id = p.id AND s.tenant_id = p.tenant_id
		WHERE p.tenant_id = $1 AND p.status = 'active'
		GROUP BY price_range
		ORDER BY price_range
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query price facets: %w", err)
	}
	defer priceRows.Close()

	var priceValues []domain.FacetValue
	for priceRows.Next() {
		var fv domain.FacetValue
		if err := priceRows.Scan(&fv.Value, &fv.Count); err != nil {
			return nil, fmt.Errorf("scan price facet: %w", err)
		}
		priceValues = append(priceValues, fv)
	}
	if len(priceValues) > 0 {
		facets = append(facets, domain.Facet{Field: "price_range", Values: priceValues})
	}

	return facets, nil
}

// IndexProduct is a no-op for PostgreSQL since the trigger handles indexing.
func (e *PostgresEngine) IndexProduct(_ context.Context, _ domain.ProductEvent) error {
	// The PostgreSQL trigger trg_products_search_vector automatically updates
	// the search_vector column on INSERT/UPDATE, so no explicit indexing is needed.
	return nil
}

// DeleteProduct is a no-op for PostgreSQL since deleting the product row
// automatically removes it from the search index.
func (e *PostgresEngine) DeleteProduct(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func buildOrderClause(sortBy, sortOrder string, hasQuery bool) string {
	direction := "ASC"
	if strings.EqualFold(sortOrder, "desc") {
		direction = "DESC"
	}

	switch strings.ToLower(sortBy) {
	case "price":
		return fmt.Sprintf("ORDER BY s.price_amount %s, p.name ASC", direction)
	case "name":
		return fmt.Sprintf("ORDER BY p.name %s", direction)
	case "created_at":
		return fmt.Sprintf("ORDER BY p.created_at %s", direction)
	default:
		if hasQuery {
			return "ORDER BY rank DESC, p.name ASC"
		}
		return "ORDER BY p.created_at DESC"
	}
}
