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

// OutboxRelay polls shipping_svc.outbox_events and publishes unpublished events
// to Pub/Sub with at-least-once delivery.
//
// Each Run cycle is split into three short phases so that no DB connection or
// row lock is held while waiting on the external Pub/Sub call:
//
//  1. Claim TX: SELECT … FOR UPDATE SKIP LOCKED + UPDATE processing_at = NOW()
//     → COMMIT.  Rows are now claimed by this replica; locks are released.
//  2. Publish (no TX): for each claimed row call publisher.Publish.
//  3. Mark TX: UPDATE published_at = NOW() for successfully published rows,
//     UPDATE attempt_count++ / last_error for failed rows → COMMIT.
//
// Crash recovery: if the process dies after step 1 but before step 3, the
// claimed rows remain with processing_at set. After 5 minutes, the expiry
// condition (processing_at < NOW() - INTERVAL '5 minutes') lets another relay
// cycle reclaim and retry them.
//
// Deduplication: the outbox row's id is used as the Pub/Sub envelope event_id.
// This means re-deliveries of the same outbox row carry the same event_id,
// which consumers can use as a stable dedup key.
//
// Timestamp stability: the outbox row's created_at is used as the envelope
// timestamp so that re-deliveries carry a stable, meaningful timestamp rather
// than the arbitrary time of the retry attempt.
type OutboxRelay struct {
	store     outboxStore
	publisher pkgpubsub.Publisher
}

// outboxStore abstracts the DB queries used by OutboxRelay.
// Keeping this internal allows the relay logic to be unit-tested without a
// real Postgres connection.
type outboxStore interface {
	// claimBatch claims up to 50 unclaimed (or expired-claim) rows by setting
	// processing_at = NOW() and returns them. Uses FOR UPDATE SKIP LOCKED so
	// concurrent replicas get non-overlapping batches.
	claimBatch(ctx context.Context) ([]outboxRow, error)

	// markPublished sets published_at = NOW() and clears processing_at for
	// successfully published rows.
	markPublished(ctx context.Context, ids []uuid.UUID) error

	// recordFailure increments attempt_count, stores the error message, and
	// clears processing_at (so the 5-min expiry clock restarts) for a single
	// failed row.
	recordFailure(ctx context.Context, id uuid.UUID, errMsg string) error
}

// outboxRow is the data returned by claimBatch.
type outboxRow struct {
	id        uuid.UUID
	eventType string
	topic     string
	tenantID  uuid.UUID
	payload   json.RawMessage
	createdAt time.Time // used as stable envelope Timestamp across retries
}

// NewOutboxRelay creates a relay backed by pgxpool. publisher must not be nil.
func NewOutboxRelay(pool *pgxpool.Pool, publisher pkgpubsub.Publisher) *OutboxRelay {
	return &OutboxRelay{store: &pgxOutboxStore{pool: pool}, publisher: publisher}
}

// newOutboxRelayWithStore is used by tests to inject a fake store.
func newOutboxRelayWithStore(store outboxStore, publisher pkgpubsub.Publisher) *OutboxRelay {
	return &OutboxRelay{store: store, publisher: publisher}
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
				slog.Warn("outbox relay: run error", "error", err)
			}
		}
	}
}

// Run executes one claim → publish → mark cycle.
func (r *OutboxRelay) Run(ctx context.Context) error {
	// ── Phase 1: claim a batch ───────────────────────────────────────────────
	batch, err := r.store.claimBatch(ctx)
	if err != nil {
		return fmt.Errorf("claim outbox batch: %w", err)
	}
	if len(batch) == 0 {
		return nil
	}

	// ── Phase 2: publish (no DB connection held) ─────────────────────────────
	// Use the outbox row's id as the stable event_id across retries and
	// created_at as the stable envelope timestamp.
	type result struct {
		id  uuid.UUID
		err error
	}
	results := make([]result, len(batch))
	for i, row := range batch {
		event := pkgpubsub.Event{
			ID:        row.id.String(),     // stable across retries — dedup key for consumers
			Type:      row.eventType,
			TenantID:  row.tenantID.String(),
			Timestamp: row.createdAt.UTC(), // stable: original event time, not retry time
			Data:      row.payload,
		}
		results[i] = result{id: row.id, err: r.publisher.Publish(ctx, row.topic, event)}
		if results[i].err != nil {
			slog.Warn("outbox relay: publish failed, will retry after processing_at expiry",
				"id", row.id, "event_type", row.eventType, "error", results[i].err)
		}
	}

	// Partition into published / failed.
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

	// ── Phase 3a: mark successfully published rows ────────────────────────────
	if len(published) > 0 {
		if err := r.store.markPublished(ctx, published); err != nil {
			slog.Warn("outbox relay: mark-published failed, events will be re-sent after expiry",
				"count", len(published), "error", err)
			return err
		}
	}

	// ── Phase 3b: record failure metadata for observability ───────────────────
	// Increment attempt_count and store the last error message so operators can
	// identify stuck rows without grepping logs. processing_at is cleared so the
	// 5-minute expiry clock starts now.
	for _, f := range failed {
		if err := r.store.recordFailure(ctx, f.id, f.err); err != nil {
			slog.Warn("outbox relay: update attempt metadata failed",
				"id", f.id, "error", err)
		}
	}

	slog.Info("outbox relay: cycle complete",
		"published", len(published),
		"retrying", len(failed))
	return nil
}

// pgxOutboxStore is the production implementation of outboxStore backed by pgxpool.
type pgxOutboxStore struct {
	pool *pgxpool.Pool
}

func (s *pgxOutboxStore) claimBatch(ctx context.Context) ([]outboxRow, error) {
	rows, err := s.pool.Query(ctx, `
		WITH claimed AS (
			SELECT id
			FROM shipping_svc.outbox_events
			WHERE published_at IS NULL
			  AND (processing_at IS NULL
			       OR processing_at < NOW() - INTERVAL '5 minutes')
			ORDER BY created_at
			LIMIT 50
			FOR UPDATE SKIP LOCKED
		)
		UPDATE shipping_svc.outbox_events
		SET processing_at = NOW()
		FROM claimed
		WHERE shipping_svc.outbox_events.id = claimed.id
		RETURNING
			shipping_svc.outbox_events.id,
			event_type,
			topic,
			tenant_id,
			payload,
			created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batch []outboxRow
	for rows.Next() {
		var row outboxRow
		if err := rows.Scan(&row.id, &row.eventType, &row.topic, &row.tenantID, &row.payload, &row.createdAt); err != nil {
			return nil, fmt.Errorf("scan outbox row: %w", err)
		}
		batch = append(batch, row)
	}
	return batch, rows.Err()
}

func (s *pgxOutboxStore) markPublished(ctx context.Context, ids []uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE shipping_svc.outbox_events
		 SET published_at = NOW(), processing_at = NULL
		 WHERE id = ANY($1)`,
		ids,
	)
	return err
}

func (s *pgxOutboxStore) recordFailure(ctx context.Context, id uuid.UUID, errMsg string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE shipping_svc.outbox_events
		 SET attempt_count  = attempt_count + 1,
		     last_error     = $1,
		     processing_at  = NULL
		 WHERE id = $2`,
		errMsg, id,
	)
	return err
}
