package redis_db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	IpLimitRequest int
	TokenLimitRequests int 
	BlockTime time.Duration
}

func NewRedis(ipLimitRequest int, tokenLimitRequests int, blockTime time.Duration) *Redis {
	return &Redis{
		IpLimitRequest: ipLimitRequest,
		TokenLimitRequests: tokenLimitRequests,
		BlockTime: blockTime,
	}
}

func (r Redis) LoadDatabase(ctx context.Context) (*redis.Client, error) {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
		return nil, err
	}

		// Set a key with a 10-minute expiration
	err = rdb.Set(ctx, "greeting", "Hello from Go and Docker!", 10* time.Minute).Err()
	if err != nil {
		log.Fatalf("Failed to set key: %v", err)
		return nil , err
	}

	return rdb, nil
}

func LoadLuaScript(path string) (string, error) {
scriptBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(scriptBytes), nil
}

func (r Redis) Allow(ctx context.Context, identifier string, isToken bool, client *redis.Client) (bool, error) {
	scriptContent, err := LoadLuaScript("scripts/config.lua")
	if err != nil {
		return false, errors.New("Error on open Lua File")
	}
	now := float64(time.Now().UnixNano()) / 1e9 // Timestamp atual em segundos (com precisão decimal)
	requested := 1
	blockTimeSec := int(r.BlockTime.Seconds())
	key := fmt.Sprintf("rate_limiter:%s", identifier)

	limit := r.IpLimitRequest
	if isToken {
		limit = r.TokenLimitRequests
	}

script := redis.NewScript(scriptContent)
	res, err := script.Run(
		ctx,
		client,
		[]string{key},
		limit,
		blockTimeSec,
		now,
		requested,
	).Int()
	if err != nil {
		internalError := errors.New("error on exec lua script")
		return false, internalError
	}

	return res == 1, nil
}
