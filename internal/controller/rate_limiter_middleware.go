package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/exceptions"
	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/usecase"
)

type Config struct {
	IPLimit    int           // Ex: 10 req/s
	TokenLimit int           // Ex: 100 req/s
	BlockTime  time.Duration // Ex: 5 * time.Minute
}

func NewConfig(ipLimit int, tokenLimit int, BlockTime time.Duration) *Config {
	return &Config{
		IPLimit: ipLimit,
		TokenLimit: tokenLimit,
		BlockTime: BlockTime,
	}
}

type MiddleWareRateLimitController[T any] struct{
	client T
	Config *Config
	Allow func(ctx context.Context, identifier string, isToken bool, client T) (bool, error)
}

func NewRateLimitMiddleware[T any](client T, config *Config, allow func(ctx context.Context, identifier string, isToken bool, client T) (bool, error)) *MiddleWareRateLimitController[T] {
	return &MiddleWareRateLimitController[T]{
		client: client,
		Config: config,
		Allow: allow,
	}
}

func extractIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (m *MiddleWareRateLimitController[Client]) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		token := r.Header.Get("API_KEY")
		var keyIdentifier string
		// var limit int

		if token != "" {
			// Hash do token para evitar expor chaves brutas no Redis
			hasher := sha256.New()
			hasher.Write([]byte(token))
			keyIdentifier = "token:" + hex.EncodeToString(hasher.Sum(nil))
		} else {
			ip := extractIP(r)
			keyIdentifier = "ip:" + ip
		}
		
		limitterUseCase := usecase.NewLimitterUseCase(m.Allow)
		res, _ := limitterUseCase.Execute(ctx, keyIdentifier, token != "", m.client)

		if res {
			next.ServeHTTP(w, r)
			return
		} else {
			err := exceptions.NewRateLimitException("you have reached the maximum number of requests or actions allowed within a certain time frame")
			w.WriteHeader(err.Code)
			json.NewEncoder(w).Encode(err)
		}
	})
}
