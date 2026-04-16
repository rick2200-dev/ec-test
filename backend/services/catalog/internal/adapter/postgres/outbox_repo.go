package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/port"
)

// OutboxRepository persists catalog_svc.outbox_events rows. The relay
// worker in internal/adapter/pubsub drains them to Pub/Sub.
type OutboxRepository struct {
	pool *pgxpool.Pool
}

// NewOutboxRepository wires the store to a pgxpool.
func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

// AppendOutboxEvent inserts an outbox row. When the caller is already
// inside a Tx (via context) the insert joins that Tx so the event commits
// atomically with the business write. Otherwise it opens a short Tx on
// the pool — useful for callers that don't need atomicity with other
// writes (currently unused but kept symmetric with shipping's pattern).
func (r *OutboxRepository) AppendOutboxEvent(ctx context.Context, e port.OutboxEvent) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	exec := func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO catalog_svc.outbox_events (id, event_type, topic, payload, created_at)
			 VALUES ($1, $2, $3, $4, NOW())`,
			e.ID, e.EventType, e.Topic, e.Payload,
		)
		return err
	}
	if tx, ok := database.TxFromContext(ctx); ok {
		return exec(tx)
	}
	return database.Tx(ctx, r.pool, exec)
}
