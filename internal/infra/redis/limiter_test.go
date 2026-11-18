package redis_db

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// Estes testes exigem um Redis real. Defina REDIS_ADDR (ex: localhost:6379) para executá-los;
// sem a variável, são pulados. Use `make test-integration` para subir o Redis via Docker.
func newTestClient(t *testing.T) (*Redis, func(string) string) {
	t.Helper()
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR não definido; pulando teste de integração com Redis")
	}

	r := NewRedis(addr)
	client, err := r.LoadDatabase(context.Background())
	if err != nil {
		t.Fatalf("conectar ao Redis: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	// Cada teste usa identificadores únicos para não sofrer interferência de execuções anteriores.
	prefix := fmt.Sprintf("test:%d:", time.Now().UnixNano())
	return r, func(name string) string { return prefix + name }
}

func allow(t *testing.T, r *Redis, id string, limit, block int) bool {
	t.Helper()
	client, err := r.LoadDatabase(context.Background())
	if err != nil {
		t.Fatalf("conectar ao Redis: %v", err)
	}
	defer client.Close()

	ok, err := r.Allow(context.Background(), id, limit, block, client)
	if err != nil {
		t.Fatalf("Allow: %v", err)
	}
	return ok
}

func TestAllowNeverBlocksWithinLimit(t *testing.T) {
	r, id := newTestClient(t)
	key := id("within-limit")

	// 5 requisições com limite 5 nunca podem ultrapassar o limite em uma janela de 1s,
	// mesmo que a rajada atravesse a fronteira de um segundo.
	for i := 1; i <= 5; i++ {
		if !allow(t, r, key, 5, 300) {
			t.Fatalf("request %d within limit was denied", i)
		}
	}
}

func TestAllowBlocksAboveLimitAndUnblocksAfterBlockTime(t *testing.T) {
	r, id := newTestClient(t)
	key := id("block")

	// Rajada de 10 requisições com limite 3: a primeira passa e a última é bloqueada,
	// pois depois de estourar o limite a identidade fica bloqueada pelo tempo configurado.
	results := make([]bool, 10)
	for i := range results {
		results[i] = allow(t, r, key, 3, 1)
	}
	if !results[0] {
		t.Fatal("first request was denied")
	}
	if results[len(results)-1] {
		t.Fatal("last request of the burst was allowed; limit was not enforced")
	}

	// Ainda dentro do bloqueio de 1s: continua negado.
	if allow(t, r, key, 3, 1) {
		t.Fatal("request during block time was allowed")
	}

	// Depois do tempo de bloqueio, a identidade volta a ser atendida.
	time.Sleep(1500 * time.Millisecond)
	if !allow(t, r, key, 3, 1) {
		t.Fatal("request after block time was still denied")
	}
}

func TestAllowWindowResetsEachSecond(t *testing.T) {
	r, id := newTestClient(t)
	key := id("window")

	// Duas requisições no limite de 2 por segundo, depois espera a próxima janela:
	// o contador recomeça e não há bloqueio, pois o limite nunca foi ultrapassado.
	for i := 1; i <= 2; i++ {
		if !allow(t, r, key, 2, 300) {
			t.Fatalf("request %d in first window was denied", i)
		}
	}

	next := time.Now().Truncate(time.Second).Add(time.Second + 50*time.Millisecond)
	time.Sleep(time.Until(next))

	for i := 1; i <= 2; i++ {
		if !allow(t, r, key, 2, 300) {
			t.Fatalf("request %d in next window was denied", i)
		}
	}
}
