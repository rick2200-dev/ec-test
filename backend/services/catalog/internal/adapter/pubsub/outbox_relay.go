// Package subscriber hosts the catalog outbox relay. The relay polls
// catalog_svc.outbox_events for unpublished rows and drains them into
// Pub/Sub (topic "product-events"). Pattern mirrors shipping's outbox —
// see services/shipping/internal/adapter/pubsub/outbox_relay.go for
// extended commentary on the three-phase claim/publish/mark cycle.
package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	pkgpubsub "github.com/Riku-KANO/ec-test/pkg/pubsub"
)

type OutboxRelay struct {
	store     outboxStore
	publisher pkgpubsub.Publisher
}

type outboxStore interface {
	claimBatch(ctx context.Context) ([]outboxRow, error)
	markPublished(ctx context.Context, ids []uuid.UUID) error
	recordFailure(ctx context.Context, id uuid.UUID, errMsg string) error
}

type outboxRow struct {
	id        uuid.UUID
	eventType string
	topic     string
	payload   json.RawMessage
	createdAt time.Time
}

// NewOutboxRelay creates a relay backed by pgxpool. publisher must not be nil.
func NewOutboxRelay(pool *pgxpool.Pool, publisher pkgpubsub.Publisher) *OutboxRelay {
	return &OutboxRelay{store: &pgxOutboxStore{pool: pool}, publisher: publisher}
}

// Start runs the relay on a 5-second ticker until ctx is cancelled.
func (r *OutboxRelay) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.Run(ctx); err != nil {
				slog.Warn("catalog outbox relay: run error", "error", err)
			}
		}
	}
}

// Run executes one claim → publish → mark cycle.
func (r *OutboxRelay) Run(ctx context.Context) error {
	batch, err := r.store.claimBatch(ctx)
	if err != nil {
		return fmt.Errorf("claim outbox batch: %w", err)
	}
	if len(batch) == 0 {
		return nil
	}

	type result struct {
		id  uuid.UUID
		err error
	}
	results := make([]result, len(batch))
	for i, row := range batch {
		event := pkgpubsub.Event{
			ID:        row.id.String(),
			Type:      row.eventType,
			Timestamp: row.createdAt.UTC(),
			Data:      row.payload,
		}
		results[i] = result{id: row.id, err: r.publisher.Publish(ctx, row.topic, event)}
		if results[i].err != nil {
			slog.Warn("catalog outbox relay: publish failed, will retry after processing_at expiry",
				"id", row.id, "event_type", row.eventType, "error", results[i].err)
		}
	}

	var published []uuid.UUID
	type failedRow struct {
		id  uuid.UUID
		err string
	}
	var failed []failedRow
	for _, res := range results {
		if res.err == nil {
			published = append(published, res.id)
		} else {
			failed = append(failed, failedRow{id: res.id, err: res.err.Error()})
		}
	}

	if len(published) > 0 {
		if err := r.store.markPublished(ctx, published); err != nil {
			slog.Warn("catalog outbox relay: mark-published failed, events will be re-sent after expiry",
				"count", len(published), "error", err)
			return err
		}
	}
	for _, f := range failed {
		if err := r.store.recordFailure(ctx, f.id, f.err); err != nil {
			slog.Warn("catalog outbox relay: update attempt metadata failed",
				"id", f.id, "error", err)
		}
	}

	slog.Info("catalog outbox relay: cycle complete",
		"published", len(published),
		"retrying", len(failed))
	return nil
}

type pgxOutboxStore struct {
	pool *pgxpool.Pool
}

func (s *pgxOutboxStore) claimBatch(ctx context.Context) ([]outboxRow, error) {
	rows, err := s.pool.Query(ctx, `
		WITH claimed AS (
			SELECT id
			FROM catalog_svc.outbox_events
			WHERE published_at IS NULL
			  AND (processing_at IS NULL
			       OR processing_at < NOW() - INTERVAL '5 minutes')
			ORDER BY created_at
			LIMIT 50
			FOR UPDATE SKIP LOCKED
		)
		UPDATE catalog_svc.outbox_events
		SET processing_at = NOW()
		FROM claimed
		WHERE catalog_svc.outbox_events.id = claimed.id
		RETURNING
			catalog_svc.outbox_events.id,
			event_type,
			topic,
			payload,
			created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batch []outboxRow
	for rows.Next() {
		var row outboxRow
		if err := rows.Scan(&row.id, &row.eventType, &row.topic, &row.payload, &row.createdAt); err != nil {
			return nil, fmt.Errorf("scan outbox row: %w", err)
		}
		batch = append(batch, row)
	}
	return batch, rows.Err()
}

func (s *pgxOutboxStore) markPublished(ctx context.Context, ids []uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE catalog_svc.outbox_events
		 SET published_at = NOW(), processing_at = NULL
		 WHERE id = ANY($1)`,
		ids,
	)
	return err
}

func (s *pgxOutboxStore) recordFailure(ctx context.Context, id uuid.UUID, errMsg string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE catalog_svc.outbox_events
		 SET attempt_count = attempt_count + 1,
		     last_error    = $1,
		     processing_at = NULL
		 WHERE id = $2`,
		errMsg, id,
	)
	return err
}
