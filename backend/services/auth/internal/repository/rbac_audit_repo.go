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

// Append inserts a new audit entry.
func (r *RBACAuditRepository) Append(ctx context.Context, e *domain.RBACAuditEntry) error {
	return database.TenantTx(ctx, r.pool, e.TenantID, func(tx pgx.Tx) error {
		return r.AppendTx(ctx, tx, e)
	})
}

// AppendTx inserts an audit entry using an existing transaction so that the
// audited mutation and the audit record commit atomically.
func (r *RBACAuditRepository) AppendTx(ctx context.Context, tx pgx.Tx, e *domain.RBACAuditEntry) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return tx.QueryRow(ctx,
		`INSERT INTO auth_svc.rbac_audit_log
		    (id, tenant_id, actor_auth0_user_id, target_auth0_user_id, scope, scope_id, action, before_role, after_role)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), NULLIF($9, ''))
		 RETURNING created_at`,
		e.ID, e.TenantID, e.ActorAuth0UserID, e.TargetAuth0UserID,
		e.Scope, e.ScopeID, e.Action, e.BeforeRole, e.AfterRole,
	).Scan(&e.CreatedAt)
}

// ListByTenant returns a paginated list of audit entries for a tenant,
// ordered by created_at DESC.
func (r *RBACAuditRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.RBACAuditEntry, int, error) {
	var entries []domain.RBACAuditEntry
	var total int
	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM auth_svc.rbac_audit_log WHERE tenant_id = $1`,
			tenantID,
		).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, actor_auth0_user_id, target_auth0_user_id,
			        scope, scope_id, action,
			        COALESCE(before_role, ''), COALESCE(after_role, ''),
			        created_at
			 FROM auth_svc.rbac_audit_log
			 WHERE tenant_id = $1
			 ORDER BY created_at DESC
			 LIMIT $2 OFFSET $3`,
			tenantID, limit, offset,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e domain.RBACAuditEntry
			if err := rows.Scan(
				&e.ID, &e.TenantID, &e.ActorAuth0UserID, &e.TargetAuth0UserID,
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
