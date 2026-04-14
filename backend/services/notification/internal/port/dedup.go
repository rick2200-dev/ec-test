package port

import "context"

// ClaimOutcome is the tri-state result of a Deduplicator.TryClaim attempt.
type ClaimOutcome uint8

const (
	// ClaimAcquired means this caller owns the claim; proceed to send.
	ClaimAcquired ClaimOutcome = iota

	// ClaimAlreadySent means the event was previously processed successfully.
	// The caller must return nil (ack) without sending.
	ClaimAlreadySent

	// ClaimActiveProcessing means another goroutine/process holds an active
	// (non-expired) 'processing' claim. The caller must return a retryable error
	// so the message is nacked and Pub/Sub redelivers it. Returning nil here
	// would ack the message before the processing_at timeout expires, potentially
	// losing the notification if the active worker crashes before MarkSent.
	ClaimActiveProcessing
)

// Deduplicator is the driven port for notification deduplication.
// It implements a status-based inbox (processing → sent) backed by persistent
// storage (typically Postgres) to prevent duplicate sends under at-least-once
// Pub/Sub delivery.
//
// Required call sequence:
//
//  1. TryClaim → ClaimAlreadySent:      ack (return nil), skip send.
//  1. TryClaim → ClaimActiveProcessing: nack (return retryable error).
//  1. TryClaim → ClaimAcquired:         proceed to send.
//  2. Send side effect.
//  3a. Send succeeded → MarkSent → return nil.
//  3b. Send failed    → Release  → return error.
//
// Crash recovery: if the process dies between TryClaim and MarkSent/Release,
// the processing_at claim expires after 10 minutes. Until expiry, redeliveries
// see ClaimActiveProcessing and nack, keeping the message alive. After expiry
// a redelivery re-claims and retries the send.
type Deduplicator interface {
	// TryClaim atomically attempts to claim eventID for processing.
	// On DB error returns (ClaimAcquired, err) — caller should log and proceed
	// (fail open) rather than silently dropping the notification.
	TryClaim(ctx context.Context, eventID, eventType string) (ClaimOutcome, error)

	// MarkSent transitions the event to 'sent' after a successful send.
	// Future TryClaim calls return ClaimAlreadySent for the same event_id.
	MarkSent(ctx context.Context, eventID string) error

	// Release removes the 'processing' claim so the next Pub/Sub redelivery can
	// claim and retry. Must be called when the send side effect fails.
	Release(ctx context.Context, eventID string) error
}
