package controller

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/usecase"
)

const (
	apiKeyHeader           = "API_KEY"
	tooManyRequestsMessage = "you have reached the maximum number of requests or actions allowed within a certain time frame"
)

type Config struct {
	IPLimit           int    // req/s por IP. Ex: 10
	TokenLimit        int    // req/s por token válido (sobrepõe o limite de IP). Ex: 100
	BlockTime         int    // segundos de bloqueio ao exceder o limite. Ex: 300
	APIToken          string // único token aceito no header API_KEY
	TrustProxyHeaders bool   // confia em X-Forwarded-For / X-Real-IP (só atrás de um proxy confiável)
}

func NewConfig(ipLimit, tokenLimit, blockTime int, apiToken string, trustProxyHeaders bool) *Config {
	return &Config{
		IPLimit:           ipLimit,
		TokenLimit:        tokenLimit,
		BlockTime:         blockTime,
		APIToken:          apiToken,
		TrustProxyHeaders: trustProxyHeaders,
	}
}

type MiddleWareRateLimitController[T any] struct {
	client  T
	Config  *Config
	limiter *usecase.LimitterUseCase[T]
}

func NewRateLimitMiddleware[T any](client T, config *Config, allow usecase.AllowFunc[T]) *MiddleWareRateLimitController[T] {
	return &MiddleWareRateLimitController[T]{
		client:  client,
		Config:  config,
		limiter: usecase.NewLimitterUseCase(allow),
	}
}

func (m *MiddleWareRateLimitController[T]) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identifier, limit := m.identify(r)

		allowed, err := m.limiter.Execute(r.Context(), identifier, limit, m.Config.BlockTime, m.client)
		if err != nil {
			// Falha no backend não deve ser confundida com excesso de requisições.
			log.Printf("rate limiter error: %v", err)
			writeText(w, http.StatusInternalServerError, "internal server error")
			return
		}

		if !allowed {
			writeText(w, http.StatusTooManyRequests, tooManyRequestsMessage)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// identify escolhe a identidade e o limite aplicável. Um token válido substitui
// completamente o limite de IP; token ausente ou inválido cai no limite por IP.
func (m *MiddleWareRateLimitController[T]) identify(r *http.Request) (string, int) {
	if token := r.Header.Get(apiKeyHeader); m.isValidToken(token) {
		// Hash do token para não gravar a chave bruta no Redis.
		sum := sha256.Sum256([]byte(token))
		return "token:" + hex.EncodeToString(sum[:]), m.Config.TokenLimit
	}

	return "ip:" + clientIP(r, m.Config.TrustProxyHeaders), m.Config.IPLimit
}

func (m *MiddleWareRateLimitController[T]) isValidToken(token string) bool {
	expected := m.Config.APIToken
	return token != "" && expected != "" &&
		subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

func clientIP(r *http.Request, trustProxyHeaders bool) string {
	if trustProxyHeaders {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			return strings.TrimSpace(strings.Split(forwarded, ",")[0])
		}
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			return realIP
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeText(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, message)
}
