package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
)

// SKURepository handles persistence of SKUs.
type SKURepository struct {
	pool *pgxpool.Pool
}

// NewSKURepository creates a new SKURepository.
func NewSKURepository(pool *pgxpool.Pool) *SKURepository {
	return &SKURepository{pool: pool}
}

// Create inserts a new SKU within a tenant-scoped transaction.
func (r *SKURepository) Create(ctx context.Context, tenantID uuid.UUID, s *domain.SKU) error {
	s.ID = uuid.New()
	s.TenantID = tenantID

	attrs := json.RawMessage("{}")
	if s.Attributes != nil {
		attrs = s.Attributes
	}

	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO catalog_svc.skus (id, tenant_id, product_id, seller_id, sku_code, price_amount, price_currency, attributes, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 RETURNING created_at, updated_at`,
			s.ID, s.TenantID, s.ProductID, s.SellerID, s.SKUCode, s.PriceAmount, s.PriceCurrency, attrs, s.Status,
		).Scan(&s.CreatedAt, &s.UpdatedAt)
	})
}

// GetByID retrieves a SKU by its ID.
func (r *SKURepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SKU, error) {
	var s domain.SKU
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, product_id, seller_id, sku_code, price_amount, price_currency, attributes, status, created_at, updated_at
			 FROM catalog_svc.skus WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&s.ID, &s.TenantID, &s.ProductID, &s.SellerID, &s.SKUCode, &s.PriceAmount, &s.PriceCurrency, &s.Attributes, &s.Status, &s.CreatedAt, &s.UpdatedAt)
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
		return nil, fmt.Errorf("get sku by id: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &s, nil
}

// List returns all SKUs for a given product.
func (r *SKURepository) List(ctx context.Context, tenantID, productID uuid.UUID) ([]domain.SKU, error) {
	var skus []domain.SKU

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
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
			skus = append(skus, s)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("list skus: %w", err)
	}
	return skus, nil
}

// Update modifies an existing SKU.
func (r *SKURepository) Update(ctx context.Context, tenantID uuid.UUID, s *domain.SKU) error {
	attrs := json.RawMessage("{}")
	if s.Attributes != nil {
		attrs = s.Attributes
	}

	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE catalog_svc.skus
			 SET sku_code = $3, price_amount = $4, price_currency = $5, attributes = $6, updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2`,
			s.ID, tenantID, s.SKUCode, s.PriceAmount, s.PriceCurrency, attrs,
		)
		if err != nil {
			return fmt.Errorf("update sku: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("sku not found")
		}
		return nil
	})
}

// UpdateStatus changes the status of a SKU.
func (r *SKURepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.ProductStatus) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE catalog_svc.skus SET status = $3, updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2`,
			id, tenantID, status,
		)
		if err != nil {
			return fmt.Errorf("update sku status: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("sku not found")
		}
		return nil
	})
}
