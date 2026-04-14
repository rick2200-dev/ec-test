package pubsub

import (
	"fmt"
	"time"
)

// RetryAfterError wraps a handler error and signals to the Subscriber that
// the message should be nacked after a delay rather than immediately.
//
// This prevents tight nack/redeliver loops when the handler knows it cannot
// proceed yet — for example, when a concurrent worker holds an active dedup
// claim that will expire after N minutes. Instead of hammering Pub/Sub with
// immediate nacks, the handler returns RetryAfter(delay, cause) and the
// subscriber sleeps for 'delay' (while holding the ack deadline open via the
// GCP client library's automatic extension) before calling msg.Nack().
//
// Note: this is distinct from the Pub/Sub subscription-level retry policy
// (min/max backoff). RetryAfterError adds a handler-controlled floor, while
// the subscription retry policy adds broker-controlled backoff on top.
type RetryAfterError struct {
	Delay time.Duration
	Cause error
}

func (e *RetryAfterError) Error() string {
	return fmt.Sprintf("retry after %s: %v", e.Delay, e.Cause)
}

func (e *RetryAfterError) Unwrap() error { return e.Cause }

// RetryAfter constructs a RetryAfterError. Use in handlers that should not be
// retried immediately:
//
//	return pubsub.RetryAfter(30*time.Second,
//	    fmt.Errorf("event %s is held by another worker", eventID))
func RetryAfter(delay time.Duration, cause error) *RetryAfterError {
	return &RetryAfterError{Delay: delay, Cause: cause}
}
