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

// CouponClient is the order service's HTTP adapter for coupon-svc. It
// implements port.CouponReserver. All calls go through the internal
// routes and carry X-Internal-Token.
type CouponClient struct {
	baseURL       string
	internalToken string
	http          *http.Client
}

func NewCouponClient(baseURL, internalToken string) *CouponClient {
	return &CouponClient{
		baseURL:       baseURL,
		internalToken: internalToken,
		http: &http.Client{
			Timeout:   5 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

type sellerSubtotalWire struct {
	SellerID string `json:"seller_id"`
	Subtotal int64  `json:"subtotal"`
}

type reserveRequest struct {
	Code            string               `json:"code"`
	BuyerAuth0ID    string               `json:"buyer_auth0_id"`
	Currency        string               `json:"currency"`
	SellerSubtotals []sellerSubtotalWire `json:"seller_subtotals"`
}

type reserveResponse struct {
	ReservationID      string    `json:"reservation_id"`
	CouponID           string    `json:"coupon_id"`
	IssuerType         string    `json:"issuer_type"`
	ApplicableSellerID string    `json:"applicable_seller_id"`
	DiscountAmount     int64     `json:"discount_amount"`
	Currency           string    `json:"currency"`
	ExpiresAt          time.Time `json:"expires_at"`
}

// Reserve calls POST /internal/reservations on coupon-svc. A 4xx from
// the coupon service surfaces as a domain-level error so the
// checkout handler can return 400 with a stable error code.
func (c *CouponClient) Reserve(ctx context.Context, code, buyerAuth0ID string, sellerSubtotals []port.SellerSubtotal, currency string) (*port.CouponReservation, error) {
	wire := reserveRequest{
		Code:         code,
		BuyerAuth0ID: buyerAuth0ID,
		Currency:     currency,
	}
	for _, s := range sellerSubtotals {
		wire.SellerSubtotals = append(wire.SellerSubtotals, sellerSubtotalWire{
			SellerID: s.SellerID.String(),
			Subtotal: s.Subtotal,
		})
	}
	body, err := c.postJSON(ctx, "/internal/reservations", wire)
	if err != nil {
		return nil, err
	}
	var resp reserveResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode reserve response: %w", err)
	}
	rid, err := uuid.Parse(resp.ReservationID)
	if err != nil {
		return nil, fmt.Errorf("parse reservation_id: %w", err)
	}
	cid, err := uuid.Parse(resp.CouponID)
	if err != nil {
		return nil, fmt.Errorf("parse coupon_id: %w", err)
	}
	out := &port.CouponReservation{
		ReservationID:  rid,
		CouponID:       cid,
		IssuerType:     resp.IssuerType,
		DiscountAmount: resp.DiscountAmount,
		Currency:       resp.Currency,
		ExpiresAt:      resp.ExpiresAt,
	}
	if resp.ApplicableSellerID != "" {
		sid, err := uuid.Parse(resp.ApplicableSellerID)
		if err != nil {
			return nil, fmt.Errorf("parse applicable_seller_id: %w", err)
		}
		out.ApplicableSellerID = &sid
	}
	return out, nil
}

type commitRequest struct {
	OrderID               string `json:"order_id"`
	StripePaymentIntentID string `json:"stripe_payment_intent_id"`
}

type commitResponse struct {
	RedemptionID     string `json:"redemption_id"`
	AlreadyCommitted bool   `json:"already_committed"`
}

func (c *CouponClient) Commit(ctx context.Context, reservationID, orderID uuid.UUID, stripePaymentIntentID string) (*port.CouponCommitResult, error) {
	body, err := c.postJSON(ctx,
		fmt.Sprintf("/internal/reservations/%s/commit", reservationID),
		commitRequest{OrderID: orderID.String(), StripePaymentIntentID: stripePaymentIntentID},
	)
	if err != nil {
		return nil, err
	}
	var resp commitResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode commit response: %w", err)
	}
	rid, err := uuid.Parse(resp.RedemptionID)
	if err != nil {
		return nil, fmt.Errorf("parse redemption_id: %w", err)
	}
	return &port.CouponCommitResult{RedemptionID: rid, AlreadyCommitted: resp.AlreadyCommitted}, nil
}

type releaseRequest struct {
	Reason string `json:"reason"`
}

func (c *CouponClient) Release(ctx context.Context, reservationID uuid.UUID, reason string) error {
	_, err := c.postJSON(ctx,
		fmt.Sprintf("/internal/reservations/%s/release", reservationID),
		releaseRequest{Reason: reason},
	)
	return err
}

// couponErrorResponse mirrors the coupon service's error envelope so we
// can preserve stable `code` values (e.g. COUPON_EXPIRED) end-to-end.
type couponErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// postJSON marshals body, calls path on coupon-svc, and returns the
// raw response body on 2xx. 4xx responses are translated to domain
// errors using the error Code so the checkout handler can surface the
// right message to the frontend.
func (c *CouponClient) postJSON(ctx context.Context, path string, body any) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal coupon request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("coupon %s: %w", path, err)
	}
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read coupon response: %w", err)
	}
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return buf, nil
	}

	// Preserve the domain-level signal so order-svc can distinguish
	// "coupon expired" from "internal error" and surface the right
	// 4xx to the buyer.
	var env couponErrorResponse
	_ = json.Unmarshal(buf, &env)
	return nil, translateCouponError(resp.StatusCode, env)
}

// translateCouponError maps well-known coupon error codes to domain
// sentinels. Unknown codes surface as a generic error so the order
// service still fails the checkout (never proceed with a silent
// "no discount"-type response).
func translateCouponError(status int, env couponErrorResponse) error {
	switch env.Code {
	case "COUPON_NOT_FOUND":
		return domain.ErrCouponNotFound
	case "COUPON_EXPIRED":
		return domain.ErrCouponExpired
	case "COUPON_REVOKED":
		return domain.ErrCouponRevoked
	case "COUPON_NOT_STARTED":
		return domain.ErrCouponNotStarted
	case "COUPON_USAGE_LIMIT":
		return domain.ErrCouponUsageLimitHit
	case "COUPON_PER_USER_LIMIT":
		return domain.ErrCouponPerUserLimit
	case "COUPON_MIN_ORDER":
		return domain.ErrCouponMinOrderNotMet
	case "COUPON_CURRENCY":
		return domain.ErrCouponCurrencyMismatch
	}
	return fmt.Errorf("coupon service returned %d: %s", status, env.Error)
}
