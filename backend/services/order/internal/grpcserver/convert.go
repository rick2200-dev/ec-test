package grpcserver

import (
	"encoding/json"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/Riku-KANO/ec-test/gen/go/common/v1"
	orderv1 "github.com/Riku-KANO/ec-test/gen/go/order/v1"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// orderToProto converts a domain Order to its proto representation.
func orderToProto(o *domain.Order, lines []domain.OrderLine) *orderv1.Order {
	pb := &orderv1.Order{
		Id:           o.ID.String(),
		TenantId:     o.TenantID.String(),
		SellerId:     o.SellerID.String(),
		BuyerAuth0Id: o.BuyerAuth0ID,
		Status:       o.Status,
		Subtotal: &commonv1.Money{
			Amount:   o.SubtotalAmount,
			Currency: o.Currency,
		},
		Commission: &commonv1.Money{
			Amount:   o.CommissionAmount,
			Currency: o.Currency,
		},
		Total: &commonv1.Money{
			Amount:   o.TotalAmount,
			Currency: o.Currency,
		},
		ShippingAddressJson: string(o.ShippingAddress),
		CreatedAt:           timestamppb.New(o.CreatedAt),
		UpdatedAt:           timestamppb.New(o.UpdatedAt),
	}

	if o.StripePaymentIntentID != nil {
		pb.StripePaymentIntentId = *o.StripePaymentIntentID
	}

	if o.PaidAt != nil {
		pb.PaidAt = timestamppb.New(*o.PaidAt)
	}

	for _, l := range lines {
		pb.Lines = append(pb.Lines, orderLineToProto(&l, o.Currency))
	}

	return pb
}

// orderLineToProto converts a domain OrderLine to its proto representation.
func orderLineToProto(l *domain.OrderLine, currency string) *orderv1.OrderLine {
	return &orderv1.OrderLine{
		Id:          l.ID.String(),
		OrderId:     l.OrderID.String(),
		SkuId:       l.SKUID.String(),
		ProductName: l.ProductName,
		SkuCode:     l.SKUCode,
		Quantity:    int32(l.Quantity),
		UnitPrice: &commonv1.Money{
			Amount:   l.UnitPrice,
			Currency: currency,
		},
		LineTotal: &commonv1.Money{
			Amount:   l.LineTotal,
			Currency: currency,
		},
	}
}

// orderSummaryToProto converts a domain Order (without lines) to its proto representation.
func orderSummaryToProto(o *domain.Order) *orderv1.Order {
	return orderToProto(o, nil)
}

// payoutToProto converts a domain Payout to its proto representation.
func payoutToProto(p *domain.Payout) *orderv1.Payout {
	pb := &orderv1.Payout{
		Id:       p.ID.String(),
		TenantId: p.TenantID.String(),
		SellerId: p.SellerID.String(),
		OrderId:  p.OrderID.String(),
		Amount: &commonv1.Money{
			Amount:   p.Amount,
			Currency: p.Currency,
		},
		Status:    p.Status,
		CreatedAt: timestamppb.New(p.CreatedAt),
	}

	if p.StripeTransferID != nil {
		pb.StripeTransferId = *p.StripeTransferID
	}

	if p.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*p.CompletedAt)
	}

	return pb
}

// protoLinesToDomain converts proto OrderLineInput to domain OrderLineInput.
func protoLinesToDomain(lines []*orderv1.OrderLineInput) []domain.OrderLineInput {
	result := make([]domain.OrderLineInput, 0, len(lines))
	for _, l := range lines {
		skuID, _ := uuid.Parse(l.GetSkuId())
		result = append(result, domain.OrderLineInput{
			SKUID:    skuID,
			Quantity: int(l.GetQuantity()),
		})
	}
	return result
}

// parseShippingAddress converts a JSON string to json.RawMessage.
func parseShippingAddress(s string) json.RawMessage {
	if s == "" {
		return json.RawMessage("{}")
	}
	return json.RawMessage(s)
}
