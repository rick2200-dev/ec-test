// Package repository contains the coupon service's postgres adapters.
// All repo methods honor an outer transaction on the context (via
// database.TxOrPool) so the app layer can batch coupon updates +
// reservation inserts atomically.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/domain"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/port"
)

// CouponRepository is the postgres implementation of CouponStore.
type CouponRepository struct {
	pool *pgxpool.Pool
}

func NewCouponRepository(pool *pgxpool.Pool) *CouponRepository {
	return &CouponRepository{pool: pool}
}

func (r *CouponRepository) Insert(ctx context.Context, c *domain.Coupon) error {
	return database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO coupon_svc.coupons
			   (id, code, issuer_type, issuer_id, discount_type,
			    discount_percent_bps, discount_amount, currency,
			    min_order_amount, max_discount_amount,
			    valid_from, expires_at,
			    usage_limit_total, usage_limit_per_user,
			    status, title, description)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
			 RETURNING created_at, updated_at, usage_count`,
			c.ID, c.Code, string(c.IssuerType), c.IssuerID, string(c.DiscountType),
			c.DiscountPercentBps, c.DiscountAmount, c.Currency,
			c.MinOrderAmount, c.MaxDiscountAmount,
			c.ValidFrom, c.ExpiresAt,
			c.UsageLimitTotal, c.UsageLimitPerUser,
			string(c.Status), c.Title, c.Description,
		).Scan(&c.CreatedAt, &c.UpdatedAt, &c.UsageCount)
	})
}

func (r *CouponRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Coupon, error) {
	return r.queryOne(ctx,
		`SELECT id, code, issuer_type, issuer_id, discount_type,
		        discount_percent_bps, discount_amount, currency,
		        min_order_amount, max_discount_amount,
		        valid_from, expires_at,
		        usage_limit_total, usage_limit_per_user, usage_count,
		        status, title, description, created_at, updated_at
		 FROM coupon_svc.coupons WHERE id = $1`,
		id,
	)
}

// GetByCodeForUpdate locks the row so the concurrent Reserve path
// can't race on usage_count. The WHERE includes an UPPER() match so
// callers don't have to case-normalize the code themselves.
func (r *CouponRepository) GetByCodeForUpdate(ctx context.Context, issuerType domain.IssuerType, issuerID *uuid.UUID, code string) (*domain.Coupon, error) {
	return r.queryOne(ctx,
		`SELECT id, code, issuer_type, issuer_id, discount_type,
		        discount_percent_bps, discount_amount, currency,
		        min_order_amount, max_discount_amount,
		        valid_from, expires_at,
		        usage_limit_total, usage_limit_per_user, usage_count,
		        status, title, description, created_at, updated_at
		 FROM coupon_svc.coupons
		 WHERE issuer_type = $1
		   AND COALESCE(issuer_id, '00000000-0000-0000-0000-000000000000'::uuid) = COALESCE($2, '00000000-0000-0000-0000-000000000000'::uuid)
		   AND UPPER(code) = UPPER($3)
		 FOR UPDATE`,
		string(issuerType), issuerID, code,
	)
}

func (r *CouponRepository) List(ctx context.Context, filter port.ListCouponsFilter) ([]domain.Coupon, int, error) {
	var (
		coupons []domain.Coupon
		total   int
	)
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		// COUNT first so the admin UI can page without extra round-trips.
		countSQL := `SELECT COUNT(*) FROM coupon_svc.coupons`
		listSQL := `SELECT id, code, issuer_type, issuer_id, discount_type,
		                   discount_percent_bps, discount_amount, currency,
		                   min_order_amount, max_discount_amount,
		                   valid_from, expires_at,
		                   usage_limit_total, usage_limit_per_user, usage_count,
		                   status, title, description, created_at, updated_at
		            FROM coupon_svc.coupons`
		where := ""
		args := []any{}
		if filter.Status != "" {
			args = append(args, filter.Status)
			if where == "" {
				where = ` WHERE status = $` + itoa(len(args))
			} else {
				where += ` AND status = $` + itoa(len(args))
			}
		}
		if filter.IssuerType != "" {
			args = append(args, string(filter.IssuerType))
			if where == "" {
				where = ` WHERE issuer_type = $` + itoa(len(args))
			} else {
				where += ` AND issuer_type = $` + itoa(len(args))
			}
		}
		if filter.IssuerID != nil {
			args = append(args, *filter.IssuerID)
			if where == "" {
				where = ` WHERE issuer_id = $` + itoa(len(args))
			} else {
				where += ` AND issuer_id = $` + itoa(len(args))
			}
		}
		countSQL += where
		listSQL += where
		if err := tx.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
			return fmt.Errorf("count coupons: %w", err)
		}
		if total == 0 {
			return nil
		}
		listSQL += ` ORDER BY created_at DESC LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)
		args = append(args, filter.Limit, filter.Offset)

		rows, err := tx.Query(ctx, listSQL, args...)
		if err != nil {
			return fmt.Errorf("query coupons: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			c, err := scanCoupon(rows.Scan)
			if err != nil {
				return err
			}
			coupons = append(coupons, *c)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return coupons, total, nil
}

func (r *CouponRepository) IncrementUsageIfBelowLimit(ctx context.Context, id uuid.UUID) error {
	return database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE coupon_svc.coupons
			 SET usage_count = usage_count + 1, updated_at = NOW()
			 WHERE id = $1
			   AND (usage_limit_total IS NULL OR usage_count < usage_limit_total)`,
			id,
		)
		if err != nil {
			return fmt.Errorf("increment usage: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return port.ErrUsageLimitExceeded
		}
		return nil
	})
}

func (r *CouponRepository) DecrementUsage(ctx context.Context, id uuid.UUID) error {
	return database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`UPDATE coupon_svc.coupons
			 SET usage_count = GREATEST(0, usage_count - 1), updated_at = NOW()
			 WHERE id = $1`,
			id,
		)
		if err != nil {
			return fmt.Errorf("decrement usage: %w", err)
		}
		return nil
	})
}

// Update writes the editable-field subset and returns the fresh row.
// The static columns (code, issuer, discount_type, discount_amount,
// currency, valid_from) are intentionally not in the SET list — a
// mutation must not retroactively change the deal buyers saw.
func (r *CouponRepository) Update(ctx context.Context, id uuid.UUID, patch port.CouponPatch) (*domain.Coupon, error) {
	var updated *domain.Coupon
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`UPDATE coupon_svc.coupons
			    SET title                = $2,
			        description          = $3,
			        min_order_amount     = $4,
			        max_discount_amount  = $5,
			        expires_at           = $6,
			        usage_limit_total    = $7,
			        usage_limit_per_user = $8,
			        updated_at           = NOW()
			  WHERE id = $1
			 RETURNING id, code, issuer_type, issuer_id, discount_type,
			           discount_percent_bps, discount_amount, currency,
			           min_order_amount, max_discount_amount,
			           valid_from, expires_at,
			           usage_limit_total, usage_limit_per_user, usage_count,
			           status, title, description, created_at, updated_at`,
			id,
			patch.Title, patch.Description,
			patch.MinOrderAmount, patch.MaxDiscountAmount,
			patch.ExpiresAt,
			patch.UsageLimitTotal, patch.UsageLimitPerUser,
		)
		c, err := scanCoupon(row.Scan)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("update coupon: %w", err)
		}
		updated = c
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *CouponRepository) SetStatus(ctx context.Context, id uuid.UUID, status domain.CouponStatus) error {
	return database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE coupon_svc.coupons SET status = $2, updated_at = NOW() WHERE id = $1`,
			id, string(status),
		)
		if err != nil {
			return fmt.Errorf("set coupon status: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("coupon %s not found", id)
		}
		return nil
	})
}

func (r *CouponRepository) queryOne(ctx context.Context, sql string, args ...any) (*domain.Coupon, error) {
	var c *domain.Coupon
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, sql, args...)
		res, err := scanCoupon(row.Scan)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		c = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

type scanFn func(dest ...any) error

func scanCoupon(scan scanFn) (*domain.Coupon, error) {
	var (
		c            domain.Coupon
		issuerType   string
		discountType string
		status       string
		title, desc  *string
	)
	err := scan(
		&c.ID, &c.Code, &issuerType, &c.IssuerID, &discountType,
		&c.DiscountPercentBps, &c.DiscountAmount, &c.Currency,
		&c.MinOrderAmount, &c.MaxDiscountAmount,
		&c.ValidFrom, &c.ExpiresAt,
		&c.UsageLimitTotal, &c.UsageLimitPerUser, &c.UsageCount,
		&status, &title, &desc, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.IssuerType = domain.IssuerType(issuerType)
	c.DiscountType = domain.DiscountType(discountType)
	c.Status = domain.CouponStatus(status)
	if title != nil {
		c.Title = *title
	}
	if desc != nil {
		c.Description = *desc
	}
	return &c, nil
}

// itoa is a tiny non-fmt int-to-string that keeps the placeholder SQL
// builder allocation-free in the hot list path.
func itoa(n int) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		return r
	}, fmtInt(n)))
}

func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
