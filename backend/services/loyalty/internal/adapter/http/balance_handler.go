package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/domain"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/port"
)

// BalanceHandler owns the buyer-facing point endpoints. The gateway
// routes /api/v1/buyer/points/* here; the internal-token middleware
// mounted in main.go keeps direct callers out.
type BalanceHandler struct {
	svc port.LoyaltyUseCase
}

func NewBalanceHandler(svc port.LoyaltyUseCase) *BalanceHandler {
	return &BalanceHandler{svc: svc}
}

// Routes mounts the buyer loyalty endpoints. Gateway forwards
// /api/v1/buyer/points/* here, and supplies the buyer_auth0_id via the
// `X-User-ID` header after JWT verification.
func (h *BalanceHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/balance", h.GetBalance)
	r.Get("/transactions", h.ListTransactions)
	// Also support the admin-addressed variant for the admin app.
	r.Get("/buyers/{buyer_auth0_id}/balance", h.AdminGetBalance)
	r.Get("/buyers/{buyer_auth0_id}/transactions", h.AdminListTransactions)
	return r
}

type balanceResponse struct {
	BuyerAuth0ID      string    `json:"buyer_auth0_id"`
	Balance           int64     `json:"balance"`
	PendingRedemption int64     `json:"pending_redemption"`
	Spendable         int64     `json:"spendable"`
	LifetimeEarned    int64     `json:"lifetime_earned"`
	LifetimeRedeemed  int64     `json:"lifetime_redeemed"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func toBalanceResponse(a *domain.Account) balanceResponse {
	return balanceResponse{
		BuyerAuth0ID:      a.BuyerAuth0ID,
		Balance:           a.Balance,
		PendingRedemption: a.PendingRedemption,
		Spendable:         a.Spendable(),
		LifetimeEarned:    a.LifetimeEarned,
		LifetimeRedeemed:  a.LifetimeRedeemed,
		UpdatedAt:         a.UpdatedAt,
	}
}

func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	buyer := r.Header.Get("X-User-ID")
	if buyer == "" {
		httputil.Error(w, apperrors.Unauthorized("missing X-User-ID"))
		return
	}
	acct, err := h.svc.GetBalance(r.Context(), buyer)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, toBalanceResponse(acct))
}

func (h *BalanceHandler) AdminGetBalance(w http.ResponseWriter, r *http.Request) {
	buyer := chi.URLParam(r, "buyer_auth0_id")
	if buyer == "" {
		httputil.Error(w, apperrors.BadRequest("buyer_auth0_id is required"))
		return
	}
	acct, err := h.svc.GetBalance(r.Context(), buyer)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, toBalanceResponse(acct))
}

type transactionListResponse struct {
	Transactions []domain.Transaction `json:"transactions"`
	Total        int                  `json:"total"`
}

func (h *BalanceHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	buyer := r.Header.Get("X-User-ID")
	if buyer == "" {
		httputil.Error(w, apperrors.Unauthorized("missing X-User-ID"))
		return
	}
	h.list(w, r, buyer)
}

func (h *BalanceHandler) AdminListTransactions(w http.ResponseWriter, r *http.Request) {
	buyer := chi.URLParam(r, "buyer_auth0_id")
	if buyer == "" {
		httputil.Error(w, apperrors.BadRequest("buyer_auth0_id is required"))
		return
	}
	h.list(w, r, buyer)
}

func (h *BalanceHandler) list(w http.ResponseWriter, r *http.Request, buyer string) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	txs, total, err := h.svc.ListTransactions(r.Context(), buyer, limit, offset)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	if txs == nil {
		txs = []domain.Transaction{}
	}
	httputil.JSON(w, http.StatusOK, transactionListResponse{Transactions: txs, Total: total})
}
