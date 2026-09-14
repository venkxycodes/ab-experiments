package main

import (
	"log"

	"github.com/venkxycodes/ab-experiments/internal/config"
	"github.com/venkxycodes/ab-experiments/internal/handler"
	"github.com/venkxycodes/ab-experiments/internal/router"
	"github.com/venkxycodes/ab-experiments/internal/service"
	"github.com/venkxycodes/ab-experiments/internal/store"
)

func main() {
	cfg := config.Load()
	repository := store.NewInMemoryExperimentStore()
	experimentService := service.NewExperimentService(repository)
	experimentHandler := handler.NewExperimentHandler(experimentService)
	engine := router.New(experimentHandler)

	log.Printf("experiment service listening on %s", cfg.Port)
	if err := engine.Run(cfg.Port); err != nil {
		log.Fatal(err)
	}
}
