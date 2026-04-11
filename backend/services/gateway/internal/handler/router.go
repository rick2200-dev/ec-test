package handler

import (
	"context"

	"github.com/go-chi/chi/v5"

	pkgauthz "github.com/Riku-KANO/ec-test/pkg/authz"
	pkgmw "github.com/Riku-KANO/ec-test/pkg/middleware"
	pkgredis "github.com/Riku-KANO/ec-test/pkg/redis"
	gwauthz "github.com/Riku-KANO/ec-test/services/gateway/internal/authz"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/config"
	gwmw "github.com/Riku-KANO/ec-test/services/gateway/internal/middleware"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/middleware/apitoken"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// NewRouter builds and returns the top-level chi router with all middleware
// and route groups registered. ctx controls the lifetime of background tasks
// such as JWKS key refresh; cancel it during service shutdown.
//
// redisClient is used by the API-token rate limiter. A nil client is
// treated as a startup error in cmd/server/main.go (we fail-fast rather
// than silently degrade) so by the time this function runs it is
// guaranteed to be non-nil.
func NewRouter(ctx context.Context, cfg config.Config, svc *proxy.Services, redisClient *pkgredis.Client) *chi.Mux {
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

	// --- API token loader + rate limiter. Constructed once per process
	// so the in-memory lookup cache and the limiter's compiled Lua
	// script are shared across requests.
	apiTokenLoader := apitoken.NewLoader(internalAuthClient, cfg.APITokenCacheTTL)
	apiTokenLimiter := apitoken.NewLimiter(redisClient, cfg.APITokenRPSDefault, cfg.APITokenBurstDefault)

	// --- Authenticated API routes ---
	//
	// Intentionally NO jwtMW.VerifyJWT at the /api/v1 level: the /seller
	// subtree accepts EITHER a JWT (dashboard UI) OR an API token (sk_live_*)
	// via apitoken.OrJWT, and a global jwtMW.Use would 401 token traffic
	// before apitoken.OrJWT could see it. JWT verification is re-applied
	// per subtree below so /buyer and /admin still get the same treatment
	// they had previously.
	r.Route("/api/v1", func(api chi.Router) {
		// Buyer routes (any authenticated user) — JWT only.
		buyer := NewBuyerHandler(svc)
		cart := NewCartHandler(svc)
		inquiry := NewInquiryHandler(svc)
		api.Route("/buyer", func(br chi.Router) {
			br.Use(jwtMW.VerifyJWT)
			br.Get("/products", buyer.ListProducts)
			br.Get("/products/{slug}", buyer.GetProduct)
			br.Get("/search", buyer.SearchProducts)
			// Buyer purchases flow through the cart service. The direct
			// POST /orders endpoint was removed; use POST /cart/checkout.
			br.Get("/orders", buyer.ListOrders)
			br.Get("/orders/{id}", buyer.GetOrder)
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

			// Buyer↔seller inquiry threads (per purchased SKU).
			br.Route("/inquiries", func(ir chi.Router) {
				ir.Get("/", inquiry.BuyerList)
				ir.Post("/", inquiry.BuyerCreate)
				ir.Get("/{id}", inquiry.BuyerGet)
				ir.Post("/{id}/messages", inquiry.BuyerPostMessage)
				ir.Post("/{id}/read", inquiry.BuyerMarkRead)
			})
		})

		// Seller routes — JWT (dashboard) OR API token (sk_live_*).
		//
		// Middleware chain, applied in order:
		//   1. apitoken.OrJWT            — resolve auth, inject tenant
		//                                  context + (optional) apitoken
		//                                  context onto r.Context.
		//   2. pkgmw.RequireRole("seller") — rejects JWTs without the
		//                                  seller role. API-token requests
		//                                  synthesize roles=["seller"]
		//                                  inside OrJWT so they always
		//                                  pass this gate.
		//   3. apitokenLimiter.Middleware — sliding-window 429 per token.
		//                                  JWT requests are a no-op here
		//                                  because there is no apitoken
		//                                  context to key on.
		//
		// Data endpoints then attach a per-route apitoken.RequireScope.
		// Scope checks are a no-op for JWT callers (the dashboard's
		// existing role middleware is what gates them); they only fire
		// for API-token calls.
		seller := NewSellerHandler(svc)
		sellerTeam := NewSellerTeamHandler(svc, rbacLoader)
		sellerAPITokens := NewSellerAPITokenHandler(svc, apiTokenLoader, cfg.APITokenPrefix)
		api.Route("/seller", func(sr chi.Router) {
			sr.Use(apitoken.OrJWT(jwtMW, apiTokenLoader, cfg.APITokenPrefix))
			sr.Use(pkgmw.RequireRole("seller"))
			sr.Use(apiTokenLimiter.Middleware)

			// Scope-gated data endpoints. Each resource×action pair maps
			// to one fine-grained scope; see services/auth/internal/domain/api_token.go
			// for the closed vocabulary.
			sr.With(apitoken.RequireScope("products:read")).Get("/products", seller.ListProducts)
			sr.With(apitoken.RequireScope("products:write")).Post("/products", seller.CreateProduct)
			sr.With(apitoken.RequireScope("products:write")).Put("/products/{id}", seller.UpdateProduct)
			sr.With(apitoken.RequireScope("orders:read")).Get("/orders", seller.ListOrders)
			sr.With(apitoken.RequireScope("orders:write")).Put("/orders/{id}/status", seller.UpdateOrderStatus)
			sr.With(apitoken.RequireScope("inventory:read")).Get("/inventory", seller.ListInventory)
			sr.With(apitoken.RequireScope("inventory:write")).Put("/inventory/{skuID}", seller.UpdateStock)

			// UI-only subtree: billing, team management, inquiry threads,
			// and api-token management itself. apitoken.Block 403s any
			// API-token request that reaches here — preventing the
			// privilege-escalation class where a live token could rewrite
			// its own role, issue a stronger sibling, or modify billing.
			//
			// Inquiries are parked here (not scope-gated) because v1 has
			// no inquiry:* scope vocabulary yet; we prefer "blocked until
			// a scope is defined" over "accidentally exposed".
			sr.Group(func(ui chi.Router) {
				ui.Use(apitoken.Block)

				ui.Get("/subscription", seller.GetSubscription)
				ui.Post("/subscription", seller.Subscribe)
				ui.Get("/plans", seller.ListPlans)

				// Inquiry threads (seller view). Requires seller team
				// membership in the same RBAC tier as team reads.
				ui.Route("/inquiries", func(ir chi.Router) {
					ir.Use(pkgauthz.RequireSellerRole(rbacLoader, pkgauthz.SellerRoleMember))
					ir.Get("/", inquiry.SellerList)
					ir.Get("/{id}", inquiry.SellerGet)
					ir.Post("/{id}/messages", inquiry.SellerPostMessage)
					ir.Post("/{id}/read", inquiry.SellerMarkRead)
					ir.Post("/{id}/close", inquiry.SellerClose)
				})

				// Seller team management. Reads (List/Me) require any team
				// member; writes require owner. Two nested groups apply
				// different minimum roles.
				ui.Route("/team", func(tr chi.Router) {
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

				// API token management — owner-only, blocked for token auth.
				ui.Route("/api-tokens", func(tr chi.Router) {
					tr.Use(pkgauthz.RequireSellerRole(rbacLoader, pkgauthz.SellerRoleOwner))
					tr.Get("/", sellerAPITokens.List)
					tr.Post("/", sellerAPITokens.Issue)
					tr.Get("/{id}", sellerAPITokens.Get)
					tr.Delete("/{id}", sellerAPITokens.Revoke)
				})
			})
		})

		// Admin routes (requires platform_admin role at the JWT level)
		admin := NewAdminHandler(svc)
		platformAdmin := NewPlatformAdminHandler(svc, rbacLoader)
		api.Route("/admin", func(ar chi.Router) {
			ar.Use(jwtMW.VerifyJWT)
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
