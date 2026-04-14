package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/services/notification/internal/port"
)

// EventDeduplicator implements port.Deduplicator using a status-based inbox
// backed by notification_svc.processed_events.
//
// TryClaim SQL:
//
//	WITH claim AS (
//	    INSERT … ON CONFLICT DO UPDATE SET processing_at=NOW(), status='processing'
//	    WHERE status='processing' AND processing_at < NOW()-INTERVAL '10 min'
//	    RETURNING event_id   ← non-empty iff insert or expired re-claim won
//	)
//	SELECT CASE
//	    WHEN (SELECT event_id FROM claim) IS NOT NULL THEN 'acquired'
//	    WHEN e.status = 'sent'                        THEN 'sent'
//	    ELSE                                               'active_processing'
//	END FROM processed_events e WHERE e.event_id = $1
//
// The CTE and outer SELECT run in the same statement — no TOCTOU gap.
type EventDeduplicator struct {
	pool *pgxpool.Pool
}

// NewEventDeduplicator creates a deduplicator backed by the given pool.
func NewEventDeduplicator(pool *pgxpool.Pool) *EventDeduplicator {
	return &EventDeduplicator{pool: pool}
}

// TryClaim implements port.Deduplicator.
func (d *EventDeduplicator) TryClaim(ctx context.Context, eventID, eventType string) (port.ClaimOutcome, error) {
	var outcome string
	err := d.pool.QueryRow(ctx, `
		WITH claim AS (
			INSERT INTO notification_svc.processed_events
				(event_id, event_type, status, processing_at, processed_at)
			VALUES ($1, $2, 'processing', NOW(), NOW())
			ON CONFLICT (event_id) DO UPDATE
				SET processing_at = NOW(),
				    status        = 'processing'
			WHERE notification_svc.processed_events.status      = 'processing'
			  AND notification_svc.processed_events.processing_at < NOW() - INTERVAL '10 minutes'
			RETURNING event_id
		)
		SELECT
			CASE
				WHEN (SELECT event_id FROM claim) IS NOT NULL THEN 'acquired'
				WHEN e.status = 'sent'                        THEN 'sent'
				ELSE                                               'active_processing'
			END
		FROM notification_svc.processed_events e
		WHERE e.event_id = $1`,
		eventID, eventType,
	).Scan(&outcome)
	if err != nil {
		// Fail open: proceed so the event is not silently dropped on DB errors.
		return port.ClaimAcquired, err
	}
	switch outcome {
	case "sent":
		return port.ClaimAlreadySent, nil
	case "active_processing":
		return port.ClaimActiveProcessing, nil
	default: // "acquired"
		return port.ClaimAcquired, nil
	}
}

// MarkSent implements port.Deduplicator.
func (d *EventDeduplicator) MarkSent(ctx context.Context, eventID string) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE notification_svc.processed_events SET status = 'sent' WHERE event_id = $1`,
		eventID,
	)
	return err
}

// Release implements port.Deduplicator.
func (d *EventDeduplicator) Release(ctx context.Context, eventID string) error {
	_, err := d.pool.Exec(ctx,
		`DELETE FROM notification_svc.processed_events WHERE event_id = $1 AND status = 'processing'`,
		eventID,
	)
	return err
}
