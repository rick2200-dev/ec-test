package handler

import (
	"context"

	"github.com/go-chi/chi/v5"

	pkgmw "github.com/Riku-KANO/ec-test/pkg/middleware"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/config"
	gwmw "github.com/Riku-KANO/ec-test/services/gateway/internal/middleware"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// NewRouter builds and returns the top-level chi router with all middleware
// and route groups registered. ctx controls the lifetime of background tasks
// such as JWKS key refresh; cancel it during service shutdown.
func NewRouter(ctx context.Context, cfg config.Config, svc *proxy.Services) *chi.Mux {
	r := chi.NewRouter()

	// --- Global middleware ---
	r.Use(gwmw.RequestID)
	r.Use(pkgmw.Logger)
	r.Use(gwmw.Recovery)
	r.Use(gwmw.CORS(cfg.AllowedOrigins))

	// --- Health endpoints (no auth required) ---
	health := NewHealthHandler()
	r.Get("/healthz", health.Liveness)
	r.Get("/readyz", health.Readiness)

	// --- JWT middleware for authenticated routes ---
	jwtMW := pkgmw.NewJWTMiddleware(ctx, pkgmw.JWTConfig{
		Issuer:   cfg.JWTIssuer,
		Audience: cfg.JWTAudience,
		JWKSURL:  cfg.JWKSURL,
	})

	// --- Authenticated API routes ---
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(jwtMW.VerifyJWT)

		// Buyer routes (any authenticated user)
		buyer := NewBuyerHandler(svc)
		api.Route("/buyer", func(br chi.Router) {
			br.Get("/products", buyer.ListProducts)
			br.Get("/products/{slug}", buyer.GetProduct)
			br.Get("/search", buyer.SearchProducts)
			br.Post("/orders", buyer.CreateOrder)
			br.Get("/orders", buyer.ListOrders)
			br.Post("/events", buyer.TrackEvent)
			br.Get("/recommendations", buyer.GetRecommendations)
			br.Get("/plans", buyer.ListBuyerPlans)
			br.Get("/subscription", buyer.GetSubscription)
			br.Post("/subscription", buyer.Subscribe)
		})

		// Seller routes (requires seller role)
		seller := NewSellerHandler(svc)
		api.Route("/seller", func(sr chi.Router) {
			sr.Use(pkgmw.RequireRole("seller"))
			sr.Get("/products", seller.ListProducts)
			sr.Post("/products", seller.CreateProduct)
			sr.Put("/products/{id}", seller.UpdateProduct)
			sr.Get("/orders", seller.ListOrders)
			sr.Put("/orders/{id}/status", seller.UpdateOrderStatus)
			sr.Get("/inventory", seller.ListInventory)
			sr.Put("/inventory/{skuID}", seller.UpdateStock)
			sr.Get("/subscription", seller.GetSubscription)
			sr.Post("/subscription", seller.Subscribe)
			sr.Get("/plans", seller.ListPlans)
		})

		// Admin routes (requires platform_admin role)
		admin := NewAdminHandler(svc)
		api.Route("/admin", func(ar chi.Router) {
			ar.Use(pkgmw.RequireRole("platform_admin"))
			ar.Get("/tenants", admin.ListTenants)
			ar.Post("/tenants", admin.CreateTenant)
			ar.Get("/sellers", admin.ListSellers)
			ar.Put("/sellers/{id}/approve", admin.ApproveSeller)
			ar.Get("/plans", admin.ListPlans)
			ar.Post("/plans", admin.CreatePlan)
			ar.Put("/plans/{id}", admin.UpdatePlan)
			ar.Get("/buyer-plans", admin.ListBuyerPlans)
			ar.Post("/buyer-plans", admin.CreateBuyerPlan)
			ar.Put("/buyer-plans/{id}", admin.UpdateBuyerPlan)
		})
	})

	return r
}
