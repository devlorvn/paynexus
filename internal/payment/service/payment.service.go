package service

import (
	"context"
	"encoding/json"
	"fmt"
	"paynexus/internal/golibs/interceptors"
	"paynexus/internal/payment/domain"
	payv1pb "paynexus/pkg/genproto/payment/v1"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type PaymentServiceServerImpl struct {
	payv1pb.UnimplementedPaymentServiceServer

	paymentRepo domain.PaymentRepository
	evenPub     domain.EventPublisher
	idempStore  domain.IdempotencyStorage
	logger      *zap.Logger
}

func NewPaymentServiceServerImpl(
	paymentRepo domain.PaymentRepository,
	evenPub domain.EventPublisher,
	idempStore domain.IdempotencyStorage,
	logger *zap.Logger,
) *PaymentServiceServerImpl {
	return &PaymentServiceServerImpl{
		paymentRepo: paymentRepo,
		evenPub:     evenPub,
		idempStore:  idempStore,
		logger:      logger,
	}
}

func (s *PaymentServiceServerImpl) CreateInvoice(ctx context.Context, req *payv1pb.CreateInvoiceRequest) (*payv1pb.CreateInvoiceResponse, error) {
	if req.GetAmount() <= 0 || req.GetCurrency() == "" || req.GetOrderReference() == "" {
		return nil, status.Error(codes.InvalidArgument, "Amount, Currency, OrderReference is required")
	}

	var idempotencyKey string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		keys := md.Get("x-idempotency-key")
		if len(keys) > 0 {
			idempotencyKey = keys[0]
		}
	}
	if idempotencyKey != "" && s.idempStore != nil {
		cachedBytes, found, err := s.idempStore.GetResponse(ctx, idempotencyKey)
		if err != nil {
			return nil, status.Error(codes.Internal, "Failed to check idempotency")
		}
		if found {
			var resp payv1pb.CreateInvoiceResponse
			if err := json.Unmarshal(cachedBytes, &resp); err != nil {
				return nil, status.Error(codes.Internal, "Failed to unmarshal cached response")
			}
			resp.IsCachedResponse = true
			return &resp, nil
		}

	}

	resourcePath := interceptors.GetResourcePathFromContext(ctx)
	merchantId := req.GetMerchantId()
	if merchantId <= 0 {
		merchantId = 1001
	}
	now := time.Now()
	invoiceCode := fmt.Sprintf("INV_%d", now.UnixNano())
	memoCode := fmt.Sprintf("NEXUS_%d", now.UnixNano()%1000)
	depositAddress := "3JpE2L76g2fD2F2g72F2g72F2g72F2g72F"
	inv := &domain.Invoice{
		MerchantID:     merchantId,
		InvoiceCode:    invoiceCode,
		OrderReference: req.GetOrderReference(),
		Currency:       req.GetCurrency(),
		Amount:         req.GetAmount(),
		DepositAddress: depositAddress,
		MemoCode:       memoCode,
		Status:         "PENDING",
		ResourcePath:   resourcePath,
		ExpiredAt:      now.Add(15 * time.Minute),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.paymentRepo.CreateInvoice(ctx, inv); err != nil {
		s.logger.Error("Failed to create invoice", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to create invoice")
	}

	resp := &payv1pb.CreateInvoiceResponse{
		Invoice: &payv1pb.Invoice{
			Id:             inv.ID,
			InvoiceCode:    inv.InvoiceCode,
			OrderReference: inv.OrderReference,
			Currency:       inv.Currency,
			Amount:         inv.Amount,
			DepositAddress: inv.DepositAddress,
			MemoCode:       inv.MemoCode,
			Status:         inv.Status,
			ResourcePath:   inv.ResourcePath,
			ExpiredAt:      inv.ExpiredAt.Format(time.RFC3339),
			CreatedAt:      inv.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      inv.UpdatedAt.Format(time.RFC3339),
		},
	}

	if s.evenPub != nil {
		eventData, _ := json.Marshal(map[string]interface{}{
			"invoice_code": inv.InvoiceCode,
			"merchant_id":  inv.MerchantID,
			"amount":       inv.Amount,
			"currency":     inv.Currency,
			"created_at":   inv.CreatedAt,
		})
		s.evenPub.PublishEvent(ctx, "payment.invoice.created", eventData)
	}

	if idempotencyKey != "" && s.idempStore != nil {
		respBytes, err := json.Marshal(resp)
		if err != nil {
			return nil, status.Error(codes.Internal, "Failed to marshal response")
		}
		if err := s.idempStore.SaveResponse(ctx, idempotencyKey, respBytes, 24*time.Hour); err != nil {
			s.logger.Error("Failed to save response", zap.Error(err))
			return nil, status.Error(codes.Internal, "Failed to save response")
		}
	}

	return resp, nil
}

func (s *PaymentServiceServerImpl) GetInvoice(ctx context.Context, req *payv1pb.GetInvoiceRequest) (*payv1pb.GetInvoiceResponse, error) {
	if req.GetInvoiceId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Invoice ID không hợp lệ")
	}
	return &payv1pb.GetInvoiceResponse{
		Invoice: &payv1pb.Invoice{
			Id:          req.GetInvoiceId(),
			InvoiceCode: "INV_DEMO_123",
			Status:      "PENDING",
		},
	}, nil
}
