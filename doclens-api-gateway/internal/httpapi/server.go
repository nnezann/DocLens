package httpapi

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/doclens/api-gateway/internal/auth"
	"github.com/doclens/api-gateway/internal/observability"
	"github.com/doclens/api-gateway/internal/ratelimit"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

type Server struct {
	mux            *http.ServeMux
	logger         *slog.Logger
	metrics        *observability.Metrics
	verifier       auth.Verifier
	authDisabled   bool
	limiter        *ratelimit.Limiter
	requestTimeout time.Duration
	healthChecks   map[string]healthv1.HealthClient
}

type Deps struct {
	Identity       IdentityClient
	Documents      DocumentsClient
	Verification   VerificationClient
	HealthChecks   map[string]healthv1.HealthClient
	Logger         *slog.Logger
	Metrics        *observability.Metrics
	JWTSecret      string
	AuthDisabled   bool
	RateLimiter    *ratelimit.Limiter
	RequestTimeout time.Duration
}

func New(deps Deps) http.Handler {
	s := &Server{
		mux:            http.NewServeMux(),
		logger:         deps.Logger,
		metrics:        deps.Metrics,
		verifier:       auth.NewVerifier(deps.JWTSecret),
		authDisabled:   deps.AuthDisabled,
		limiter:        deps.RateLimiter,
		requestTimeout: deps.RequestTimeout,
		healthChecks:   deps.HealthChecks,
	}
	if s.logger == nil {
		s.logger = observability.NewLogger()
	}
	if s.metrics == nil {
		s.metrics = observability.NewMetrics()
	}

	h := handlers{identity: deps.Identity, documents: deps.Documents, verification: deps.Verification}
	s.routes(h)
	return s.middleware(s.mux)
}

func (s *Server) routes(h handlers) {
	s.mux.HandleFunc("GET /healthz", h.health)
	s.mux.HandleFunc("GET /readyz", s.ready)
	s.mux.Handle("GET /metrics", s.metrics)
	s.mux.HandleFunc("POST /identity/login", h.login)
	s.mux.HandleFunc("POST /identity/users", h.createUser)
	s.mux.HandleFunc("POST /documents", h.createDocument)
	s.mux.HandleFunc("GET /documents/{id}", h.getDocument)
	s.mux.HandleFunc("POST /verifications", h.startVerification)
	s.mux.HandleFunc("GET /verifications/{id}", h.getVerification)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		requestID := requestID(r)
		ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
		defer cancel()
		r = r.WithContext(ctx)
		w.Header().Set("X-Request-ID", requestID)

		if s.limiter != nil && !s.limiter.Allow(clientIP(r)) {
			observability.Error(recorder, http.StatusTooManyRequests, "rate limit exceeded")
			s.finish(r, requestID, recorder.status, start)
			return
		}
		if !s.authDisabled && requiresAuth(r) {
			claims, err := s.verifier.VerifyBearer(r.Header.Get("Authorization"))
			if err != nil {
				observability.Error(recorder, http.StatusUnauthorized, "unauthorized")
				s.finish(r, requestID, recorder.status, start)
				return
			}
			r = r.WithContext(auth.WithClaims(r.Context(), claims))
		}
		next.ServeHTTP(recorder, r)
		s.finish(r, requestID, recorder.status, start)
	})
}

func (s *Server) finish(r *http.Request, requestID string, status int, start time.Time) {
	duration := time.Since(start)
	if s.metrics != nil {
		s.metrics.Observe(r.Method+" "+r.URL.Path, status, duration)
	}
	s.logger.Info("request",
		"request_id", requestID,
		"method", r.Method,
		"path", r.URL.Path,
		"duration", duration.Milliseconds(),
		"status", status,
	)
}

func requiresAuth(r *http.Request) bool {
	if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" {
		return false
	}
	return !(r.Method == http.MethodPost && r.URL.Path == "/identity/login")
}

func requestID(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get("X-Request-ID")); id != "" {
		return id
	}
	return time.Now().UTC().Format("20060102150405.000000000")
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}
