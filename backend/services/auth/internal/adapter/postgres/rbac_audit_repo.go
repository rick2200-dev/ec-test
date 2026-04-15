package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// RBACAuditRepository handles persistence of rbac_audit_log entries.
type RBACAuditRepository struct {
	pool *pgxpool.Pool
}

// NewRBACAuditRepository creates a new RBACAuditRepository.
func NewRBACAuditRepository(pool *pgxpool.Pool) *RBACAuditRepository {
	return &RBACAuditRepository{pool: pool}
}

// withTx uses the transaction from ctx if one was placed there by
// database.WithTx, otherwise opens a new transaction.
func (r *RBACAuditRepository) withTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	if tx, ok := database.TxFromContext(ctx); ok {
		return fn(tx)
	}
	return database.Tx(ctx, r.pool, fn)
}

// Append inserts a new audit entry. When ctx carries a transaction it joins
// that transaction so the audited mutation and the audit record commit
// atomically; otherwise opens its own transaction.
func (r *RBACAuditRepository) Append(ctx context.Context, e *domain.RBACAuditEntry) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return r.withTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO auth_svc.rbac_audit_log
			    (id, actor_auth0_user_id, target_auth0_user_id, scope, scope_id, action, before_role, after_role)
			 VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''))
			 RETURNING created_at`,
			e.ID, e.ActorAuth0UserID, e.TargetAuth0UserID,
			e.Scope, e.ScopeID, e.Action, e.BeforeRole, e.AfterRole,
		).Scan(&e.CreatedAt)
	})
}

// List returns a paginated list of audit entries, ordered by created_at DESC.
func (r *RBACAuditRepository) List(ctx context.Context, limit, offset int) ([]domain.RBACAuditEntry, int, error) {
	var entries []domain.RBACAuditEntry
	var total int
	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM auth_svc.rbac_audit_log`,
		).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx,
			`SELECT id, actor_auth0_user_id, target_auth0_user_id,
			        scope, scope_id, action,
			        COALESCE(before_role, ''), COALESCE(after_role, ''),
			        created_at
			 FROM auth_svc.rbac_audit_log
			 ORDER BY created_at DESC
			 LIMIT $1 OFFSET $2`,
			limit, offset,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e domain.RBACAuditEntry
			if err := rows.Scan(
				&e.ID, &e.ActorAuth0UserID, &e.TargetAuth0UserID,
				&e.Scope, &e.ScopeID, &e.Action,
				&e.BeforeRole, &e.AfterRole,
				&e.CreatedAt,
			); err != nil {
				return err
			}
			entries = append(entries, e)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list rbac_audit_log: %w", err)
	}
	return entries, total, nil
}
