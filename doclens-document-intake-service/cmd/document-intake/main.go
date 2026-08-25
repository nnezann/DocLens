package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

	localStorage, err := store.NewLocalObjectStorage(cfg.StorageDir)
	if err != nil {
		logger.Error("initialize object storage", slog.String("error", err.Error()))
		os.Exit(1)
	}
	var objectStorage store.ObjectStorage = localStorage
	if cfg.R2Endpoint != "" || cfg.R2Bucket != "" {
		if cfg.R2Endpoint == "" || cfg.R2AccessKeyID == "" || cfg.R2SecretAccessKey == "" || cfg.R2Bucket == "" {
			logger.Error("r2 configuration is incomplete", slog.String("error", "R2_ENDPOINT, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, and R2_BUCKET are required together"))
			os.Exit(1)
		}
		awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
			awsconfig.WithRegion("auto"),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.R2AccessKeyID, cfg.R2SecretAccessKey, "")),
		)
		if err != nil {
			logger.Error("initialize r2 config", slog.String("error", err.Error()))
			os.Exit(1)
		}
		r2Client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
			endpoint := strings.TrimRight(cfg.R2Endpoint, "/")
			options.BaseEndpoint = &endpoint
			options.UsePathStyle = true
		})
		objectStorage, err = store.NewR2ObjectStorage(r2Client, cfg.R2Bucket)
		if err != nil {
			logger.Error("initialize r2 storage", slog.String("error", err.Error()))
			os.Exit(1)
		}
		logger.Info("using cloudflare r2 object storage", slog.String("bucket", cfg.R2Bucket))
	}

	var metadataStore store.Store = store.NewMemoryStore()
	var postgresStore *store.PostgresStore
	if cfg.DatabaseURL != "" {
		postgresStore, err = store.NewPostgresStore(context.Background(), cfg.DatabaseURL)
		if err != nil {
			logger.Error("initialize postgres store", slog.String("error", err.Error()))
			os.Exit(1)
		}
		defer postgresStore.Close()
		migration, err := os.ReadFile("migrations/001_initial.sql")
		if err != nil {
			logger.Error("read database migration", slog.String("error", err.Error()))
			os.Exit(1)
		}
		migrationCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err = postgresStore.Migrate(migrationCtx, string(migration))
		cancel()
		if err != nil {
			logger.Error("run database migration", slog.String("error", err.Error()))
			os.Exit(1)
		}
		metadataStore = postgresStore
		logger.Info("using postgres metadata store")
	}
	documentService := service.NewService(
		metadataStore,
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
