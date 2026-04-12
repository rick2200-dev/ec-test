package domain_test

import (
	"testing"

	domain "github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

func TestOrder_CanBeCancelled(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"pending is cancellable", domain.StatusPending, true},
		{"paid is cancellable", domain.StatusPaid, true},
		{"processing is cancellable", domain.StatusProcessing, true},
		{"shipped is not cancellable", domain.StatusShipped, false},
		{"delivered is not cancellable", domain.StatusDelivered, false},
		{"completed is not cancellable", domain.StatusCompleted, false},
		{"cancelled is not cancellable", domain.StatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &domain.Order{Status: tt.status}
			if got := o.CanBeCancelled(); got != tt.want {
				t.Errorf("Order{Status: %q}.CanBeCancelled() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestOrder_CanBeCancelled_EmptyStatus(t *testing.T) {
	o := &domain.Order{Status: ""}
	if o.CanBeCancelled() {
		t.Error("Order with empty status should not be cancellable")
	}
}

func TestOrderStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"StatusPending", domain.StatusPending, "pending"},
		{"StatusPaid", domain.StatusPaid, "paid"},
		{"StatusProcessing", domain.StatusProcessing, "processing"},
		{"StatusShipped", domain.StatusShipped, "shipped"},
		{"StatusDelivered", domain.StatusDelivered, "delivered"},
		{"StatusCompleted", domain.StatusCompleted, "completed"},
		{"StatusCancelled", domain.StatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.want)
			}
		})
	}
}

func TestPayoutStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"PayoutStatusPending", domain.PayoutStatusPending, "pending"},
		{"PayoutStatusCompleted", domain.PayoutStatusCompleted, "completed"},
		{"PayoutStatusFailed", domain.PayoutStatusFailed, "failed"},
		{"PayoutStatusReversed", domain.PayoutStatusReversed, "reversed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.want)
			}
		})
	}
}

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"EventTypeOrderCreated", domain.EventTypeOrderCreated, "order.created"},
		{"EventTypeOrderPaid", domain.EventTypeOrderPaid, "order.paid"},
		{"EventTypeOrderShipped", domain.EventTypeOrderShipped, "order.shipped"},
		{"EventTypePayoutFailed", domain.EventTypePayoutFailed, "payout.failed"},
		{"EventTypePayoutCompleted", domain.EventTypePayoutCompleted, "payout.completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.want)
			}
		})
	}
}

func TestDomainErrors_NotNil(t *testing.T) {
	errs := []struct {
		name string
		err  error
	}{
		{"ErrOrderNotFound", domain.ErrOrderNotFound},
		{"ErrEmptyOrder", domain.ErrEmptyOrder},
		{"ErrBuyerRequired", domain.ErrBuyerRequired},
		{"ErrInvalidQuantity", domain.ErrInvalidQuantity},
		{"ErrInvalidOrderStatus", domain.ErrInvalidOrderStatus},
		{"ErrOrderNotPending", domain.ErrOrderNotPending},
	}

	for _, tt := range errs {
		t.Run(tt.name+" is not nil", func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
		})
	}

	// Verify all error messages are distinct.
	seen := make(map[string]string) // message → error name
	for _, tt := range errs {
		msg := tt.err.Error()
		if msg == "" {
			t.Errorf("%s has an empty error message", tt.name)
			continue
		}
		if prev, ok := seen[msg]; ok {
			t.Errorf("%s and %s have the same error message: %q", prev, tt.name, msg)
		}
		seen[msg] = tt.name
	}
}
