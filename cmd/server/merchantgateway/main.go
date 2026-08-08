package main

import (
	"context"
	"log"

	"paynexus/internal/golibs/bootstrap"
	"paynexus/internal/golibs/configs"
	"paynexus/internal/golibs/database"
	"paynexus/internal/golibs/interceptors"
	"paynexus/internal/merchant/domain"
	"paynexus/internal/merchant/repository"
	"paynexus/internal/merchant/service"
	merchv1pb "paynexus/pkg/genproto/merchant/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// AppServer implements interface bootstrap.ServiceRegistrar
type AppServer struct {
	merchantSvc *service.MerchantServiceServerImpl
}

func (a *AppServer) RegisterGRPC(server *grpc.Server) {
	// register merchant service to grpc server
	merchv1pb.RegisterMerchantServiceServer(server, a.merchantSvc)
}

func main() {
	ctx := context.Background()
	// 1. Config server (Microservice name = merchantgateway, port = :50051)
	appCfg := configs.LoadConfigFromEnv("merchantgateway", ":50051")

	ignoreMethods := []string{
		"/grpc.health.v1.Health/Check",
	}

	tempLogger := zap.NewExample()

	authInterceptor := interceptors.UnaryAuthInterceptor(tempLogger, ignoreMethods)

	// 2. Initial Bootstrap Engine
	boot := bootstrap.NewServerBootstrap(appCfg.Server, grpc.UnaryInterceptor(authInterceptor))
	db, err := database.NewDBWrapper(ctx, appCfg.Postgres, boot.Logger)
	var dbExec database.QueryExecer
	if err != nil {
		boot.Logger.Warn("Không thể kết nối PostgreSQL local", zap.Error(err))
	} else {
		dbExec = db.Pool
		defer db.Close()
	}
	// Inject DB into Repo (return domain.MerchantRepository interface)
	var merchantRepo domain.MerchantRepository

	merchantRepo = repository.NewMerchantRepoImpl(dbExec)

	// 3. Initial Service Implementation
	merchantSvc := service.NewMerchantServiceServerImpl(merchantRepo, boot.Logger)

	app := &AppServer{
		merchantSvc: merchantSvc,
	}

	// 4. Run server
	if err := boot.Run(app); err != nil {
		log.Fatalf("Failed to run merchantgateway: %v", err)
	}
}
