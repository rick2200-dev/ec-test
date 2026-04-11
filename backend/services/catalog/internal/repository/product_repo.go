package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
)

// ProductFilter holds optional filters for listing products.
type ProductFilter struct {
	TenantID   uuid.UUID
	SellerID   *uuid.UUID
	Status     *domain.ProductStatus
	CategoryID *uuid.UUID
}

// ProductRepository handles persistence of products.
type ProductRepository struct {
	pool *pgxpool.Pool
}

// NewProductRepository creates a new ProductRepository.
func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

// Create inserts a new product within a tenant-scoped transaction.
func (r *ProductRepository) Create(ctx context.Context, tenantID uuid.UUID, p *domain.Product) error {
	p.ID = uuid.New()
	p.TenantID = tenantID

	attrs := json.RawMessage("{}")
	if p.Attributes != nil {
		attrs = p.Attributes
	}

	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO catalog_svc.products (id, tenant_id, seller_id, category_id, name, slug, description, status, attributes)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 RETURNING created_at, updated_at`,
			p.ID, p.TenantID, p.SellerID, p.CategoryID, p.Name, p.Slug, p.Description, p.Status, attrs,
		).Scan(&p.CreatedAt, &p.UpdatedAt)
	})
}

// GetByID retrieves a product by its ID.
func (r *ProductRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	var p domain.Product
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, seller_id, category_id, name, slug, description, status, attributes, image_url, created_at, updated_at
			 FROM catalog_svc.products WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&p.ID, &p.TenantID, &p.SellerID, &p.CategoryID, &p.Name, &p.Slug, &p.Description, &p.Status, &p.Attributes, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get product by id: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &p, nil
}

// GetBySlug retrieves a product by its slug within a tenant.
func (r *ProductRepository) GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.Product, error) {
	var p domain.Product
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, seller_id, category_id, name, slug, description, status, attributes, image_url, created_at, updated_at
			 FROM catalog_svc.products WHERE slug = $1 AND tenant_id = $2`,
			slug, tenantID,
		).Scan(&p.ID, &p.TenantID, &p.SellerID, &p.CategoryID, &p.Name, &p.Slug, &p.Description, &p.Status, &p.Attributes, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get product by slug: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &p, nil
}

// List returns a filtered, paginated list of products.
func (r *ProductRepository) List(ctx context.Context, filter ProductFilter, limit, offset int) ([]domain.Product, int, error) {
	var products []domain.Product
	var total int

	err := database.TenantTx(ctx, r.pool, filter.TenantID, func(tx pgx.Tx) error {
		// Build WHERE clause dynamically.
		conditions := []string{"tenant_id = $1"}
		args := []any{filter.TenantID}
		argIdx := 2

		if filter.SellerID != nil {
			conditions = append(conditions, fmt.Sprintf("seller_id = $%d", argIdx))
			args = append(args, *filter.SellerID)
			argIdx++
		}
		if filter.Status != nil {
			conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
			args = append(args, *filter.Status)
			argIdx++
		}
		if filter.CategoryID != nil {
			conditions = append(conditions, fmt.Sprintf("category_id = $%d", argIdx))
			args = append(args, *filter.CategoryID)
			argIdx++
		}

		where := strings.Join(conditions, " AND ")

		// Count total.
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM catalog_svc.products WHERE %s", where)
		if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
			return fmt.Errorf("count products: %w", err)
		}

		// Fetch page.
		query := fmt.Sprintf(
			`SELECT id, tenant_id, seller_id, category_id, name, slug, description, status, attributes, image_url, created_at, updated_at
			 FROM catalog_svc.products WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
			where, argIdx, argIdx+1,
		)
		args = append(args, limit, offset)

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("list products: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var p domain.Product
			if err := rows.Scan(&p.ID, &p.TenantID, &p.SellerID, &p.CategoryID, &p.Name, &p.Slug, &p.Description, &p.Status, &p.Attributes, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt); err != nil {
				return fmt.Errorf("scan product: %w", err)
			}
			products = append(products, p)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

// Update modifies an existing product.
func (r *ProductRepository) Update(ctx context.Context, tenantID uuid.UUID, p *domain.Product) error {
	attrs := json.RawMessage("{}")
	if p.Attributes != nil {
		attrs = p.Attributes
	}

	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE catalog_svc.products
			 SET name = $3, slug = $4, description = $5, category_id = $6, attributes = $7, updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2`,
			p.ID, tenantID, p.Name, p.Slug, p.Description, p.CategoryID, attrs,
		)
		if err != nil {
			return fmt.Errorf("update product: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("product not found")
		}
		return nil
	})
}

// UpdateStatus changes the status of a product.
func (r *ProductRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.ProductStatus) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE catalog_svc.products SET status = $3, updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2`,
			id, tenantID, status,
		)
		if err != nil {
			return fmt.Errorf("update product status: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("product not found")
		}
		return nil
	})
}

// GetWithSKUs retrieves a product along with all its SKUs.
func (r *ProductRepository) GetWithSKUs(ctx context.Context, tenantID uuid.UUID, productID uuid.UUID) (*domain.ProductWithSKUs, error) {
	var result domain.ProductWithSKUs
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		// Fetch product.
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, seller_id, category_id, name, slug, description, status, attributes, image_url, created_at, updated_at
			 FROM catalog_svc.products WHERE id = $1 AND tenant_id = $2`,
			productID, tenantID,
		).Scan(&result.ID, &result.TenantID, &result.SellerID, &result.CategoryID, &result.Name, &result.Slug, &result.Description, &result.Status, &result.Attributes, &result.ImageURL, &result.CreatedAt, &result.UpdatedAt)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		found = true

		// Fetch SKUs.
		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, product_id, seller_id, sku_code, price_amount, price_currency, attributes, status, created_at, updated_at
			 FROM catalog_svc.skus WHERE product_id = $1 AND tenant_id = $2
			 ORDER BY created_at ASC`,
			productID, tenantID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var s domain.SKU
			if err := rows.Scan(&s.ID, &s.TenantID, &s.ProductID, &s.SellerID, &s.SKUCode, &s.PriceAmount, &s.PriceCurrency, &s.Attributes, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
				return err
			}
			result.SKUs = append(result.SKUs, s)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("get product with skus: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &result, nil
}

// GetWithSKUsBySlug retrieves a product along with all its SKUs by slug.
func (r *ProductRepository) GetWithSKUsBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.ProductWithSKUs, error) {
	var result domain.ProductWithSKUs
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, seller_id, category_id, name, slug, description, status, attributes, image_url, created_at, updated_at
			 FROM catalog_svc.products WHERE slug = $1 AND tenant_id = $2`,
			slug, tenantID,
		).Scan(&result.ID, &result.TenantID, &result.SellerID, &result.CategoryID, &result.Name, &result.Slug, &result.Description, &result.Status, &result.Attributes, &result.ImageURL, &result.CreatedAt, &result.UpdatedAt)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		found = true

		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, product_id, seller_id, sku_code, price_amount, price_currency, attributes, status, created_at, updated_at
			 FROM catalog_svc.skus WHERE product_id = $1 AND tenant_id = $2
			 ORDER BY created_at ASC`,
			result.ID, tenantID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var s domain.SKU
			if err := rows.Scan(&s.ID, &s.TenantID, &s.ProductID, &s.SellerID, &s.SKUCode, &s.PriceAmount, &s.PriceCurrency, &s.Attributes, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
				return err
			}
			result.SKUs = append(result.SKUs, s)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("get product with skus by slug: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &result, nil
}
