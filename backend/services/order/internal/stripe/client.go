package stripe

import (
	"fmt"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/refund"
	"github.com/stripe/stripe-go/v82/transfer"
	"github.com/stripe/stripe-go/v82/transferreversal"
)

// Client wraps the Stripe API for payment and transfer operations.
type Client struct {
	secretKey string
}

// NewClient creates a new Stripe client with the given secret key.
func NewClient(secretKey string) *Client {
	gostripe.Key = secretKey
	return &Client{secretKey: secretKey}
}

// CreatePaymentIntent creates a Stripe PaymentIntent with a connected account as the destination.
//
// Deprecated: this is the Destination Charges model which supports only one
// seller per PaymentIntent. Multi-seller checkouts use CreatePlatformPaymentIntent
// + CreateTransfer (Separate Charges and Transfers). Retained for reference.
func (c *Client) CreatePaymentIntent(amount int64, currency string, sellerStripeAccountID string, metadata map[string]string) (paymentIntentID, clientSecret string, err error) {
	params := &gostripe.PaymentIntentParams{
		Amount:   gostripe.Int64(amount),
		Currency: gostripe.String(currency),
		TransferData: &gostripe.PaymentIntentTransferDataParams{
			Destination: gostripe.String(sellerStripeAccountID),
		},
	}

	for k, v := range metadata {
		params.AddMetadata(k, v)
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return "", "", fmt.Errorf("create payment intent: %w", err)
	}

	return pi.ID, pi.ClientSecret, nil
}

// CreatePlatformPaymentIntent creates a Stripe PaymentIntent that charges the
// platform account (no TransferData). This is the Separate Charges and
// Transfers model: funds land on the platform first, and per-seller Transfers
// are created later by the webhook handler once payment succeeds. This is
// required for multi-seller checkouts where one PaymentIntent spans N sellers.
func (c *Client) CreatePlatformPaymentIntent(amount int64, currency string, metadata map[string]string) (paymentIntentID, clientSecret string, err error) {
	params := &gostripe.PaymentIntentParams{
		Amount:   gostripe.Int64(amount),
		Currency: gostripe.String(currency),
	}

	for k, v := range metadata {
		params.AddMetadata(k, v)
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return "", "", fmt.Errorf("create platform payment intent: %w", err)
	}

	return pi.ID, pi.ClientSecret, nil
}

// CreateTransfer creates a Stripe Transfer to a connected account.
func (c *Client) CreateTransfer(amount int64, currency string, sellerStripeAccountID string, paymentIntentID string) (transferID string, err error) {
	params := &gostripe.TransferParams{
		Amount:            gostripe.Int64(amount),
		Currency:          gostripe.String(currency),
		Destination:       gostripe.String(sellerStripeAccountID),
		SourceTransaction: gostripe.String(paymentIntentID),
	}

	t, err := transfer.New(params)
	if err != nil {
		return "", fmt.Errorf("create transfer: %w", err)
	}

	return t.ID, nil
}

// CreateRefund issues a partial or full refund on a PaymentIntent for the
// given amount. Used by the order-cancellation flow: when a seller approves
// a cancellation request, we refund `order.TotalAmount` against the shared
// PaymentIntent. Because a single PaymentIntent may cover multiple orders
// (multi-seller checkout), this is almost always a partial refund.
//
// The idempotency key MUST be deterministic per (request, action) so that
// safe retries never produce duplicate refunds. Callers pass a key shaped
// like "cancellation:<request_id>:refund".
//
// `reverse_transfer` is intentionally left false here; the caller reverses
// transfers explicitly via ReverseTransfer so we can track each reversal id
// per payout row in the DB.
func (c *Client) CreateRefund(paymentIntentID string, amount int64, idempotencyKey string) (refundID string, err error) {
	params := &gostripe.RefundParams{
		PaymentIntent: gostripe.String(paymentIntentID),
		Amount:        gostripe.Int64(amount),
	}
	params.SetIdempotencyKey(idempotencyKey)

	r, err := refund.New(params)
	if err != nil {
		return "", fmt.Errorf("create refund: %w", err)
	}

	return r.ID, nil
}

// ReverseTransfer issues a Transfer Reversal against an existing Transfer,
// pulling the given amount back from the connected seller account into the
// platform balance. Used after CreateRefund during an approved cancellation
// so the seller's payout is undone.
//
// Idempotency key format: "cancellation:<request_id>:reverse:<payout_id>".
// Distinct keys per payout allow multi-payout orders to be retried safely
// without double-reversing a transfer that already succeeded on a previous
// attempt.
func (c *Client) ReverseTransfer(transferID string, amount int64, idempotencyKey string) (reversalID string, err error) {
	params := &gostripe.TransferReversalParams{
		ID:     gostripe.String(transferID),
		Amount: gostripe.Int64(amount),
	}
	params.SetIdempotencyKey(idempotencyKey)

	rev, err := transferreversal.New(params)
	if err != nil {
		return "", fmt.Errorf("reverse transfer: %w", err)
	}

	return rev.ID, nil
}
