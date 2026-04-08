package grpcclient

import (
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	catalogv1 "github.com/Riku-KANO/ec-test/gen/go/catalog/v1"
	inventoryv1 "github.com/Riku-KANO/ec-test/gen/go/inventory/v1"
	orderv1 "github.com/Riku-KANO/ec-test/gen/go/order/v1"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/config"
)

// GRPCClients holds gRPC client connections and typed clients for downstream services.
type GRPCClients struct {
	catalogConn   *grpc.ClientConn
	inventoryConn *grpc.ClientConn
	orderConn     *grpc.ClientConn

	CatalogClient   catalogv1.CatalogServiceClient
	InventoryClient inventoryv1.InventoryServiceClient
	OrderClient     orderv1.OrderServiceClient
}

// NewGRPCClients creates gRPC connections to each downstream service.
func NewGRPCClients(cfg config.Config) (*GRPCClients, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	catalogConn, err := grpc.NewClient(cfg.CatalogGRPCAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to catalog gRPC: %w", err)
	}

	inventoryConn, err := grpc.NewClient(cfg.InventoryGRPCAddr, opts...)
	if err != nil {
		catalogConn.Close()
		return nil, fmt.Errorf("failed to connect to inventory gRPC: %w", err)
	}

	orderConn, err := grpc.NewClient(cfg.OrderGRPCAddr, opts...)
	if err != nil {
		catalogConn.Close()
		inventoryConn.Close()
		return nil, fmt.Errorf("failed to connect to order gRPC: %w", err)
	}

	slog.Info("gRPC clients connected",
		"catalog", cfg.CatalogGRPCAddr,
		"inventory", cfg.InventoryGRPCAddr,
		"order", cfg.OrderGRPCAddr,
	)

	return &GRPCClients{
		catalogConn:     catalogConn,
		inventoryConn:   inventoryConn,
		orderConn:       orderConn,
		CatalogClient:   catalogv1.NewCatalogServiceClient(catalogConn),
		InventoryClient: inventoryv1.NewInventoryServiceClient(inventoryConn),
		OrderClient:     orderv1.NewOrderServiceClient(orderConn),
	}, nil
}

// Close closes all gRPC connections.
func (c *GRPCClients) Close() {
	if c.catalogConn != nil {
		c.catalogConn.Close()
	}
	if c.inventoryConn != nil {
		c.inventoryConn.Close()
	}
	if c.orderConn != nil {
		c.orderConn.Close()
	}
}
