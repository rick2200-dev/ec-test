package proxy

import (
	catalogv1 "github.com/Riku-KANO/ec-test/gen/go/catalog/v1"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/config"
)

// Services holds ServiceClient instances for each downstream micro-service.
//
// Most handlers still proxy via HTTP through ServiceClient. A small number of
// read paths have been migrated to gRPC — their typed clients live here
// alongside the HTTP clients and are wired from main.go after the gRPC
// connections are established.
type Services struct {
	Auth      *ServiceClient
	Catalog   *ServiceClient
	Order     *ServiceClient
	Inventory *ServiceClient
	Search    *ServiceClient
	Recommend *ServiceClient
	Cart      *ServiceClient
	Inquiry   *ServiceClient
	Review    *ServiceClient

	// CatalogGRPC is used by the buyer read path (ListProducts, GetProduct).
	// Other catalog routes still go through the HTTP Catalog client above.
	CatalogGRPC catalogv1.CatalogServiceClient
}

// NewServices creates all service clients from the gateway configuration.
func NewServices(cfg config.Config) *Services {
	return &Services{
		Auth:      NewServiceClient(cfg.AuthServiceURL),
		Catalog:   NewServiceClient(cfg.CatalogServiceURL),
		Order:     NewServiceClient(cfg.OrderServiceURL),
		Inventory: NewServiceClient(cfg.InventoryServiceURL),
		Search:    NewServiceClient(cfg.SearchServiceURL),
		Recommend: NewServiceClient(cfg.RecommendServiceURL),
		Cart:      NewServiceClient(cfg.CartServiceURL),
		Inquiry:   NewServiceClient(cfg.InquiryServiceURL),
		Review:    NewServiceClient(cfg.ReviewServiceURL),
	}
}
