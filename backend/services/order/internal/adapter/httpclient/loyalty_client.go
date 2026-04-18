package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
	"github.com/Riku-KANO/ec-test/services/order/internal/port"
)

// LoyaltyClient is the order service's HTTP adapter for loyalty-svc.
// Implements port.PointReserver. Carries X-Internal-Token on every
// call. Phase 2 only shipped the /internal/earn endpoint on the
// loyalty service; the Reserve/Commit/Release endpoints land in a
// Phase 4 follow-up when loyalty gains the reservation flow.
type LoyaltyClient struct {
	baseURL       string
	internalToken string
	http          *http.Client
}

func NewLoyaltyClient(baseURL, internalToken string) *LoyaltyClient {
	return &LoyaltyClient{
		baseURL:       baseURL,
		internalToken: internalToken,
		http: &http.Client{
			Timeout:   5 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

type loyaltyReserveRequest struct {
	BuyerAuth0ID string `json:"buyer_auth0_id"`
	Amount       int64  `json:"amount"`
}

type reservePointsResponse struct {
	ReservationID string    `json:"reservation_id"`
	Amount        int64     `json:"amount"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// Reserve calls POST /internal/reservations on loyalty-svc. A 400
// with code=INSUFFICIENT_POINTS surfaces as domain.ErrInsufficientPoints
// so the checkout handler can emit the right buyer-facing error.
func (c *LoyaltyClient) Reserve(ctx context.Context, buyerAuth0ID string, amount int64) (*port.PointReservation, error) {
	body, err := c.doJSON(ctx, http.MethodPost, "/internal/reservations", loyaltyReserveRequest{
		BuyerAuth0ID: buyerAuth0ID,
		Amount:       amount,
	})
	if err != nil {
		return nil, err
	}
	var wire reservePointsResponse
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("decode reserve response: %w", err)
	}
	rid, err := uuid.Parse(wire.ReservationID)
	if err != nil {
		return nil, fmt.Errorf("parse reservation_id: %w", err)
	}
	return &port.PointReservation{
		ReservationID: rid,
		Amount:        wire.Amount,
		ExpiresAt:     wire.ExpiresAt,
	}, nil
}

type commitRequestBody struct {
	OrderID               string `json:"order_id"`
	StripePaymentIntentID string `json:"stripe_payment_intent_id"`
	PaidSubtotal          int64  `json:"paid_subtotal"`
	Currency              string `json:"currency"`
}

type commitPointsResponse struct {
	Redeemed         int64 `json:"redeemed"`
	Earned           int64 `json:"earned"`
	NewBalance       int64 `json:"new_balance"`
	AlreadyCommitted bool  `json:"already_committed"`
}

// Commit routes between the earn-only path and the reservation commit
// based on whether reservationID is set. The loyalty service treats
// the two call shapes as distinct endpoints so the ledger writes carry
// the correct idempotency source_type.
func (c *LoyaltyClient) Commit(ctx context.Context, reservationID *uuid.UUID, buyerAuth0ID, orderID string, stripePaymentIntentID string, paidSubtotal int64, currency string) (*port.PointCommitResult, error) {
	if reservationID == nil {
		return c.AwardEarn(ctx, buyerAuth0ID, orderID, paidSubtotal, currency)
	}
	body, err := c.doJSON(ctx, http.MethodPost,
		fmt.Sprintf("/internal/reservations/%s/commit", reservationID),
		commitRequestBody{
			OrderID:               orderID,
			StripePaymentIntentID: stripePaymentIntentID,
			PaidSubtotal:          paidSubtotal,
			Currency:              currency,
		})
	if err != nil {
		return nil, err
	}
	var wire commitPointsResponse
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("decode commit response: %w", err)
	}
	return &port.PointCommitResult{
		Redeemed:         wire.Redeemed,
		Earned:           wire.Earned,
		NewBalance:       wire.NewBalance,
		AlreadyCommitted: wire.AlreadyCommitted,
	}, nil
}

type releaseRequestBody struct {
	Reason string `json:"reason"`
}

func (c *LoyaltyClient) Release(ctx context.Context, reservationID uuid.UUID, reason string) error {
	_, err := c.doJSON(ctx, http.MethodPost,
		fmt.Sprintf("/internal/reservations/%s/release", reservationID),
		releaseRequestBody{Reason: reason})
	return err
}

// loyaltyErrorResponse mirrors loyalty-svc's error envelope so stable
// codes like INSUFFICIENT_POINTS can be preserved back to the buyer.
type loyaltyErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// doJSON is the shared helper for the reservation endpoints. It
// preserves the `code` field on 4xx errors so the order-svc error
// mapper can translate to domain sentinels (e.g. INSUFFICIENT_POINTS).
func (c *LoyaltyClient) doJSON(ctx context.Context, method, path string, body any) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal loyalty request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("loyalty %s: %w", path, err)
	}
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read loyalty response: %w", err)
	}
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return buf, nil
	}
	var env loyaltyErrorResponse
	_ = json.Unmarshal(buf, &env)
	return nil, translateLoyaltyError(resp.StatusCode, env)
}

// translateLoyaltyError maps stable codes to domain sentinels so
// checkout errors render with the intended buyer-facing message.
func translateLoyaltyError(status int, env loyaltyErrorResponse) error {
	switch env.Code {
	case "INSUFFICIENT_POINTS":
		return domain.ErrInsufficientPoints
	case "RESERVATION_TERMINAL":
		return fmt.Errorf("loyalty reservation no longer pending")
	}
	return fmt.Errorf("loyalty service returned %d: %s", status, env.Error)
}

type balanceResponse struct {
	Balance           int64 `json:"balance"`
	PendingRedemption int64 `json:"pending_redemption"`
	Spendable         int64 `json:"spendable"`
}

func (c *LoyaltyClient) GetBalance(ctx context.Context, buyerAuth0ID string) (*port.PointBalance, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/points/balance", nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("X-Internal-Token", c.internalToken)
	req.Header.Set("X-User-ID", buyerAuth0ID)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("loyalty get balance: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("loyalty get balance returned %s", resp.Status)
	}
	var wire balanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
		return nil, fmt.Errorf("decode balance: %w", err)
	}
	return &port.PointBalance{
		Balance:           wire.Balance,
		PendingRedemption: wire.PendingRedemption,
		Spendable:         wire.Spendable,
	}, nil
}

type awardEarnRequest struct {
	BuyerAuth0ID string `json:"buyer_auth0_id"`
	OrderID      string `json:"order_id"`
	PaidSubtotal int64  `json:"paid_subtotal"`
	Currency     string `json:"currency"`
}

type awardEarnResponse struct {
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	BalanceAfter  int64  `json:"balance_after"`
}

// AwardEarn calls POST /internal/earn on loyalty-svc. A 204 (zero-earn)
// returns a non-error (0, 0, 0) result so the caller can treat the
// award as idempotent success.
func (c *LoyaltyClient) AwardEarn(ctx context.Context, buyerAuth0ID, orderID string, paidSubtotal int64, currency string) (*port.PointCommitResult, error) {
	raw, err := json.Marshal(awardEarnRequest{
		BuyerAuth0ID: buyerAuth0ID,
		OrderID:      orderID,
		PaidSubtotal: paidSubtotal,
		Currency:     currency,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal earn request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/earn", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("loyalty award earn: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read earn response: %w", err)
		}
		var wire awardEarnResponse
		if err := json.Unmarshal(buf, &wire); err != nil {
			return nil, fmt.Errorf("decode earn: %w", err)
		}
		return &port.PointCommitResult{
			Redeemed:   0,
			Earned:     wire.Amount,
			NewBalance: wire.BalanceAfter,
		}, nil
	case http.StatusNoContent:
		// Below the minimum-earn threshold. Treat as successful no-op.
		return &port.PointCommitResult{}, nil
	default:
		return nil, fmt.Errorf("loyalty award earn returned %s", resp.Status)
	}
}
