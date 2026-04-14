package grpcserver

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/Riku-KANO/ec-test/gen/go/common/v1"
	subscriptionv1 "github.com/Riku-KANO/ec-test/gen/go/subscription/v1"
	"github.com/Riku-KANO/ec-test/services/subscription/internal/domain"
)

func tsOrNil(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func tsPtrOrNil(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

func planFeaturesToProto(f domain.PlanFeatures) *subscriptionv1.PlanFeatures {
	return &subscriptionv1.PlanFeatures{
		MaxProducts:     int32(f.MaxProducts),
		SearchBoost:     f.SearchBoost,
		FeaturedSlots:   int32(f.FeaturedSlots),
		PromotedResults: int32(f.PromotedResults),
	}
}

func planFeaturesFromProto(f *subscriptionv1.PlanFeatures) domain.PlanFeatures {
	if f == nil {
		return domain.PlanFeatures{}
	}
	return domain.PlanFeatures{
		MaxProducts:     int(f.GetMaxProducts()),
		SearchBoost:     f.GetSearchBoost(),
		FeaturedSlots:   int(f.GetFeaturedSlots()),
		PromotedResults: int(f.GetPromotedResults()),
	}
}

func buyerPlanFeaturesToProto(f domain.BuyerPlanFeatures) *subscriptionv1.BuyerPlanFeatures {
	return &subscriptionv1.BuyerPlanFeatures{
		FreeShipping: f.FreeShipping,
	}
}

func buyerPlanFeaturesFromProto(f *subscriptionv1.BuyerPlanFeatures) domain.BuyerPlanFeatures {
	if f == nil {
		return domain.BuyerPlanFeatures{}
	}
	return domain.BuyerPlanFeatures{FreeShipping: f.GetFreeShipping()}
}

func sellerPlanToProto(p *domain.SubscriptionPlan) *subscriptionv1.SellerPlan {
	return &subscriptionv1.SellerPlan{
		Id:            p.ID.String(),
		Name:          p.Name,
		Slug:          p.Slug,
		Tier:          int32(p.Tier),
		Price:         &commonv1.Money{Amount: p.PriceAmount, Currency: p.PriceCurrency},
		Features:      planFeaturesToProto(p.Features),
		StripePriceId: p.StripePriceID,
		Status:        p.Status,
		CreatedAt:     tsOrNil(p.CreatedAt),
		UpdatedAt:     tsOrNil(p.UpdatedAt),
	}
}

func sellerSubscriptionToProto(s *domain.SellerSubscriptionWithPlan) *subscriptionv1.SellerSubscription {
	return &subscriptionv1.SellerSubscription{
		Id:                   s.ID.String(),
		SellerId:             s.SellerID.String(),
		PlanId:               s.PlanID.String(),
		StripeSubscriptionId: s.StripeSubscriptionID,
		StripeCustomerId:     s.StripeCustomerID,
		Status:               string(s.Status),
		CurrentPeriodStart:   tsPtrOrNil(s.CurrentPeriodStart),
		CurrentPeriodEnd:     tsPtrOrNil(s.CurrentPeriodEnd),
		CanceledAt:           tsPtrOrNil(s.CanceledAt),
		CreatedAt:            tsOrNil(s.CreatedAt),
		UpdatedAt:            tsOrNil(s.UpdatedAt),
		PlanName:             s.PlanName,
		PlanSlug:             s.PlanSlug,
		PlanTier:             int32(s.PlanTier),
	}
}

func sellerSubscriptionBareToProto(s *domain.SellerSubscription) *subscriptionv1.SellerSubscription {
	return &subscriptionv1.SellerSubscription{
		Id:                   s.ID.String(),
		SellerId:             s.SellerID.String(),
		PlanId:               s.PlanID.String(),
		StripeSubscriptionId: s.StripeSubscriptionID,
		StripeCustomerId:     s.StripeCustomerID,
		Status:               string(s.Status),
		CurrentPeriodStart:   tsPtrOrNil(s.CurrentPeriodStart),
		CurrentPeriodEnd:     tsPtrOrNil(s.CurrentPeriodEnd),
		CanceledAt:           tsPtrOrNil(s.CanceledAt),
		CreatedAt:            tsOrNil(s.CreatedAt),
		UpdatedAt:            tsOrNil(s.UpdatedAt),
	}
}

func buyerPlanToProto(p *domain.BuyerPlan) *subscriptionv1.BuyerPlan {
	return &subscriptionv1.BuyerPlan{
		Id:            p.ID.String(),
		Name:          p.Name,
		Slug:          p.Slug,
		Price:         &commonv1.Money{Amount: p.PriceAmount, Currency: p.PriceCurrency},
		Features:      buyerPlanFeaturesToProto(p.Features),
		StripePriceId: p.StripePriceID,
		Status:        p.Status,
		CreatedAt:     tsOrNil(p.CreatedAt),
		UpdatedAt:     tsOrNil(p.UpdatedAt),
	}
}

func buyerSubscriptionToProto(s *domain.BuyerSubscriptionWithPlan) *subscriptionv1.BuyerSubscription {
	return &subscriptionv1.BuyerSubscription{
		Id:                   s.ID.String(),
		BuyerAuth0Id:         s.BuyerAuth0ID,
		PlanId:               s.PlanID.String(),
		StripeSubscriptionId: s.StripeSubscriptionID,
		StripeCustomerId:     s.StripeCustomerID,
		Status:               string(s.Status),
		CurrentPeriodStart:   tsPtrOrNil(s.CurrentPeriodStart),
		CurrentPeriodEnd:     tsPtrOrNil(s.CurrentPeriodEnd),
		CanceledAt:           tsPtrOrNil(s.CanceledAt),
		CreatedAt:            tsOrNil(s.CreatedAt),
		UpdatedAt:            tsOrNil(s.UpdatedAt),
		PlanName:             s.PlanName,
		PlanSlug:             s.PlanSlug,
		Features:             buyerPlanFeaturesToProto(s.Features),
	}
}

func buyerSubscriptionBareToProto(s *domain.BuyerSubscription) *subscriptionv1.BuyerSubscription {
	return &subscriptionv1.BuyerSubscription{
		Id:                   s.ID.String(),
		BuyerAuth0Id:         s.BuyerAuth0ID,
		PlanId:               s.PlanID.String(),
		StripeSubscriptionId: s.StripeSubscriptionID,
		StripeCustomerId:     s.StripeCustomerID,
		Status:               string(s.Status),
		CurrentPeriodStart:   tsPtrOrNil(s.CurrentPeriodStart),
		CurrentPeriodEnd:     tsPtrOrNil(s.CurrentPeriodEnd),
		CanceledAt:           tsPtrOrNil(s.CanceledAt),
		CreatedAt:            tsOrNil(s.CreatedAt),
		UpdatedAt:            tsOrNil(s.UpdatedAt),
	}
}
