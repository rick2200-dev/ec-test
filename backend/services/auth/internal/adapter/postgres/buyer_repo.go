package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// BuyerRepository persists minimal buyer profile records (see
// domain.Buyer). Rows are upserted on first Auth0 login; identity is owned
// by Auth0 and the buyer UUID here is not referenced by other services.
type BuyerRepository struct {
	pool *pgxpool.Pool
}

// NewBuyerRepository creates a new BuyerRepository.
func NewBuyerRepository(pool *pgxpool.Pool) *BuyerRepository {
	return &BuyerRepository{pool: pool}
}

// Upsert inserts or updates a buyer row keyed by auth0_sub.
// On update the email and display_name are refreshed; created_at is
// preserved. The resulting row (with id/created_at/updated_at populated)
// is written back into b.
func (r *BuyerRepository) Upsert(ctx context.Context, b *domain.Buyer) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}

	var displayName sql.NullString
	if b.DisplayName != "" {
		displayName = sql.NullString{String: b.DisplayName, Valid: true}
	}

	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		var nullableDisplay sql.NullString
		err := tx.QueryRow(ctx,
			`INSERT INTO auth_svc.buyers
			 (id, auth0_sub, email, display_name)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (auth0_sub)
			 DO UPDATE SET email = EXCLUDED.email,
			               display_name = EXCLUDED.display_name,
			               updated_at = NOW()
			 RETURNING id, email, display_name, created_at, updated_at`,
			b.ID, b.Auth0Sub, b.Email, displayName,
		).Scan(&b.ID, &b.Email, &nullableDisplay, &b.CreatedAt, &b.UpdatedAt)
		if err != nil {
			return fmt.Errorf("upsert buyer: %w", err)
		}
		if nullableDisplay.Valid {
			b.DisplayName = nullableDisplay.String
		} else {
			b.DisplayName = ""
		}
		return nil
	})
}
