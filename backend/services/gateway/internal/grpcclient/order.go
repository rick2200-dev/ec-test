package grpcclient

import (
	"context"
	"fmt"

	orderv1 "github.com/Riku-KANO/ec-test/gen/go/order/v1"
)

// CreateOrder calls the order gRPC service to create a new order.
func (c *GRPCClients) CreateOrder(
	ctx context.Context,
	tenantID, sellerID, buyerID string,
	lines []*orderv1.OrderLineInput,
	shippingAddress string,
) (*orderv1.Order, string, error) {
	req := &orderv1.CreateOrderRequest{
		TenantId:            tenantID,
		SellerId:            sellerID,
		BuyerAuth0Id:        buyerID,
		Lines:               lines,
		ShippingAddressJson: shippingAddress,
	}
	resp, err := c.OrderClient.CreateOrder(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("create order: %w", err)
	}
	return resp.GetOrder(), resp.GetStripeClientSecret(), nil
}
