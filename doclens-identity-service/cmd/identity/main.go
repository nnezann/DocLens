package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/doclens/identity-service/internal/auth"
	"github.com/doclens/identity-service/internal/config"
	identityv1 "github.com/doclens/identity-service/internal/gen/doclens/identity/v1"
	"github.com/doclens/identity-service/internal/observability"
	"github.com/doclens/identity-service/internal/service"
	"github.com/doclens/identity-service/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	logger := observability.NewLogger()
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		logger.Error("listen", "address", cfg.Address, "error", err)
		os.Exit(1)
	}

	var identityStore store.Store
	var closeStore func()
	if cfg.DatabaseURL == "" {
		logger.Warn("DATABASE_URL is not set; using development-only in-memory identity store")
		identityStore = store.NewMemory()
	} else {
		postgres, err := store.NewPostgres(context.Background(), cfg.DatabaseURL, cfg.DatabaseTimeout)
		if err != nil {
			logger.Error("database connection error", "error", err)
			os.Exit(1)
		}
		if err := postgres.Migrate(context.Background()); err != nil {
			postgres.Close()
			logger.Error("database migration error", "error", err)
			os.Exit(1)
		}
		identityStore = postgres
		closeStore = postgres.Close
	}
	if closeStore != nil {
		defer closeStore()
	}

	identity := service.NewIdentity(
		identityStore,
		auth.NewTokenIssuer(cfg.JWTSecret),
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
	)
	if cfg.DevelopmentSeed {
		if err := service.SeedDevelopmentUser(context.Background(), identity, cfg.DevelopmentOrgID, cfg.DevelopmentEmail, cfg.DevelopmentPassword); err != nil {
			logger.Error("seed development user", "error", err)
			os.Exit(1)
		}
		logger.Info("seeded development identity", slog.String("email", cfg.DevelopmentEmail), slog.String("organization_id", cfg.DevelopmentOrgID))
	}

	server := grpc.NewServer()
	identityv1.RegisterIdentityServiceServer(server, identity)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthv1.HealthCheckResponse_SERVING)
	healthv1.RegisterHealthServer(server, healthServer)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	logger.Info("doclens identity service listening", slog.String("address", cfg.Address))
	if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		logger.Error("grpc server failed", "error", err)
		os.Exit(1)
	}
	logger.Info("doclens identity service stopped")
}
