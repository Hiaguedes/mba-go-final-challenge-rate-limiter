package redis_db

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// limiterScriptSrc é embutido no binário, então funciona na imagem Docker (scratch).
//
//go:embed limiter.lua
var limiterScriptSrc string

var limiterScript = redis.NewScript(limiterScriptSrc)

// counterTTLSeconds deixa o contador viver um pouco além da janela de 1s,
// para que ele não expire no meio da janela.
const counterTTLSeconds = 2

type Redis struct {
	Addr string
}

func NewRedis(addr string) *Redis {
	return &Redis{Addr: addr}
}

func (r Redis) LoadDatabase(ctx context.Context) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{Addr: r.Addr})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("could not connect to redis at %s: %w", r.Addr, err)
	}

	return rdb, nil
}

// Allow aplica a janela de 1 segundo e o bloqueio para a identidade informada.
func (r Redis) Allow(ctx context.Context, identifier string, limit int, blockSeconds int, client *redis.Client) (bool, error) {
	blockKey := "blocked:" + identifier
	countKey := fmt.Sprintf("rate:%s:%d", identifier, time.Now().Unix())

	allowed, err := limiterScript.Run(
		ctx,
		client,
		[]string{blockKey, countKey},
		limit,
		blockSeconds,
		counterTTLSeconds,
	).Int()
	if err != nil {
		return false, fmt.Errorf("run rate limit script: %w", err)
	}

	return allowed == 1, nil
}
