package stripe

import (
	"fmt"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/transfer"
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
