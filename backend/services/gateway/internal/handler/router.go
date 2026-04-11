package handler

import (
	"context"

	"github.com/go-chi/chi/v5"

	pkgauthz "github.com/Riku-KANO/ec-test/pkg/authz"
	pkgmw "github.com/Riku-KANO/ec-test/pkg/middleware"
	gwauthz "github.com/Riku-KANO/ec-test/services/gateway/internal/authz"
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

	// --- RBAC loader (talks to auth service /internal/authz/* with the
	// shared internal token). One Loader per process so the in-memory cache
	// is shared across requests.
	internalAuthClient := svc.Auth.WithHeader("X-Internal-Token", cfg.AuthInternalToken)
	rbacLoader := gwauthz.NewLoader(internalAuthClient)

	// --- Authenticated API routes ---
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(jwtMW.VerifyJWT)

		// Buyer routes (any authenticated user)
		buyer := NewBuyerHandler(svc)
		cart := NewCartHandler(svc)
		api.Route("/buyer", func(br chi.Router) {
			br.Get("/products", buyer.ListProducts)
			br.Get("/products/{slug}", buyer.GetProduct)
			br.Get("/search", buyer.SearchProducts)
			// Buyer purchases flow through the cart service. The direct
			// POST /orders endpoint was removed; use POST /cart/checkout.
			br.Get("/orders", buyer.ListOrders)
			br.Post("/events", buyer.TrackEvent)
			br.Get("/recommendations", buyer.GetRecommendations)
			br.Get("/plans", buyer.ListBuyerPlans)
			br.Get("/subscription", buyer.GetSubscription)
			br.Post("/subscription", buyer.Subscribe)

			// Cart routes — all buyer purchases start here.
			br.Route("/cart", func(cr chi.Router) {
				cr.Get("/", cart.Get)
				cr.Delete("/", cart.Clear)
				cr.Post("/items", cart.AddItem)
				cr.Put("/items/{skuId}", cart.UpdateItem)
				cr.Delete("/items/{skuId}", cart.RemoveItem)
				cr.Post("/checkout", cart.Checkout)
			})
		})

		// Seller routes (requires seller role at the JWT level)
		seller := NewSellerHandler(svc)
		sellerTeam := NewSellerTeamHandler(svc, rbacLoader)
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

			// Seller team management. Reads (List/Me) require any team
			// member; writes require owner. Two nested groups apply
			// different minimum roles.
			sr.Route("/team", func(tr chi.Router) {
				tr.Group(func(read chi.Router) {
					read.Use(pkgauthz.RequireSellerRole(rbacLoader, pkgauthz.SellerRoleMember))
					read.Get("/", sellerTeam.List)
					read.Get("/me", sellerTeam.Me)
				})
				tr.Group(func(write chi.Router) {
					write.Use(pkgauthz.RequireSellerRole(rbacLoader, pkgauthz.SellerRoleOwner))
					write.Post("/", sellerTeam.Add)
					write.Put("/{id}/role", sellerTeam.UpdateRole)
					write.Delete("/{id}", sellerTeam.Remove)
					write.Post("/transfer-ownership", sellerTeam.TransferOwnership)
				})
			})
		})

		// Admin routes (requires platform_admin role at the JWT level)
		admin := NewAdminHandler(svc)
		platformAdmin := NewPlatformAdminHandler(svc, rbacLoader)
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

			// Platform admin management. Reads (List/Me) need any
			// platform admin role; writes need super_admin.
			ar.Route("/admins", func(adr chi.Router) {
				adr.Group(func(read chi.Router) {
					read.Use(pkgauthz.RequirePlatformAdminRole(rbacLoader, pkgauthz.PlatformAdminRoleSupport))
					read.Get("/", platformAdmin.List)
					read.Get("/me", platformAdmin.Me)
				})
				adr.Group(func(write chi.Router) {
					write.Use(pkgauthz.RequirePlatformAdminRole(rbacLoader, pkgauthz.PlatformAdminRoleSuperAdmin))
					write.Post("/", platformAdmin.Grant)
					write.Put("/{id}/role", platformAdmin.UpdateRole)
					write.Delete("/{id}", platformAdmin.Revoke)
				})
			})

			// RBAC audit log (super_admin only).
			ar.Group(func(adr chi.Router) {
				adr.Use(pkgauthz.RequirePlatformAdminRole(rbacLoader, pkgauthz.PlatformAdminRoleSuperAdmin))
				adr.Get("/audit", platformAdmin.ListAudit)
			})
		})
	})

	return r
}
