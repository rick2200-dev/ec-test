package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
)

// CategoryRepository handles persistence of categories.
type CategoryRepository struct {
	pool *pgxpool.Pool
}

// NewCategoryRepository creates a new CategoryRepository.
func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

// Create inserts a new category within a tenant-scoped transaction.
func (r *CategoryRepository) Create(ctx context.Context, tenantID uuid.UUID, c *domain.Category) error {
	c.ID = uuid.New()
	c.TenantID = tenantID

	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO catalog_svc.categories (id, tenant_id, parent_id, name, slug, sort_order)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING created_at`,
			c.ID, c.TenantID, c.ParentID, c.Name, c.Slug, c.SortOrder,
		).Scan(&c.CreatedAt)
	})
}

// GetByID retrieves a category by its ID within a tenant.
func (r *CategoryRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Category, error) {
	var c domain.Category
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, parent_id, name, slug, sort_order, created_at
			 FROM catalog_svc.categories WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&c.ID, &c.TenantID, &c.ParentID, &c.Name, &c.Slug, &c.SortOrder, &c.CreatedAt)
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
		return nil, fmt.Errorf("get category by id: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &c, nil
}

// GetBySlug retrieves a category by its slug within a tenant.
func (r *CategoryRepository) GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.Category, error) {
	var c domain.Category
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, parent_id, name, slug, sort_order, created_at
			 FROM catalog_svc.categories WHERE slug = $1 AND tenant_id = $2`,
			slug, tenantID,
		).Scan(&c.ID, &c.TenantID, &c.ParentID, &c.Name, &c.Slug, &c.SortOrder, &c.CreatedAt)
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
		return nil, fmt.Errorf("get category by slug: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &c, nil
}

// List returns all categories for a tenant.
func (r *CategoryRepository) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Category, error) {
	var categories []domain.Category

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, parent_id, name, slug, sort_order, created_at
			 FROM catalog_svc.categories WHERE tenant_id = $1
			 ORDER BY sort_order ASC, name ASC`,
			tenantID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var c domain.Category
			if err := rows.Scan(&c.ID, &c.TenantID, &c.ParentID, &c.Name, &c.Slug, &c.SortOrder, &c.CreatedAt); err != nil {
				return err
			}
			categories = append(categories, c)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return categories, nil
}

// Update modifies an existing category.
func (r *CategoryRepository) Update(ctx context.Context, tenantID uuid.UUID, c *domain.Category) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE catalog_svc.categories SET parent_id = $3, name = $4, slug = $5, sort_order = $6
			 WHERE id = $1 AND tenant_id = $2`,
			c.ID, tenantID, c.ParentID, c.Name, c.Slug, c.SortOrder,
		)
		if err != nil {
			return fmt.Errorf("update category: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("category not found")
		}
		return nil
	})
}

// Delete removes a category.
func (r *CategoryRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`DELETE FROM catalog_svc.categories WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		)
		if err != nil {
			return fmt.Errorf("delete category: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("category not found")
		}
		return nil
	})
}
