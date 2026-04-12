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

func TestEncode_NonSerializableData(t *testing.T) {
	event := pubsub.Event{
		ID:        uuid.New().String(),
		Type:      "test.event",
		TenantID:  uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Data:      make(chan int), // channels cannot be JSON-marshaled
	}

	_, err := pubsub.Encode(event)
	if err == nil {
		t.Fatal("expected error when encoding non-serializable data (chan), got nil")
	}
}

func TestDecode_PartialJSON(t *testing.T) {
	// JSON with only the "event_type" field populated.
	partial := []byte(`{"event_type":"test"}`)

	event, err := pubsub.Decode(partial)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if event.Type != "test" {
		t.Errorf("Type = %q, want %q", event.Type, "test")
	}
	// Missing fields should have their zero values.
	if event.ID != "" {
		t.Errorf("ID = %q, want empty string", event.ID)
	}
	if event.TenantID != "" {
		t.Errorf("TenantID = %q, want empty string", event.TenantID)
	}
	if !event.Timestamp.IsZero() {
		t.Errorf("Timestamp = %v, want zero time", event.Timestamp)
	}
	if event.Data != nil {
		t.Errorf("Data = %v, want nil", event.Data)
	}
}

func TestDecode_ExtraFields(t *testing.T) {
	// JSON with all known fields plus extra unknown ones.
	raw := []byte(`{
		"event_id": "abc-123",
		"event_type": "order.shipped",
		"tenant_id": "tenant-1",
		"timestamp": "2025-01-15T10:30:00Z",
		"data": {"key": "value"},
		"unknown_field": "should be ignored",
		"another_extra": 42
	}`)

	event, err := pubsub.Decode(raw)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if event.ID != "abc-123" {
		t.Errorf("ID = %q, want %q", event.ID, "abc-123")
	}
	if event.Type != "order.shipped" {
		t.Errorf("Type = %q, want %q", event.Type, "order.shipped")
	}
	if event.TenantID != "tenant-1" {
		t.Errorf("TenantID = %q, want %q", event.TenantID, "tenant-1")
	}
}

func TestNewEvent_FieldsPopulated(t *testing.T) {
	tid := uuid.New()
	data := map[string]int{"quantity": 5}

	before := time.Now().UTC()
	event := pubsub.NewEvent("inventory.updated", tid, data)
	after := time.Now().UTC()

	// ID must be a valid, non-nil UUID.
	if event.ID == "" {
		t.Fatal("ID should not be empty")
	}
	parsed, err := uuid.Parse(event.ID)
	if err != nil {
		t.Fatalf("ID is not a valid UUID: %v", err)
	}
	if parsed == uuid.Nil {
		t.Error("ID should not be the nil UUID")
	}

	// Type must match.
	if event.Type != "inventory.updated" {
		t.Errorf("Type = %q, want %q", event.Type, "inventory.updated")
	}

	// TenantID must match the provided UUID string.
	if event.TenantID != tid.String() {
		t.Errorf("TenantID = %q, want %q", event.TenantID, tid.String())
	}

	// Timestamp must be recent (between before and after).
	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Errorf("Timestamp = %v, want between %v and %v", event.Timestamp, before, after)
	}

	// Data must be set.
	if event.Data == nil {
		t.Error("Data should not be nil")
	}
	m, ok := event.Data.(map[string]int)
	if !ok {
		t.Fatalf("Data type = %T, want map[string]int", event.Data)
	}
	if m["quantity"] != 5 {
		t.Errorf("Data[\"quantity\"] = %d, want 5", m["quantity"])
	}
}

func TestEncode_Decode_PreservesAllFields(t *testing.T) {
	type address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}
	type orderData struct {
		OrderID  string  `json:"order_id"`
		Amount   float64 `json:"amount"`
		Shipping address `json:"shipping"`
	}

	tid := uuid.New()
	ts := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	original := pubsub.Event{
		ID:        uuid.New().String(),
		Type:      "order.completed",
		TenantID:  tid.String(),
		Timestamp: ts,
		Data: orderData{
			OrderID: "ord-999",
			Amount:  149.99,
			Shipping: address{
				Street: "123 Main St",
				City:   "Tokyo",
			},
		},
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
	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", decoded.Timestamp, original.Timestamp)
	}

	// Data round-trips as map[string]any via JSON; compare by normalizing
	// both sides through marshal-then-unmarshal into map[string]any to
	// avoid key-ordering differences.
	normalize := func(v any) (map[string]any, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, err
		}
		return m, nil
	}

	origMap, err := normalize(original.Data)
	if err != nil {
		t.Fatalf("normalize original data: %v", err)
	}
	decMap, err := normalize(decoded.Data)
	if err != nil {
		t.Fatalf("normalize decoded data: %v", err)
	}

	// Compare top-level and nested values.
	origBytes, _ := json.Marshal(origMap)
	decBytes, _ := json.Marshal(decMap)
	if string(origBytes) != string(decBytes) {
		t.Errorf("Data mismatch:\n  got:  %s\n  want: %s", decBytes, origBytes)
	}
}

func TestPublishEvent_BuildsCorrectEvent(t *testing.T) {
	pub := &mockPublisher{}
	tid := uuid.New()
	data := map[string]string{"item": "widget"}

	pubsub.PublishEvent(context.Background(), pub, tid, "cart.added", "cart-events", data)

	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(pub.published))
	}

	event := pub.published[0]

	// Verify the event type matches what was passed to PublishEvent.
	if event.Type != "cart.added" {
		t.Errorf("Type = %q, want %q", event.Type, "cart.added")
	}

	// Verify TenantID matches.
	if event.TenantID != tid.String() {
		t.Errorf("TenantID = %q, want %q", event.TenantID, tid.String())
	}

	// Verify ID is a valid UUID.
	if event.ID == "" {
		t.Fatal("ID should not be empty")
	}
	if _, err := uuid.Parse(event.ID); err != nil {
		t.Errorf("ID is not a valid UUID: %v", err)
	}

	// Verify Timestamp is recent (within the last second).
	if time.Since(event.Timestamp) > time.Second {
		t.Errorf("Timestamp = %v, expected within the last second", event.Timestamp)
	}

	// Verify Data is populated.
	if event.Data == nil {
		t.Error("Data should not be nil")
	}
}
