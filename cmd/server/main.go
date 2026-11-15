package main

import (
	"context"
	"log"
	"net/http"

	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/config"
	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/controller"
	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/exceptions"
	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/infra/database"
	redis_db "github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/infra/redis"
	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/infra/webserver"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfig(".")
	if err != nil {
		errorLog := exceptions.NewConfigException("error loading configuration", err)
		log.Fatal("Error loading configuration: ", errorLog)
	}

	databaseStrategy := database.NewDatabaseStrategy[*redis.Client]()

	redis := redis_db.NewRedis(
		cfg.IpRequestsPerSecondLimitter, 
		cfg.ApiTokenRequestsPerSecondLimitter, 
		cfg.RequestBlockTime,
	)

	databaseStrategy.SetStrategy(redis)

	client, err := databaseStrategy.LoadDatabase(ctx)
	if err != nil {
		errorLog := exceptions.NewOpenDatabaseException("error on open database", err)
		log.Fatal(errorLog)
		return
	}

	ws := webserver.NewWebServer(":" + cfg.ServerPort)

	configMiddleware := controller.NewConfig(cfg.IpRequestsPerSecondLimitter, cfg.ApiTokenRequestsPerSecondLimitter, cfg.RequestBlockTime)
	rateLimitterMiddleware := controller.NewRateLimitMiddleware(
		client, 
		configMiddleware, 
		databaseStrategy.Allow,
	)

	dumbServer := controller.NewDumbServer().DumbServerController
	handler := rateLimitterMiddleware.RateLimitMiddleware(http.HandlerFunc(dumbServer))
	ws.AddHandler("GET /", handler.ServeHTTP)

	ws.Start()
}