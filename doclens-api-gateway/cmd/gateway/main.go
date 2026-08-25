package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/doclens/api-gateway/internal/config"
	"github.com/doclens/api-gateway/internal/grpcclient"
	"github.com/doclens/api-gateway/internal/httpapi"
	"github.com/doclens/api-gateway/internal/observability"
	"github.com/doclens/api-gateway/internal/ratelimit"
)

func main() {
	logger := observability.NewLogger()
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	clients, err := grpcclient.Dial(ctx, grpcclient.Targets{
		Identity:     cfg.IdentityTarget,
		Documents:    cfg.DocumentsTarget,
		Verification: cfg.VerificationTarget,
		Insecure:     cfg.GRPCInsecure,
	})
	if err != nil {
		logger.Error("connect grpc services", "error", err)
		os.Exit(1)
	}
	defer clients.Close()

	handler := httpapi.New(httpapi.Deps{
		Identity:       clients.Identity,
		Documents:      clients.Documents,
		Verification:   clients.Verification,
		HealthChecks:   clients.HealthChecks,
		Logger:         logger,
		Metrics:        observability.NewMetrics(),
		JWTSecret:      cfg.JWTSecret,
		AuthDisabled:   cfg.AuthDisabled,
		RateLimiter:    ratelimit.New(cfg.RateLimitRPS, cfg.RateLimitBurst),
		RequestTimeout: cfg.RequestTimeout,
	})

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("doclens api gateway listening", slog.String("address", cfg.Address))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("doclens api gateway stopped")
}
