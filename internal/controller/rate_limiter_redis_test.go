package controller_test

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/controller"
	redis_db "github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/infra/redis"
)

func randomHex(t *testing.T, n int) string {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random: %v", err)
	}
	return hex.EncodeToString(b)
}

// TestRedisPrecedenceTokenOverIP exercita o middleware completo contra um Redis real.
// Requer REDIS_ADDR; sem a variável o teste é pulado (use `make test-integration`).
func TestRedisPrecedenceTokenOverIP(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR não definido; pulando teste de integração com Redis")
	}

	strategy := redis_db.NewRedis(addr)
	client, err := strategy.LoadDatabase(t.Context())
	if err != nil {
		t.Fatalf("conectar ao Redis: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	// Token e IP únicos por execução, para que bloqueios antigos no Redis não interfiram.
	token := "token-" + randomHex(t, 8)
	octets := make([]byte, 3)
	if _, err := rand.Read(octets); err != nil {
		t.Fatalf("random: %v", err)
	}
	ip := fmt.Sprintf("10.%d.%d.%d", octets[0], octets[1], octets[2])

	// Limite por IP = 2 req/s, limite por token = 5 req/s, bloqueio de 30s.
	cfg := controller.NewConfig(2, 5, 30, token, false)
	mw := controller.NewRateLimitMiddleware(client, cfg, strategy.Allow)
	h := mw.RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Uma rajada grande sem token estoura o limite do IP. Depois de estourar, o IP fica bloqueado,
	// então a última requisição da rajada precisa ser 429 (independe de em qual janela de 1s caiu).
	var last int
	for i := 0; i < 10; i++ {
		last = send(h, ip, nil).Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("IP burst above limit: last status = %d, want 429", last)
	}

	// Mesmo IP bloqueado, com token válido a requisição passa (token > IP).
	// São 3 requisições, abaixo do limite de token em qualquer janela de 1s.
	for i := 1; i <= 3; i++ {
		if rec := send(h, ip, withToken(token)); rec.Code != http.StatusOK {
			t.Fatalf("valid token request %d from blocked IP: got %d, want 200", i, rec.Code)
		}
	}

	// O token também tem limite próprio: uma rajada acima de 5 precisa terminar em 429.
	for i := 0; i < 20; i++ {
		last = send(h, ip, withToken(token)).Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("token burst above token limit: last status = %d, want 429", last)
	}

	// Sem token, o IP continua bloqueado.
	if rec := send(h, ip, nil); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("IP without token after token traffic: got %d, want 429", rec.Code)
	}
}
