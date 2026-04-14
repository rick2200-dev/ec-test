package repository

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/shipping/internal/domain"
	"github.com/Riku-KANO/ec-test/services/shipping/internal/port"
)

// ShipmentRepository is the Postgres implementation of port.ShipmentRepository.
type ShipmentRepository struct {
	pool *pgxpool.Pool
}

// NewShipmentRepository creates a new ShipmentRepository.
func NewShipmentRepository(pool *pgxpool.Pool) *ShipmentRepository {
	return &ShipmentRepository{pool: pool}
}

// Create inserts a ready_to_ship shipment for the given order.
//
// Idempotency: duplicate order.paid deliveries are silently skipped via
// ON CONFLICT on the unique (order_id) index.
//
// Race / out-of-order protection: an advisory lock on the order_id serializes
// concurrent Create and CancelByOrderID calls on the same order. Within the
// lock, a single INSERT … SELECT … WHERE NOT EXISTS atomically checks the
// cancellation tombstone and inserts the shipment — no TOCTOU gap.
func (r *ShipmentRepository) Create(ctx context.Context, s *domain.Shipment) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	addrJSON := json.RawMessage("{}")
	if len(s.ShippingAddress) > 0 {
		addrJSON = s.ShippingAddress
	}
	return database.TenantTx(ctx, r.pool, s.TenantID, func(tx pgx.Tx) error {
		// Serialize with CancelByOrderID on the same order so the tombstone
		// check and shipment insert are never interleaved with a concurrent
		// cancel that would otherwise win the race.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, orderLockKey(s.OrderID)); err != nil {
			return fmt.Errorf("acquire order advisory lock: %w", err)
		}

		// Single atomic statement: insert only when no tombstone exists.
		// ON CONFLICT handles duplicate order.paid re-deliveries.
		_, err := tx.Exec(ctx,
			`INSERT INTO shipping_svc.shipments
			 (id, tenant_id, seller_id, order_id, buyer_auth0_id, status, shipping_address, created_at, updated_at)
			 SELECT $1,$2,$3,$4,$5,$6,$7,NOW(),NOW()
			 WHERE NOT EXISTS (
			     SELECT 1 FROM shipping_svc.cancelled_order_tombstones WHERE order_id = $4
			 )
			 ON CONFLICT (order_id) DO NOTHING`,
			s.ID, s.TenantID, s.SellerID, s.OrderID, s.BuyerAuth0ID, domain.StatusReadyToShip, addrJSON,
		)
		return err
	})
}

// GetByID returns the shipment with the given id scoped to the tenant.
func (r *ShipmentRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Shipment, error) {
	return r.fetchOne(ctx, tenantID,
		`SELECT id,tenant_id,seller_id,order_id,buyer_auth0_id,status,shipping_address,
		        carrier,tracking_number,shipped_at,delivered_at,note,created_at,updated_at
		 FROM shipping_svc.shipments
		 WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
	)
}

// GetByOrderID returns the shipment for the given order scoped to the tenant.
func (r *ShipmentRepository) GetByOrderID(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.Shipment, error) {
	return r.fetchOne(ctx, tenantID,
		`SELECT id,tenant_id,seller_id,order_id,buyer_auth0_id,status,shipping_address,
		        carrier,tracking_number,shipped_at,delivered_at,note,created_at,updated_at
		 FROM shipping_svc.shipments
		 WHERE order_id=$1 AND tenant_id=$2`,
		orderID, tenantID,
	)
}

// ListBySeller returns paginated shipments for a seller.
func (r *ShipmentRepository) ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]*domain.Shipment, int, error) {
	var total int
	var items []*domain.Shipment

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		args := []any{tenantID, sellerID}
		statusFilter := ""
		if status != "" {
			args = append(args, status)
			statusFilter = fmt.Sprintf(" AND status=$%d", len(args))
		}

		if err := tx.QueryRow(ctx,
			"SELECT COUNT(*) FROM shipping_svc.shipments WHERE tenant_id=$1 AND seller_id=$2"+statusFilter,
			args...,
		).Scan(&total); err != nil {
			return fmt.Errorf("count shipments: %w", err)
		}

		args = append(args, limit, offset)
		rows, err := tx.Query(ctx,
			`SELECT id,tenant_id,seller_id,order_id,buyer_auth0_id,status,shipping_address,
			        carrier,tracking_number,shipped_at,delivered_at,note,created_at,updated_at
			 FROM shipping_svc.shipments
			 WHERE tenant_id=$1 AND seller_id=$2`+statusFilter+
				fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)),
			args...,
		)
		if err != nil {
			return fmt.Errorf("list shipments: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			s, err := scanShipment(rows)
			if err != nil {
				return err
			}
			items = append(items, s)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// UpdateStatus atomically sets the new status on the shipment, guarded by
// WHERE status = expectedStatus. Returns ErrInvalidTransition when no row
// was updated (either the shipment is gone or the guard failed).
func (r *ShipmentRepository) UpdateStatus(ctx context.Context, s *domain.Shipment, expectedStatus string) error {
	exec := func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE shipping_svc.shipments
			 SET status=$1, carrier=$2, tracking_number=$3, shipped_at=$4, delivered_at=$5, note=$6, updated_at=NOW()
			 WHERE id=$7 AND tenant_id=$8 AND status=$9`,
			s.Status, s.Carrier, s.TrackingNumber, s.ShippedAt, s.DeliveredAt, s.Note,
			s.ID, s.TenantID, expectedStatus,
		)
		if err != nil {
			return fmt.Errorf("update shipment status: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrInvalidTransition
		}
		return nil
	}

	if tx, ok := database.TxFromContext(ctx); ok {
		return exec(tx)
	}
	return database.TenantTx(ctx, r.pool, s.TenantID, exec)
}

// CancelByOrderID transitions the shipment for the given order to cancelled,
// but only if it is still in pending or ready_to_ship. Already shipped or
// delivered shipments are silently skipped (the cancellation request must be
// rejected by the order layer before reaching this point).
//
// Out-of-order guard (with advisory lock): when order.cancelled arrives before
// order.paid and no shipment exists yet, a tombstone is inserted so that a
// later order.paid event is absorbed. The advisory lock serializes this
// operation with Create so there is no race between tombstone insert and
// shipment insert.
//
// The tombstone is only written when the shipment is truly absent — not when
// it already exists in a non-cancellable state — so the table meaning stays
// "cancel arrived before paid" rather than "cancel arrived".
func (r *ShipmentRepository) CancelByOrderID(ctx context.Context, tenantID, orderID uuid.UUID) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		// Serialize with Create on the same order.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, orderLockKey(orderID)); err != nil {
			return fmt.Errorf("acquire order advisory lock: %w", err)
		}

		tag, err := tx.Exec(ctx,
			`UPDATE shipping_svc.shipments
			 SET status='cancelled', updated_at=NOW()
			 WHERE order_id=$1 AND tenant_id=$2 AND status IN ('pending','ready_to_ship')`,
			orderID, tenantID,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			return nil // shipment was cancelled; no tombstone needed
		}

		// RowsAffected == 0: either no shipment exists yet, or the shipment is
		// already shipped/delivered. Only insert a tombstone in the first case.
		var shipmentExists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM shipping_svc.shipments WHERE order_id=$1 AND tenant_id=$2)`,
			orderID, tenantID,
		).Scan(&shipmentExists); err != nil {
			return fmt.Errorf("check shipment existence: %w", err)
		}
		if shipmentExists {
			// Already shipped/delivered — no-op, no tombstone.
			return nil
		}

		// No shipment yet: record tombstone so a future order.paid is absorbed.
		_, err = tx.Exec(ctx,
			`INSERT INTO shipping_svc.cancelled_order_tombstones (order_id, tenant_id)
			 VALUES ($1, $2) ON CONFLICT (order_id) DO NOTHING`,
			orderID, tenantID,
		)
		return err
	})
}

// AppendEvent inserts an audit event row.
func (r *ShipmentRepository) AppendEvent(ctx context.Context, e *domain.ShipmentEvent) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	var payload any
	if len(e.Payload) > 0 {
		payload = e.Payload
	}

	exec := func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO shipping_svc.shipment_events
			 (id, tenant_id, shipment_id, from_status, to_status, actor_type, actor_id, payload, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())`,
			e.ID, e.TenantID, e.ShipmentID, nullableStr(e.FromStatus), e.ToStatus,
			e.ActorType, nullableStr(e.ActorID), payload,
		)
		return err
	}

	if tx, ok := database.TxFromContext(ctx); ok {
		return exec(tx)
	}
	return database.TenantTx(ctx, r.pool, e.TenantID, exec)
}

// fetchOne runs a SELECT ... and scans a single Shipment row.
func (r *ShipmentRepository) fetchOne(ctx context.Context, tenantID uuid.UUID, query string, args ...any) (*domain.Shipment, error) {
	var s *domain.Shipment
	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, query, args...)
		var err error
		s, err = scanShipmentRow(row)
		return err
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrShipmentNotFound
		}
		return nil, err
	}
	return s, nil
}

// scanShipment scans a Shipment from a Rows cursor.
func scanShipment(row pgx.Rows) (*domain.Shipment, error) {
	var s domain.Shipment
	var addrJSON []byte
	var carrier, tracking, note *string
	var shippedAt, deliveredAt *time.Time

	if err := row.Scan(
		&s.ID, &s.TenantID, &s.SellerID, &s.OrderID, &s.BuyerAuth0ID, &s.Status,
		&addrJSON, &carrier, &tracking, &shippedAt, &deliveredAt, &note,
		&s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan shipment row: %w", err)
	}
	if len(addrJSON) > 0 {
		s.ShippingAddress = json.RawMessage(addrJSON)
	}
	if carrier != nil {
		s.Carrier = *carrier
	}
	if tracking != nil {
		s.TrackingNumber = *tracking
	}
	if note != nil {
		s.Note = *note
	}
	s.ShippedAt = shippedAt
	s.DeliveredAt = deliveredAt
	return &s, nil
}

// scanShipmentRow scans a Shipment from a QueryRow result.
func scanShipmentRow(row pgx.Row) (*domain.Shipment, error) {
	var s domain.Shipment
	var addrJSON []byte
	var carrier, tracking, note *string
	var shippedAt, deliveredAt *time.Time

	if err := row.Scan(
		&s.ID, &s.TenantID, &s.SellerID, &s.OrderID, &s.BuyerAuth0ID, &s.Status,
		&addrJSON, &carrier, &tracking, &shippedAt, &deliveredAt, &note,
		&s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(addrJSON) > 0 {
		s.ShippingAddress = json.RawMessage(addrJSON)
	}
	if carrier != nil {
		s.Carrier = *carrier
	}
	if tracking != nil {
		s.TrackingNumber = *tracking
	}
	if note != nil {
		s.Note = *note
	}
	s.ShippedAt = shippedAt
	s.DeliveredAt = deliveredAt
	return &s, nil
}

// AppendOutboxEvent inserts an outbox event row, joining the caller's
// transaction via context if one is present. Called from the app layer
// inside the same TX that updates shipment status, so the event is
// durably committed before the status change becomes visible.
func (r *ShipmentRepository) AppendOutboxEvent(ctx context.Context, e port.OutboxEvent) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	exec := func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO shipping_svc.outbox_events (id, event_type, topic, tenant_id, payload, created_at)
			 VALUES ($1,$2,$3,$4,$5,NOW())`,
			e.ID, e.EventType, e.Topic, e.TenantID, e.Payload,
		)
		return err
	}
	if tx, ok := database.TxFromContext(ctx); ok {
		return exec(tx)
	}
	return database.TenantTx(ctx, r.pool, e.TenantID, exec)
}

// orderLockKey converts a UUID to a stable int64 suitable for
// pg_advisory_xact_lock. Using the first 8 bytes of the UUID gives a
// well-distributed key for random (v4) UUIDs with negligible collision rate.
func orderLockKey(id uuid.UUID) int64 {
	return int64(binary.BigEndian.Uint64(id[:8]))
}

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

