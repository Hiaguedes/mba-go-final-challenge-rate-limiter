package main

import (
	"context"
	"log"

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
		errorLog := exceptions.NewLogConfigException("error loading configuration", err)
		log.Fatal("Error loading configuration: ", errorLog)
	}

	databaseStrategy := database.NewDatabaseStrategy[*redis.Client]()

	redis := redis_db.NewRedis()
	databaseStrategy.SetStrategy(redis)

	_, err = databaseStrategy.LoadDatabase(ctx)
	if err != nil {
		errorLog := exceptions.NewOpenDatabaseException("error on open database", err)
		log.Fatal(errorLog)
		return
	}

	ws := webserver.NewWebServer(":" + cfg.ServerPort)

	dumbServer := controller.NewDumbServer()
	ws.AddHandler("GET /", dumbServer.DumbServerController)

	ws.Start()
}