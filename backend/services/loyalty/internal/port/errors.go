package port

import "errors"

// Adapter-layer sentinel errors. They don't cross service boundaries —
// app maps them into domain errors before returning to handlers.
var (
	// ErrDuplicateIdempotencyKey indicates an Insert lost to an earlier
	// row with the same (source_type, source_id, type). App treats this
	// as "already committed" and returns the existing row's data.
	ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")

	// ErrOptimisticLock indicates ApplyDelta raced another writer. App
	// retries a bounded number of times.
	ErrOptimisticLock = errors.New("optimistic lock contention")
)
