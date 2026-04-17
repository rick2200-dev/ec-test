package grpcclient

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Riku-KANO/ec-test/pkg/tracing"
	catalogv1 "github.com/Riku-KANO/ec-test/services/catalog/api/gen/go/catalog/v1"
)

// CatalogClient is an outbound gRPC client for the catalog service. Used by
// the order service at checkout time to resolve SKU ids to their parent
// product ids without reading catalog_svc directly. Replaces the earlier
// cross-schema JOIN that order performed in its own transaction.
//
// Implements port.SKULookup.
type CatalogClient struct {
	conn   *grpc.ClientConn
	client catalogv1.CatalogServiceClient
}

// NewCatalogClient dials the catalog gRPC endpoint. Insecure credentials are
// used because this is an in-cluster call; the same internalToken pattern
// applied to subscription is reused so the catalog side can authenticate the
// caller.
func NewCatalogClient(addr, internalToken string) (*CatalogClient, error) {
	conn, err := grpc.NewClient(addr, tracing.DialOptions(
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(internalTokenCreds{token: internalToken}),
	)...)
	if err != nil {
		return nil, fmt.Errorf("dial catalog service: %w", err)
	}
	return &CatalogClient{
		conn:   conn,
		client: catalogv1.NewCatalogServiceClient(conn),
	}, nil
}

// Close releases the underlying gRPC connection.
func (c *CatalogClient) Close() error { return c.conn.Close() }

// BatchGetSKUProductIDs returns a map sku_id -> product_id for each sku the
// catalog service knows about. Unknown ids are silently omitted — callers
// must reject the checkout if a line refers to a missing SKU.
func (c *CatalogClient) BatchGetSKUProductIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]uuid.UUID{}, nil
	}
	rawIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		rawIDs = append(rawIDs, id.String())
	}
	resp, err := c.client.BatchGetSKUs(ctx, &catalogv1.BatchGetSKUsRequest{SkuIds: rawIDs})
	if err != nil {
		return nil, fmt.Errorf("catalog BatchGetSKUs: %w", err)
	}
	out := make(map[uuid.UUID]uuid.UUID, len(resp.GetMappings()))
	for _, m := range resp.GetMappings() {
		skuID, err := uuid.Parse(m.GetSkuId())
		if err != nil {
			return nil, fmt.Errorf("invalid sku_id %q: %w", m.GetSkuId(), err)
		}
		productID, err := uuid.Parse(m.GetProductId())
		if err != nil {
			return nil, fmt.Errorf("invalid product_id %q: %w", m.GetProductId(), err)
		}
		out[skuID] = productID
	}
	return out, nil
}
