package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/inquiry/internal/domain"
)

// pgUniqueViolation is the PostgreSQL SQLSTATE for a unique constraint
// violation (23505). We detect it via *pgconn.PgError rather than string
// matching on err.Error() so the check is robust across driver versions
// and locales.
const pgUniqueViolation = "23505"

// isUniqueViolation reports whether err is a pg unique-constraint violation
// on the given constraint name.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == constraint
}

// InquiryRepository persists inquiries and their messages.
type InquiryRepository struct {
	pool *pgxpool.Pool
}

// NewInquiryRepository constructs an InquiryRepository.
func NewInquiryRepository(pool *pgxpool.Pool) *InquiryRepository {
	return &InquiryRepository{pool: pool}
}

// Create inserts a new inquiry + its first message atomically.
//
// If a thread already exists for (tenant, buyer, seller, sku), the unique
// constraint `uq_inquiry_per_sku` trips and Create instead loads the
// existing row and appends the new message to it — giving callers
// idempotent "open a thread" semantics. The in parameter is mutated with
// the final id/timestamps on success.
func (r *InquiryRepository) Create(
	ctx context.Context,
	tenantID uuid.UUID,
	inq *domain.Inquiry,
	firstMsg *domain.InquiryMessage,
) (*domain.InquiryWithMessages, error) {
	var result *domain.InquiryWithMessages

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		// Check for existing thread (idempotent create: same buyer+seller+sku
		// collapses to one thread). We're already inside a tenant-scoped
		// transaction so a concurrent insert will trip the unique constraint
		// on INSERT below — see the fallback path there.
		existing, err := loadInquiryByParticipantsTx(ctx, tx, tenantID, inq.BuyerAuth0ID, inq.SellerID, inq.SKUID)
		if err != nil {
			return err
		}
		if existing != nil {
			*inq = *existing
		} else {
			inq.ID = uuid.New()
			inq.TenantID = tenantID
			inq.Status = domain.InquiryStatusOpen

			err := tx.QueryRow(ctx,
				`INSERT INTO inquiry_svc.inquiries
				 (id, tenant_id, buyer_auth0_id, seller_id, sku_id, product_name, sku_code, subject, status)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				 RETURNING last_message_at, created_at, updated_at`,
				inq.ID, inq.TenantID, inq.BuyerAuth0ID, inq.SellerID, inq.SKUID,
				inq.ProductName, inq.SKUCode, inq.Subject, inq.Status,
			).Scan(&inq.LastMessageAt, &inq.CreatedAt, &inq.UpdatedAt)
			if err != nil {
				// A concurrent create lost the race and hit uq_inquiry_per_sku.
				// Fall back to the existing row.
				if isUniqueViolation(err, "uq_inquiry_per_sku") {
					row, loadErr := loadInquiryByParticipantsTx(ctx, tx, tenantID, inq.BuyerAuth0ID, inq.SellerID, inq.SKUID)
					if loadErr != nil {
						return loadErr
					}
					if row == nil {
						return fmt.Errorf("inquiry unique violation but no row found")
					}
					*inq = *row
				} else {
					return fmt.Errorf("insert inquiry: %w", err)
				}
			}
		}

		// Append the first message + bump last_message_at.
		firstMsg.InquiryID = inq.ID
		if err := appendMessageTx(ctx, tx, tenantID, firstMsg); err != nil {
			return err
		}
		inq.LastMessageAt = firstMsg.CreatedAt

		messages, err := loadMessagesTx(ctx, tx, tenantID, inq.ID)
		if err != nil {
			return err
		}

		result = &domain.InquiryWithMessages{
			Inquiry:  *inq,
			Messages: messages,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetByID returns an inquiry with the full message thread, or nil if the
// id is unknown within this tenant.
func (r *InquiryRepository) GetByID(ctx context.Context, tenantID, inquiryID uuid.UUID) (*domain.InquiryWithMessages, error) {
	var result *domain.InquiryWithMessages

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		inq, err := loadInquiryByIDTx(ctx, tx, tenantID, inquiryID)
		if err != nil {
			return err
		}
		if inq == nil {
			return nil
		}
		messages, err := loadMessagesTx(ctx, tx, tenantID, inq.ID)
		if err != nil {
			return err
		}
		result = &domain.InquiryWithMessages{Inquiry: *inq, Messages: messages}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get inquiry: %w", err)
	}
	return result, nil
}

// ListByBuyer returns paginated inquiries for a buyer, newest message first.
// The UnreadCount field on each row counts seller messages the buyer has not
// yet marked as read.
func (r *InquiryRepository) ListByBuyer(
	ctx context.Context,
	tenantID uuid.UUID,
	buyerAuth0ID string,
	limit, offset int,
) ([]domain.Inquiry, int, error) {
	return r.listByRole(ctx, tenantID, listFilter{
		role:         "buyer",
		buyerAuth0ID: buyerAuth0ID,
	}, limit, offset)
}

// ListBySeller returns paginated inquiries for a seller. status may be
// empty to include both open and closed.
func (r *InquiryRepository) ListBySeller(
	ctx context.Context,
	tenantID, sellerID uuid.UUID,
	status string,
	limit, offset int,
) ([]domain.Inquiry, int, error) {
	return r.listByRole(ctx, tenantID, listFilter{
		role:     "seller",
		sellerID: sellerID,
		status:   status,
	}, limit, offset)
}

type listFilter struct {
	role         string // "buyer" | "seller"
	buyerAuth0ID string
	sellerID     uuid.UUID
	status       string
}

func (r *InquiryRepository) listByRole(
	ctx context.Context,
	tenantID uuid.UUID,
	f listFilter,
	limit, offset int,
) ([]domain.Inquiry, int, error) {
	var out []domain.Inquiry
	var total int

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		conds := "tenant_id = $1"
		args := []any{tenantID}
		idx := 2

		if f.role == "buyer" {
			conds += fmt.Sprintf(" AND buyer_auth0_id = $%d", idx)
			args = append(args, f.buyerAuth0ID)
			idx++
		} else {
			conds += fmt.Sprintf(" AND seller_id = $%d", idx)
			args = append(args, f.sellerID)
			idx++
		}
		if f.status != "" {
			conds += fmt.Sprintf(" AND status = $%d", idx)
			args = append(args, f.status)
			idx++
		}

		if err := tx.QueryRow(ctx,
			fmt.Sprintf(`SELECT COUNT(*) FROM inquiry_svc.inquiries WHERE %s`, conds),
			args...,
		).Scan(&total); err != nil {
			return fmt.Errorf("count inquiries: %w", err)
		}

		// Viewer for unread-count: buyer counts messages from seller and vice versa.
		viewer := f.role
		other := domain.SenderTypeSeller
		if viewer == "seller" {
			other = domain.SenderTypeBuyer
		}

		listQuery := fmt.Sprintf(
			`SELECT i.id, i.tenant_id, i.buyer_auth0_id, i.seller_id, i.sku_id,
			        i.product_name, i.sku_code, i.subject, i.status,
			        i.last_message_at, i.created_at, i.updated_at,
			        COALESCE((
			            SELECT COUNT(*)
			              FROM inquiry_svc.inquiry_messages m
			             WHERE m.inquiry_id = i.id
			               AND m.tenant_id  = i.tenant_id
			               AND m.sender_type = $%d
			               AND m.read_at IS NULL
			        ), 0) AS unread_count
			   FROM inquiry_svc.inquiries i
			  WHERE %s
			  ORDER BY i.last_message_at DESC
			  LIMIT $%d OFFSET $%d`,
			idx, conds, idx+1, idx+2,
		)
		args = append(args, other, limit, offset)

		rows, err := tx.Query(ctx, listQuery, args...)
		if err != nil {
			return fmt.Errorf("list inquiries: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var inq domain.Inquiry
			if err := rows.Scan(
				&inq.ID, &inq.TenantID, &inq.BuyerAuth0ID, &inq.SellerID, &inq.SKUID,
				&inq.ProductName, &inq.SKUCode, &inq.Subject, &inq.Status,
				&inq.LastMessageAt, &inq.CreatedAt, &inq.UpdatedAt, &inq.UnreadCount,
			); err != nil {
				return fmt.Errorf("scan inquiry: %w", err)
			}
			out = append(out, inq)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// AppendMessage inserts a new message into an existing thread and bumps
// last_message_at atomically.
func (r *InquiryRepository) AppendMessage(ctx context.Context, tenantID uuid.UUID, msg *domain.InquiryMessage) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return appendMessageTx(ctx, tx, tenantID, msg)
	})
}

// MarkRead stamps read_at = NOW() on unread messages written by the *other*
// participant. readerType is either SenderTypeBuyer or SenderTypeSeller —
// messages authored by that role are left alone.
func (r *InquiryRepository) MarkRead(ctx context.Context, tenantID, inquiryID uuid.UUID, readerType string) error {
	// Determine the author side whose messages should be marked read.
	otherSide := domain.SenderTypeSeller
	if readerType == domain.SenderTypeSeller {
		otherSide = domain.SenderTypeBuyer
	}

	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`UPDATE inquiry_svc.inquiry_messages
			    SET read_at = NOW()
			  WHERE tenant_id  = $1
			    AND inquiry_id = $2
			    AND sender_type = $3
			    AND read_at IS NULL`,
			tenantID, inquiryID, otherSide,
		)
		if err != nil {
			return fmt.Errorf("mark messages read: %w", err)
		}
		return nil
	})
}

// Close transitions the thread to status = closed.
func (r *InquiryRepository) Close(ctx context.Context, tenantID, inquiryID uuid.UUID) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE inquiry_svc.inquiries
			    SET status = $3, updated_at = NOW()
			  WHERE id = $1 AND tenant_id = $2`,
			inquiryID, tenantID, domain.InquiryStatusClosed,
		)
		if err != nil {
			return fmt.Errorf("close inquiry: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("inquiry not found")
		}
		return nil
	})
}

// --- internal helpers ---

func loadInquiryByIDTx(ctx context.Context, tx pgx.Tx, tenantID, inquiryID uuid.UUID) (*domain.Inquiry, error) {
	var inq domain.Inquiry
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, buyer_auth0_id, seller_id, sku_id,
		        product_name, sku_code, subject, status,
		        last_message_at, created_at, updated_at
		   FROM inquiry_svc.inquiries
		  WHERE id = $1 AND tenant_id = $2`,
		inquiryID, tenantID,
	).Scan(
		&inq.ID, &inq.TenantID, &inq.BuyerAuth0ID, &inq.SellerID, &inq.SKUID,
		&inq.ProductName, &inq.SKUCode, &inq.Subject, &inq.Status,
		&inq.LastMessageAt, &inq.CreatedAt, &inq.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load inquiry by id: %w", err)
	}
	return &inq, nil
}

func loadInquiryByParticipantsTx(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	buyerAuth0ID string,
	sellerID, skuID uuid.UUID,
) (*domain.Inquiry, error) {
	var inq domain.Inquiry
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, buyer_auth0_id, seller_id, sku_id,
		        product_name, sku_code, subject, status,
		        last_message_at, created_at, updated_at
		   FROM inquiry_svc.inquiries
		  WHERE tenant_id = $1
		    AND buyer_auth0_id = $2
		    AND seller_id = $3
		    AND sku_id = $4`,
		tenantID, buyerAuth0ID, sellerID, skuID,
	).Scan(
		&inq.ID, &inq.TenantID, &inq.BuyerAuth0ID, &inq.SellerID, &inq.SKUID,
		&inq.ProductName, &inq.SKUCode, &inq.Subject, &inq.Status,
		&inq.LastMessageAt, &inq.CreatedAt, &inq.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load inquiry by participants: %w", err)
	}
	return &inq, nil
}

func appendMessageTx(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, msg *domain.InquiryMessage) error {
	msg.ID = uuid.New()
	msg.TenantID = tenantID

	err := tx.QueryRow(ctx,
		`INSERT INTO inquiry_svc.inquiry_messages
		 (id, tenant_id, inquiry_id, sender_type, sender_id, body)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at`,
		msg.ID, msg.TenantID, msg.InquiryID, msg.SenderType, msg.SenderID, msg.Body,
	).Scan(&msg.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}

	// Bump parent thread's last_message_at so list queries can sort without a JOIN.
	_, err = tx.Exec(ctx,
		`UPDATE inquiry_svc.inquiries
		    SET last_message_at = $3, updated_at = NOW()
		  WHERE id = $1 AND tenant_id = $2`,
		msg.InquiryID, tenantID, msg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("bump last_message_at: %w", err)
	}
	return nil
}

func loadMessagesTx(ctx context.Context, tx pgx.Tx, tenantID, inquiryID uuid.UUID) ([]domain.InquiryMessage, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, inquiry_id, sender_type, sender_id, body, read_at, created_at
		   FROM inquiry_svc.inquiry_messages
		  WHERE inquiry_id = $1 AND tenant_id = $2
		  ORDER BY created_at ASC`,
		inquiryID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("load messages: %w", err)
	}
	defer rows.Close()

	var out []domain.InquiryMessage
	for rows.Next() {
		var m domain.InquiryMessage
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.InquiryID, &m.SenderType, &m.SenderID, &m.Body, &m.ReadAt, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
