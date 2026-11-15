package controller

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/exceptions"
)

type MiddleWareRateLimitController struct{}

func NewRateLimitMiddleware() *MiddleWareRateLimitController {
	return &MiddleWareRateLimitController{}
}

func (m *MiddleWareRateLimitController) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Executando rate limiter")

		err := exceptions.NewRateLimitException("Rate Limit exceeded")
		w.WriteHeader(err.Code)
		json.NewEncoder(w).Encode(err)

		// liberar next se tiver ok pra prosseguir
		// next.ServeHTTP(w, r)
	})
}
