package server

import (
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"e-com-rental-cloth/backend/internal/platform/auth"
	"e-com-rental-cloth/backend/internal/platform/httpx"
)

const maxRequestBodyBytes int64 = 1 << 20 // 1 MiB default body limit

type Options struct {
	AllowedOrigins []string
}

type HealthStatus struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func NewRouter(opts Options) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(limitBodySize(maxRequestBodyBytes))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   opts.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Idempotency-Key", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Global baseline rate limit per authenticated user_id (or unspoofable peer IP for guests)
	r.Use(RateLimitByUserOrIP(120, time.Minute))

	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
			httpx.JSON(w, http.StatusOK, HealthStatus{
				Status:  "ok",
				Service: "e-com-rental-cloth-api",
			}, nil)
		})
	})

	return r
}

// RateLimitByUserOrIP keys rate limits to the authenticated user_id when present,
// preventing bots with valid sessions from bypassing limits via proxy IP rotation,
// and falls back to an unspoofable peer IP key without trusting spoofable X-Forwarded-For headers.
func RateLimitByUserOrIP(requestLimit int, windowLength time.Duration) func(http.Handler) http.Handler {
	return httprate.LimitBy(
		requestLimit,
		windowLength,
		func(r *http.Request) (string, error) {
			if uid, ok := auth.UserIDFromContext(r.Context()); ok {
				return "uid:" + uid.String(), nil
			}
			return clientIPKey(r), nil
		},
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			httpx.WriteError(w, r, httpx.NewError(
				http.StatusTooManyRequests,
				"RATE_LIMIT_EXCEEDED",
				"ทำรายการถี่เกินไป กรุณารอสักครู่แล้วลองใหม่",
				nil,
			))
		}),
	)
}

func clientIPKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		if addr.Is6() {
			if prefix, err := addr.Prefix(64); err == nil {
				return "ip:" + prefix.Masked().Addr().String()
			}
		}
		return "ip:" + addr.String()
	}
	return "ip:" + host
}

func limitBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
