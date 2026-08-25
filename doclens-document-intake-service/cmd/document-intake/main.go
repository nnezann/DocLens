package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/doclens/document-intake-service/internal/config"
	documentsv1 "github.com/doclens/document-intake-service/internal/gen/doclens/documents/v1"
	"github.com/doclens/document-intake-service/internal/observability"
	"github.com/doclens/document-intake-service/internal/service"
	"github.com/doclens/document-intake-service/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	logger := observability.NewLogger()
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	objectStorage, err := store.NewLocalObjectStorage(cfg.StorageDir)
	if err != nil {
		logger.Error("initialize object storage", slog.String("error", err.Error()))
		os.Exit(1)
	}

	documentService := service.NewService(
		store.NewMemoryStore(),
		objectStorage,
		cfg.MaxUploadBytes,
		cfg.AllowedContentTypes,
	)

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		logger.Error("listen", slog.String("address", cfg.Address), slog.String("error", err.Error()))
		os.Exit(1)
	}

	server := grpc.NewServer()
	documentsv1.RegisterDocumentIntakeServiceServer(server, documentService)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthv1.HealthCheckResponse_SERVING)
	healthv1.RegisterHealthServer(server, healthServer)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	logger.Info("doclens document intake service listening", slog.String("address", cfg.Address))
	if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		logger.Error("grpc server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("doclens document intake service stopped")
}
