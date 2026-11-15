package main

import (
	"log"

	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/config"
	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/controller"
	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/infra/webserver"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal("Error loading configuration: ", err)
	}

	ws := webserver.NewWebServer(":" + cfg.ServerPort)

	dumbServer := controller.NewDumbServer()
	ws.AddHandler("GET /", dumbServer.DumbServerController)

	ws.Start()
}