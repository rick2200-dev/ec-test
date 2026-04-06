package proxy

import (
	"github.com/Riku-KANO/ec-test/services/gateway/internal/config"
)

// Services holds ServiceClient instances for each downstream micro-service.
type Services struct {
	Auth      *ServiceClient
	Catalog   *ServiceClient
	Order     *ServiceClient
	Inventory *ServiceClient
	Search    *ServiceClient
	Recommend *ServiceClient
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
	}
}
