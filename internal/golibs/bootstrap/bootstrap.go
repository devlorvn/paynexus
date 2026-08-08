package bootstrap

import (
	"context"
	"net"
	"os"
	"os/signal"
	"paynexus/internal/golibs/logger"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// ServiceRegistrar là interface bắt buộc các Microservices phải implement
type ServiceRegistrar interface {
	RegisterGRPC(server *grpc.Server)
}

type ServerBootstrap struct {
	Config     ServerConfig
	Logger     *zap.Logger
	GRPCServer *grpc.Server
}

func NewServerBootstrap(cfg ServerConfig, opts ...grpc.ServerOption) *ServerBootstrap {
	zapLog := logger.NewLogger(cfg.ServiceName, cfg.IsDebug)
	grpcServer := grpc.NewServer(opts...)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus(cfg.ServiceName, healthpb.HealthCheckResponse_SERVING)

	return &ServerBootstrap{
		Config:     cfg,
		Logger:     zapLog,
		GRPCServer: grpcServer,
	}
}

func (sb *ServerBootstrap) Run(registrar ServiceRegistrar) error {
	// 1. Register GRPC services
	registrar.RegisterGRPC(sb.GRPCServer)

	// 2. Start GRPC server
	listener, err := net.Listen("tcp", sb.Config.Port)

	if err != nil {
		sb.Logger.Error("Fail to listen on port", zap.String("port", sb.Config.Port), zap.Error(err))
		return err
	}

	sb.Logger.Info("Starting gRPC Server", zap.String("port", sb.Config.Port))

	go func() {
		sb.Logger.Info("gRpc Server listing on port:", zap.String("port", sb.Config.Port))
		if err = sb.GRPCServer.Serve(listener); err != nil {
			sb.Logger.Error("Fail to serve gRPC", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	sb.Logger.Info("Shutting down server", zap.String("signal", sig.String()))

	timeoutCtx, cancel := context.WithTimeout(context.Background(), sb.Config.ShutdownTimeout)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		sb.GRPCServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		sb.Logger.Info("Graceful shutdown GrpcServer done")
	case <-timeoutCtx.Done():
		sb.Logger.Warn("Graceful shutdown timeout")
		sb.GRPCServer.Stop()
	}

	return nil
}
