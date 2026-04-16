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
	"github.com/Riku-KANO/ec-test/services/catalog/internal/port"
)

// SKURepository handles persistence of SKUs.
type SKURepository struct {
	pool *pgxpool.Pool
}

// NewSKURepository creates a new SKURepository.
func NewSKURepository(pool *pgxpool.Pool) *SKURepository {
	return &SKURepository{pool: pool}
}

// Create inserts a new SKU.
func (r *SKURepository) Create(ctx context.Context, s *domain.SKU) error {
	s.ID = uuid.New()

	attrs := json.RawMessage("{}")
	if s.Attributes != nil {
		attrs = s.Attributes
	}

	return database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO catalog_svc.skus (id, product_id, seller_id, sku_code, price_amount, price_currency, attributes, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 RETURNING created_at, updated_at`,
			s.ID, s.ProductID, s.SellerID, s.SKUCode, s.PriceAmount, s.PriceCurrency, attrs, s.Status,
		).Scan(&s.CreatedAt, &s.UpdatedAt)
	})
}

// GetByID retrieves a SKU by its ID.
func (r *SKURepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.SKU, error) {
	var s domain.SKU
	var found bool

	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, product_id, seller_id, sku_code, price_amount, price_currency, attributes, status, created_at, updated_at
			 FROM catalog_svc.skus WHERE id = $1`,
			id,
		).Scan(&s.ID, &s.ProductID, &s.SellerID, &s.SKUCode, &s.PriceAmount, &s.PriceCurrency, &s.Attributes, &s.Status, &s.CreatedAt, &s.UpdatedAt)
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
func (r *SKURepository) List(ctx context.Context, productID uuid.UUID) ([]domain.SKU, error) {
	var skus []domain.SKU

	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, product_id, seller_id, sku_code, price_amount, price_currency, attributes, status, created_at, updated_at
			 FROM catalog_svc.skus WHERE product_id = $1
			 ORDER BY created_at ASC`,
			productID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var s domain.SKU
			if err := rows.Scan(&s.ID, &s.ProductID, &s.SellerID, &s.SKUCode, &s.PriceAmount, &s.PriceCurrency, &s.Attributes, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
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
func (r *SKURepository) Update(ctx context.Context, s *domain.SKU) error {
	attrs := json.RawMessage("{}")
	if s.Attributes != nil {
		attrs = s.Attributes
	}

	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE catalog_svc.skus
			 SET sku_code = $2, price_amount = $3, price_currency = $4, attributes = $5, updated_at = NOW()
			 WHERE id = $1`,
			s.ID, s.SKUCode, s.PriceAmount, s.PriceCurrency, attrs,
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
func (r *SKURepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ProductStatus) error {
	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE catalog_svc.skus SET status = $2, updated_at = NOW()
			 WHERE id = $1`,
			id, status,
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

// CheapestPrice returns the minimum SKU price for the product. Joins the
// caller's Tx via TxOrPool so publishers running inside the outbox Tx see
// the SKUs they just inserted.
func (r *SKURepository) CheapestPrice(ctx context.Context, productID uuid.UUID) (int64, string, bool, error) {
	var amount int64
	var currency string
	var found bool
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT price_amount, price_currency
			 FROM catalog_svc.skus
			 WHERE product_id = $1
			 ORDER BY price_amount ASC
			 LIMIT 1`,
			productID,
		).Scan(&amount, &currency)
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
		return 0, "", false, fmt.Errorf("cheapest sku price: %w", err)
	}
	return amount, currency, found, nil
}

// BatchGetMappings returns (id, product_id, seller_id) for each known SKU in
// ids. Unknown ids are silently omitted.
func (r *SKURepository) BatchGetMappings(ctx context.Context, ids []uuid.UUID) ([]port.SKUMapping, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, product_id, seller_id FROM catalog_svc.skus WHERE id = ANY($1)`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get sku mappings: %w", err)
	}
	defer rows.Close()
	out := make([]port.SKUMapping, 0, len(ids))
	for rows.Next() {
		var m port.SKUMapping
		if err := rows.Scan(&m.SKUID, &m.ProductID, &m.SellerID); err != nil {
			return nil, fmt.Errorf("scan sku mapping: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
