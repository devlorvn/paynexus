package main

import (
	"context"
	"log"

	"paynexus/internal/golibs/bootstrap"
	"paynexus/internal/golibs/configs"
	"paynexus/internal/golibs/database"
	"paynexus/internal/golibs/nats"
	"paynexus/internal/golibs/redis"
	"paynexus/internal/payment/domain"
	"paynexus/internal/payment/repository"
	"paynexus/internal/payment/service"
	payv1pb "paynexus/pkg/genproto/payment/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type AppServer struct {
	paymentSvc *service.PaymentServiceServerImpl
}

func (a *AppServer) RegisterGRPC(server *grpc.Server) {
	payv1pb.RegisterPaymentServiceServer(server, a.paymentSvc)
}

func main() {
	ctx := context.Background()

	appCfg := configs.LoadConfigFromEnv("paymentengine", ":50052")
	boot := bootstrap.NewServerBootstrap(appCfg.Server)

	// 1. DB Adapter
	db, err := database.NewDBWrapper(ctx, appCfg.Postgres, boot.Logger)
	if err != nil {
		boot.Logger.Warn("Không thể kết nối Postgres local", zap.Error(err))
	} else {
		defer db.Close()
	}

	// 2. Redis Adapter (IdempotencyStorage Port)
	rdbClient, err := redis.NewRedisClient(ctx, appCfg.Redis, boot.Logger)
	var idempStore domain.IdempotencyStorage
	if err == nil {
		idempStore = redis.NewIdempotencyEngine(rdbClient.Client)
		defer rdbClient.Close()
	}

	// 3. NATS Adapter (EventPublisher Port)
	natsClient, err := nats.NewNATSClient(appCfg.NATS, boot.Logger)
	var eventPub domain.EventPublisher
	if err == nil {
		eventPub = natsClient
		defer natsClient.Close()
	}

	// 4. Inject Adapters vào Service thông qua Port Interfaces
	paymentRepo := repository.NewPaymentRepoImpl(db.Pool)
	paymentSvc := service.NewPaymentServiceServerImpl(paymentRepo, eventPub, idempStore, boot.Logger)

	app := &AppServer{
		paymentSvc: paymentSvc,
	}

	if err := boot.Run(app); err != nil {
		log.Fatalf("Khởi chạy paymentengine thất bại: %v", err)
	}
}
