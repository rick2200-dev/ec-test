package port

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/shipping/internal/domain"
)

// OutboxEvent is a domain event that must be published to Pub/Sub.
// It is written atomically alongside the shipment status update and
// published asynchronously by the outbox relay worker.
type OutboxEvent struct {
	ID        uuid.UUID
	EventType string
	Topic     string
	TenantID  uuid.UUID
	Payload   json.RawMessage
}

// ShipmentRepository is the driven port for shipment persistence.
type ShipmentRepository interface {
	// Create inserts a new shipment. If a shipment for the same order_id already
	// exists (UNIQUE constraint), it is a no-op (ON CONFLICT DO NOTHING) and
	// returns nil — this makes the order.paid subscriber idempotent.
	Create(ctx context.Context, s *domain.Shipment) error

	// GetByID returns the shipment with the given ID scoped to the tenant.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Shipment, error)

	// GetByOrderID returns the shipment for the given order scoped to the tenant.
	GetByOrderID(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.Shipment, error)

	// ListBySeller returns shipments for a seller with optional status filter and
	// pagination. An empty status string means "all statuses".
	ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]*domain.Shipment, int, error)

	// UpdateStatus atomically transitions the shipment to the new status.
	// The update is guarded by WHERE status = expectedStatus so concurrent
	// requests are safe. Returns ErrInvalidTransition when RowsAffected == 0.
	UpdateStatus(ctx context.Context, s *domain.Shipment, expectedStatus string) error

	// AppendEvent inserts a ShipmentEvent audit row. Called within the same
	// transaction as UpdateStatus.
	AppendEvent(ctx context.Context, e *domain.ShipmentEvent) error

	// CancelByOrderID transitions the shipment for the given order to cancelled,
	// but only if it is still in pending or ready_to_ship. If the shipment is
	// already shipped or delivered this is a no-op (not an error).
	CancelByOrderID(ctx context.Context, tenantID, orderID uuid.UUID) error

	// AppendOutboxEvent inserts an outbox event row inside the caller's
	// transaction (via context). The outbox relay worker will pick it up and
	// publish it to Pub/Sub, providing at-least-once delivery even when the
	// process crashes between the DB commit and a best-effort publish call.
	AppendOutboxEvent(ctx context.Context, e OutboxEvent) error
}

// TxRunner is the driven port for running database transactions.
type TxRunner interface {
	RunTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(ctx context.Context) error) error
}
