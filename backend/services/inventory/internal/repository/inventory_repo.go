package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/domain"
)

// InventoryRepository handles persistence of inventory records.
type InventoryRepository struct {
	pool *pgxpool.Pool
}

// NewInventoryRepository creates a new InventoryRepository.
func NewInventoryRepository(pool *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{pool: pool}
}

// GetBySKUID retrieves inventory for a specific SKU within a tenant.
func (r *InventoryRepository) GetBySKUID(ctx context.Context, tenantID, skuID uuid.UUID) (*domain.Inventory, error) {
	var inv domain.Inventory
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`SELECT id, tenant_id, sku_id, seller_id, quantity_available, quantity_reserved, low_stock_threshold, updated_at
			 FROM inventory_svc.inventory
			 WHERE tenant_id = $1 AND sku_id = $2`, tenantID, skuID)

		err := row.Scan(&inv.ID, &inv.TenantID, &inv.SKUID, &inv.SellerID,
			&inv.QuantityAvailable, &inv.QuantityReserved, &inv.LowStockThreshold, &inv.UpdatedAt)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return fmt.Errorf("get inventory by sku_id: %w", err)
		}
		found = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &inv, nil
}

// List returns a paginated list of inventory records for a seller within a tenant.
func (r *InventoryRepository) List(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Inventory, int, error) {
	var items []domain.Inventory
	var total int

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		// Count total.
		err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM inventory_svc.inventory WHERE tenant_id = $1 AND seller_id = $2`,
			tenantID, sellerID,
		).Scan(&total)
		if err != nil {
			return fmt.Errorf("count inventory: %w", err)
		}

		// Fetch page.
		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, sku_id, seller_id, quantity_available, quantity_reserved, low_stock_threshold, updated_at
			 FROM inventory_svc.inventory
			 WHERE tenant_id = $1 AND seller_id = $2
			 ORDER BY updated_at DESC
			 LIMIT $3 OFFSET $4`,
			tenantID, sellerID, limit, offset,
		)
		if err != nil {
			return fmt.Errorf("list inventory: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var inv domain.Inventory
			if err := rows.Scan(&inv.ID, &inv.TenantID, &inv.SKUID, &inv.SellerID,
				&inv.QuantityAvailable, &inv.QuantityReserved, &inv.LowStockThreshold, &inv.UpdatedAt); err != nil {
				return fmt.Errorf("scan inventory: %w", err)
			}
			items = append(items, inv)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Upsert creates or updates an inventory record (INSERT ... ON CONFLICT).
func (r *InventoryRepository) Upsert(ctx context.Context, tenantID uuid.UUID, inv *domain.Inventory) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		inv.TenantID = tenantID
		if inv.ID == uuid.Nil {
			inv.ID = uuid.New()
		}

		err := tx.QueryRow(ctx,
			`INSERT INTO inventory_svc.inventory (id, tenant_id, sku_id, seller_id, quantity_available, quantity_reserved, low_stock_threshold)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (tenant_id, sku_id)
			 DO UPDATE SET
			   quantity_available = EXCLUDED.quantity_available,
			   low_stock_threshold = EXCLUDED.low_stock_threshold,
			   updated_at = NOW()
			 RETURNING id, updated_at`,
			inv.ID, tenantID, inv.SKUID, inv.SellerID,
			inv.QuantityAvailable, inv.QuantityReserved, inv.LowStockThreshold,
		).Scan(&inv.ID, &inv.UpdatedAt)
		if err != nil {
			return fmt.Errorf("upsert inventory: %w", err)
		}
		return nil
	})
}

// Reserve atomically reserves stock for a SKU. Uses SELECT FOR UPDATE to prevent race conditions.
// Returns an error if available quantity is insufficient.
func (r *InventoryRepository) Reserve(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		var available int
		err := tx.QueryRow(ctx,
			`SELECT quantity_available FROM inventory_svc.inventory
			 WHERE tenant_id = $1 AND sku_id = $2
			 FOR UPDATE`,
			tenantID, skuID,
		).Scan(&available)
		if err == pgx.ErrNoRows {
			return fmt.Errorf("inventory not found for sku %s", skuID)
		}
		if err != nil {
			return fmt.Errorf("lock inventory row: %w", err)
		}

		if available < quantity {
			return fmt.Errorf("insufficient stock: available=%d, requested=%d", available, quantity)
		}

		_, err = tx.Exec(ctx,
			`UPDATE inventory_svc.inventory
			 SET quantity_available = quantity_available - $3,
			     quantity_reserved = quantity_reserved + $3,
			     updated_at = NOW()
			 WHERE tenant_id = $1 AND sku_id = $2`,
			tenantID, skuID, quantity,
		)
		if err != nil {
			return fmt.Errorf("reserve stock: %w", err)
		}
		return nil
	})
}

// Release returns reserved stock back to available.
func (r *InventoryRepository) Release(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		var reserved int
		err := tx.QueryRow(ctx,
			`SELECT quantity_reserved FROM inventory_svc.inventory
			 WHERE tenant_id = $1 AND sku_id = $2
			 FOR UPDATE`,
			tenantID, skuID,
		).Scan(&reserved)
		if err == pgx.ErrNoRows {
			return fmt.Errorf("inventory not found for sku %s", skuID)
		}
		if err != nil {
			return fmt.Errorf("lock inventory row: %w", err)
		}

		if reserved < quantity {
			return fmt.Errorf("insufficient reserved stock: reserved=%d, requested=%d", reserved, quantity)
		}

		_, err = tx.Exec(ctx,
			`UPDATE inventory_svc.inventory
			 SET quantity_available = quantity_available + $3,
			     quantity_reserved = quantity_reserved - $3,
			     updated_at = NOW()
			 WHERE tenant_id = $1 AND sku_id = $2`,
			tenantID, skuID, quantity,
		)
		if err != nil {
			return fmt.Errorf("release stock: %w", err)
		}
		return nil
	})
}

// ConfirmSold removes quantity from reserved (stock has been sold and shipped).
func (r *InventoryRepository) ConfirmSold(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		var reserved int
		err := tx.QueryRow(ctx,
			`SELECT quantity_reserved FROM inventory_svc.inventory
			 WHERE tenant_id = $1 AND sku_id = $2
			 FOR UPDATE`,
			tenantID, skuID,
		).Scan(&reserved)
		if err == pgx.ErrNoRows {
			return fmt.Errorf("inventory not found for sku %s", skuID)
		}
		if err != nil {
			return fmt.Errorf("lock inventory row: %w", err)
		}

		if reserved < quantity {
			return fmt.Errorf("insufficient reserved stock: reserved=%d, requested=%d", reserved, quantity)
		}

		_, err = tx.Exec(ctx,
			`UPDATE inventory_svc.inventory
			 SET quantity_reserved = quantity_reserved - $3,
			     updated_at = NOW()
			 WHERE tenant_id = $1 AND sku_id = $2`,
			tenantID, skuID, quantity,
		)
		if err != nil {
			return fmt.Errorf("confirm sold: %w", err)
		}
		return nil
	})
}

// AdjustStock performs a manual stock adjustment (positive or negative).
func (r *InventoryRepository) AdjustStock(ctx context.Context, tenantID, skuID uuid.UUID, quantity int) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE inventory_svc.inventory
			 SET quantity_available = quantity_available + $3,
			     updated_at = NOW()
			 WHERE tenant_id = $1 AND sku_id = $2`,
			tenantID, skuID, quantity,
		)
		if err != nil {
			return fmt.Errorf("adjust stock: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("inventory not found for sku %s", skuID)
		}
		return nil
	})
}

// RecordMovement inserts a stock movement record for auditing.
func (r *InventoryRepository) RecordMovement(ctx context.Context, tenantID uuid.UUID, m *domain.StockMovement) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		m.ID = uuid.New()
		m.TenantID = tenantID

		err := tx.QueryRow(ctx,
			`INSERT INTO inventory_svc.stock_movements (id, tenant_id, sku_id, movement_type, quantity, reference_type, reference_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING created_at`,
			m.ID, tenantID, m.SKUID, m.MovementType, m.Quantity, m.ReferenceType, m.ReferenceID,
		).Scan(&m.CreatedAt)
		if err != nil {
			return fmt.Errorf("record movement: %w", err)
		}
		return nil
	})
}
