package grpcserver

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	loyaltyv1 "github.com/Riku-KANO/ec-test/services/loyalty/api/gen/go/loyalty/v1"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/domain"
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

func transactionToProto(tx *domain.Transaction) *loyaltyv1.Transaction {
	if tx == nil {
		return nil
	}
	out := &loyaltyv1.Transaction{
		Id:           tx.ID.String(),
		BuyerAuth0Id: tx.BuyerAuth0ID,
		Type:         string(tx.Type),
		Amount:       tx.Amount,
		BalanceAfter: tx.BalanceAfter,
		SourceType:   string(tx.SourceType),
		SourceId:     tx.SourceID,
		Note:         tx.Note,
		ExpiresAt:    tsPtrOrNil(tx.ExpiresAt),
		CreatedAt:    tsOrNil(tx.CreatedAt),
	}
	if tx.ReservationID != nil {
		out.ReservationId = tx.ReservationID.String()
	}
	return out
}
