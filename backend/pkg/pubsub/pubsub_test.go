package pubsub_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
)

func TestNewEvent(t *testing.T) {
	tid := uuid.New()
	data := map[string]string{"product_id": "p1"}
	before := time.Now().UTC()
	event := pubsub.NewEvent("order.created", tid, data)
	after := time.Now().UTC()

	if event.Type != "order.created" {
		t.Errorf("Type = %q, want %q", event.Type, "order.created")
	}
	if event.TenantID != tid.String() {
		t.Errorf("TenantID = %q, want %q", event.TenantID, tid.String())
	}
	if event.ID == "" {
		t.Error("ID should not be empty")
	}
	if _, err := uuid.Parse(event.ID); err != nil {
		t.Errorf("ID is not a valid UUID: %v", err)
	}
	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Errorf("Timestamp = %v, want between %v and %v", event.Timestamp, before, after)
	}
}

func TestEncode_Decode_RoundTrip(t *testing.T) {
	tid := uuid.New()
	original := pubsub.Event{
		ID:        uuid.New().String(),
		Type:      "product.updated",
		TenantID:  tid.String(),
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		Data:      map[string]any{"sku": "ABC-123"},
	}

	encoded, err := pubsub.Encode(original)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}

	decoded, err := pubsub.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, original.Type)
	}
	if decoded.TenantID != original.TenantID {
		t.Errorf("TenantID = %q, want %q", decoded.TenantID, original.TenantID)
	}
}

func TestEncode_ValidJSON(t *testing.T) {
	event := pubsub.NewEvent("test.event", uuid.New(), "simple-data")
	encoded, err := pubsub.Encode(event)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(encoded, &m); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if m["event_type"] != "test.event" {
		t.Errorf("event_type = %v, want %q", m["event_type"], "test.event")
	}
}

func TestDecode_InvalidJSON(t *testing.T) {
	_, err := pubsub.Decode([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDecode_EmptyJSON(t *testing.T) {
	event, err := pubsub.Decode([]byte("{}"))
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if event.ID != "" || event.Type != "" {
		t.Errorf("expected zero-value fields, got ID=%q Type=%q", event.ID, event.Type)
	}
}

// mockPublisher implements pubsub.Publisher for testing.
type mockPublisher struct {
	published []pubsub.Event
	err       error
}

func (m *mockPublisher) Publish(_ context.Context, _ string, event pubsub.Event) error {
	if m.err != nil {
		return m.err
	}
	m.published = append(m.published, event)
	return nil
}

func (m *mockPublisher) Close() error { return nil }

func TestPublishEvent_Success(t *testing.T) {
	pub := &mockPublisher{}
	tid := uuid.New()
	data := map[string]string{"order_id": "o1"}

	pubsub.PublishEvent(context.Background(), pub, tid, "order.created", "orders", data)

	if len(pub.published) != 1 {
		t.Fatalf("expected 1 event, got %d", len(pub.published))
	}
	event := pub.published[0]
	if event.Type != "order.created" {
		t.Errorf("Type = %q, want %q", event.Type, "order.created")
	}
	if event.TenantID != tid.String() {
		t.Errorf("TenantID = %q, want %q", event.TenantID, tid.String())
	}
}

func TestPublishEvent_NilPublisher(t *testing.T) {
	// Should be a no-op without panic.
	pubsub.PublishEvent(context.Background(), nil, uuid.New(), "test", "topic", nil)
}

func TestPublishEvent_Error(t *testing.T) {
	pub := &mockPublisher{err: errors.New("publish failed")}

	// Should not panic; just logs a warning.
	pubsub.PublishEvent(context.Background(), pub, uuid.New(), "test", "topic", nil)
}
