package controller_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/controller"
)

const (
	testToken    = "token_de_teste"
	blockSeconds = 300
)

// fakeLimiter simula a porta de persistência em memória: deixa passar até `limit`
// requisições por identidade e bloqueia a partir daí. A janela de 1s e o TTL real do
// bloqueio pertencem à camada Redis e são cobertos pelos testes de integração.
type fakeLimiter struct {
	mu      sync.Mutex
	counts  map[string]int
	blocked map[string]bool
	calls   []allowCall
	err     error
}

type allowCall struct {
	identifier   string
	limit        int
	blockSeconds int
}

func newFakeLimiter() *fakeLimiter {
	return &fakeLimiter{counts: map[string]int{}, blocked: map[string]bool{}}
}

func (f *fakeLimiter) allow(_ context.Context, identifier string, limit, blockSeconds int, _ struct{}) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, allowCall{identifier, limit, blockSeconds})
	if f.err != nil {
		return false, f.err
	}
	if f.blocked[identifier] {
		return false, nil
	}
	f.counts[identifier]++
	if f.counts[identifier] > limit {
		f.blocked[identifier] = true
		return false, nil
	}
	return true, nil
}

func (f *fakeLimiter) lastCall(t *testing.T) allowCall {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		t.Fatal("limiter was not called")
	}
	return f.calls[len(f.calls)-1]
}

func newHandler(cfg *controller.Config, fl *fakeLimiter) http.Handler {
	mw := controller.NewRateLimitMiddleware(struct{}{}, cfg, fl.allow)
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mw.RateLimitMiddleware(ok)
}

// send faz uma requisição GET vinda de `ip` com os headers informados.
func send(h http.Handler, ip string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip + ":40000"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func withToken(token string) map[string]string {
	return map[string]string{"API_KEY": token}
}

func TestIPLimitIsEnforcedPerIP(t *testing.T) {
	cfg := controller.NewConfig(3, 100, blockSeconds, testToken, false)
	h := newHandler(cfg, newFakeLimiter())

	for i := 1; i <= 3; i++ {
		if rec := send(h, "10.0.0.1", nil); rec.Code != http.StatusOK {
			t.Fatalf("request %d from IP within limit: got %d, want 200", i, rec.Code)
		}
	}

	rec := send(h, "10.0.0.1", nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("request above IP limit: got %d, want 429", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "maximum number of requests") {
		t.Errorf("429 body = %q, want rate limit message", rec.Body.String())
	}

	// Outro IP tem a própria cota.
	if rec := send(h, "10.0.0.2", nil); rec.Code != http.StatusOK {
		t.Fatalf("different IP: got %d, want 200", rec.Code)
	}
}

func TestValidTokenUsesTokenLimitInsteadOfIPLimit(t *testing.T) {
	// IP limit 2, token limit 5: com token válido, a 3ª requisição já não pode ser barrada pelo IP.
	cfg := controller.NewConfig(2, 5, blockSeconds, testToken, false)
	fl := newFakeLimiter()
	h := newHandler(cfg, fl)

	for i := 1; i <= 5; i++ {
		if rec := send(h, "10.0.0.1", withToken(testToken)); rec.Code != http.StatusOK {
			t.Fatalf("token request %d: got %d, want 200 (token limit is 5)", i, rec.Code)
		}
	}

	if rec := send(h, "10.0.0.1", withToken(testToken)); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("token request above token limit: got %d, want 429", rec.Code)
	}

	call := fl.lastCall(t)
	if !strings.HasPrefix(call.identifier, "token:") {
		t.Errorf("identifier = %q, want token: prefix", call.identifier)
	}
	if call.limit != 5 {
		t.Errorf("limit = %d, want token limit 5", call.limit)
	}
	if call.blockSeconds != blockSeconds {
		t.Errorf("blockSeconds = %d, want %d", call.blockSeconds, blockSeconds)
	}
}

func TestTokenIdentityIsNotTiedToIP(t *testing.T) {
	// Mesmo token vindo de IPs diferentes compartilha a mesma identidade (e cota).
	cfg := controller.NewConfig(1, 2, blockSeconds, testToken, false)
	fl := newFakeLimiter()
	h := newHandler(cfg, fl)

	if rec := send(h, "10.0.0.1", withToken(testToken)); rec.Code != http.StatusOK {
		t.Fatalf("first request: got %d, want 200", rec.Code)
	}
	if rec := send(h, "10.0.0.2", withToken(testToken)); rec.Code != http.StatusOK {
		t.Fatalf("second request from other IP: got %d, want 200", rec.Code)
	}
	if rec := send(h, "10.0.0.3", withToken(testToken)); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("third request with same token: got %d, want 429", rec.Code)
	}
}

func TestValidTokenIsNotAffectedByBlockedIP(t *testing.T) {
	cfg := controller.NewConfig(1, 100, blockSeconds, testToken, false)
	h := newHandler(cfg, newFakeLimiter())

	if rec := send(h, "10.0.0.1", nil); rec.Code != http.StatusOK {
		t.Fatalf("first request without token: got %d, want 200", rec.Code)
	}
	if rec := send(h, "10.0.0.1", nil); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("IP over limit: got %d, want 429", rec.Code)
	}

	// IP está bloqueado, mas o token válido tem precedência e segue o próprio limite.
	if rec := send(h, "10.0.0.1", withToken(testToken)); rec.Code != http.StatusOK {
		t.Fatalf("valid token from blocked IP: got %d, want 200", rec.Code)
	}
	// Sem token, o IP continua bloqueado.
	if rec := send(h, "10.0.0.1", nil); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("IP without token after valid-token request: got %d, want 429", rec.Code)
	}
}

func TestInvalidOrMissingTokenFallsBackToIPLimit(t *testing.T) {
	cases := map[string]map[string]string{
		"sem header":       nil,
		"header vazio":     withToken(""),
		"token errado":     withToken("token_errado"),
		"token com espaço": withToken(" " + testToken),
		"maiúsculas":       withToken(strings.ToUpper(testToken)),
	}

	for name, headers := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := controller.NewConfig(2, 100, blockSeconds, testToken, false)
			fl := newFakeLimiter()
			h := newHandler(cfg, fl)

			for i := 1; i <= 2; i++ {
				if rec := send(h, "10.0.0.9", headers); rec.Code != http.StatusOK {
					t.Fatalf("request %d: got %d, want 200", i, rec.Code)
				}
			}
			if rec := send(h, "10.0.0.9", headers); rec.Code != http.StatusTooManyRequests {
				t.Fatalf("third request: got %d, want 429 (IP limit 2)", rec.Code)
			}

			call := fl.lastCall(t)
			if call.identifier != "ip:10.0.0.9" {
				t.Errorf("identifier = %q, want ip:10.0.0.9", call.identifier)
			}
			if call.limit != 2 {
				t.Errorf("limit = %d, want IP limit 2", call.limit)
			}
		})
	}
}

func TestEmptyConfiguredTokenNeverGrantsTokenLimit(t *testing.T) {
	cfg := controller.NewConfig(2, 100, blockSeconds, "", false)
	fl := newFakeLimiter()
	h := newHandler(cfg, fl)

	send(h, "10.0.0.5", nil)
	if call := fl.lastCall(t); call.identifier != "ip:10.0.0.5" || call.limit != 2 {
		t.Errorf("got identifier=%q limit=%d, want ip:10.0.0.5 limit 2", call.identifier, call.limit)
	}
}

func TestTokenIsHashedBeforeReachingLimiter(t *testing.T) {
	cfg := controller.NewConfig(2, 100, blockSeconds, testToken, false)
	fl := newFakeLimiter()
	h := newHandler(cfg, fl)

	send(h, "10.0.0.1", withToken(testToken))
	call := fl.lastCall(t)

	if strings.Contains(call.identifier, testToken) {
		t.Errorf("identifier %q leaks the raw token", call.identifier)
	}
	// "token:" + sha256 em hex (64 caracteres).
	if len(call.identifier) != len("token:")+64 {
		t.Errorf("identifier %q has unexpected length", call.identifier)
	}
}

func TestForwardedHeadersIgnoredWithoutTrustedProxy(t *testing.T) {
	cfg := controller.NewConfig(2, 100, blockSeconds, testToken, false)
	fl := newFakeLimiter()
	h := newHandler(cfg, fl)

	send(h, "10.0.0.1", map[string]string{"X-Forwarded-For": "203.0.113.7"})
	if call := fl.lastCall(t); call.identifier != "ip:10.0.0.1" {
		t.Errorf("identifier = %q, want ip:10.0.0.1 (XFF must be ignored)", call.identifier)
	}
}

func TestForwardedForUsedWhenProxyIsTrusted(t *testing.T) {
	cfg := controller.NewConfig(2, 100, blockSeconds, testToken, true)
	fl := newFakeLimiter()
	h := newHandler(cfg, fl)

	send(h, "10.0.0.1", map[string]string{"X-Forwarded-For": "203.0.113.7, 10.0.0.1"})
	if call := fl.lastCall(t); call.identifier != "ip:203.0.113.7" {
		t.Errorf("identifier = %q, want ip:203.0.113.7 (first XFF entry)", call.identifier)
	}
}

func TestLimiterErrorReturns500NotTooManyRequests(t *testing.T) {
	cfg := controller.NewConfig(2, 100, blockSeconds, testToken, false)
	fl := newFakeLimiter()
	fl.err = errors.New("redis down")
	h := newHandler(cfg, fl)

	if rec := send(h, "10.0.0.1", nil); rec.Code != http.StatusInternalServerError {
		t.Fatalf("backend failure: got %d, want 500", rec.Code)
	}
}
