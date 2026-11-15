package redis_db

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {}

func NewRedis() *Redis {
	return &Redis{}
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
