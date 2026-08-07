package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"paynexus/internal/merchant/domain"
	merchv1pb "paynexus/pkg/genproto/merchant/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MerchantServiceServerImpl struct {
	merchv1pb.UnimplementedMerchantServiceServer
	merchantRepo domain.MerchantRepository // 🟢 Service chỉ cần phụ thuộc vào Interface này
	logger       *zap.Logger
}

func NewMerchantServiceServerImpl(repo domain.MerchantRepository, logger *zap.Logger) *MerchantServiceServerImpl {
	return &MerchantServiceServerImpl{
		merchantRepo: repo,
		logger:       logger,
	}
}

func (s *MerchantServiceServerImpl) CreateMerchant(ctx context.Context, req *merchv1pb.CreateMerchantRequest) (*merchv1pb.CreateMerchantResponse, error) {
	if req.GetName() == "" || req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "Name và Email không được để trống")
	}

	now := time.Now()
	merchantDomain := &domain.Merchant{
		Code:         fmt.Sprintf("MERCH_%d", now.Unix()),
		Name:         req.GetName(),
		Email:        req.GetEmail(),
		Status:       "ACTIVE",
		ResourcePath: "/org/default/",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.merchantRepo.Create(ctx, merchantDomain); err != nil {
		s.logger.Error("Lỗi lưu Merchant vào DB", zap.Error(err))
		return nil, status.Error(codes.Internal, "Lưu dữ liệu thất bại")
	}

	return &merchv1pb.CreateMerchantResponse{
		Merchant: &merchv1pb.Merchant{
			Id:           merchantDomain.ID,
			Code:         merchantDomain.Code,
			Name:         merchantDomain.Name,
			Email:        merchantDomain.Email,
			Status:       merchantDomain.Status,
			ResourcePath: merchantDomain.ResourcePath,
			CreatedAt:    merchantDomain.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *MerchantServiceServerImpl) GetMerchant(ctx context.Context, req *merchv1pb.GetMerchantRequest) (*merchv1pb.GetMerchantResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Merchant ID không hợp lệ")
	}

	m, err := s.merchantRepo.GetByID(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "Không tìm thấy Merchant")
		}
		s.logger.Error("Lỗi truy vấn DB", zap.Error(err))
		return nil, status.Error(codes.Internal, "Lỗi truy vấn dữ liệu")
	}

	return &merchv1pb.GetMerchantResponse{
		Merchant: &merchv1pb.Merchant{
			Id:           m.ID,
			Code:         m.Code,
			Name:         m.Name,
			Email:        m.Email,
			Status:       m.Status,
			ResourcePath: m.ResourcePath,
			CreatedAt:    m.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *MerchantServiceServerImpl) GenerateAPIKey(ctx context.Context, req *merchv1pb.GenerateAPIKeyRequest) (*merchv1pb.GenerateAPIKeyResponse, error) {
	s.logger.Info("Đang sinh cặp khóa API Key/Secret mới", zap.Int64("merchant_id", req.GetMerchantId()))

	return &merchv1pb.GenerateAPIKeyResponse{
		PublicKey:        fmt.Sprintf("pk_live_%d", time.Now().UnixNano()),
		SecretKeyPrivate: fmt.Sprintf("sk_live_%d_secret_token", time.Now().UnixNano()),
		ExpiresAt:        time.Now().AddDate(1, 0, 0).Format(time.RFC3339),
	}, nil
}
