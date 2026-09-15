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
	repository, err := store.OpenPostgresExperimentStore(cfg.DatabaseURL, store.PostgresConfig{
		MaxOpenConns: cfg.DBMaxOpenConns,
		MaxIdleConns: cfg.DBMaxIdleConns,
	})
	if err != nil {
		log.Fatalf("open postgres store: %v", err)
	}
	defer func() {
		if err := repository.Close(); err != nil {
			log.Printf("close postgres store: %v", err)
		}
	}()

	experimentService := service.NewExperimentService(repository)
	experimentHandler := handler.NewExperimentHandler(experimentService)
	engine := router.New(experimentHandler)

	log.Printf("experiment service listening on %s with postgres storage", cfg.Port)
	if err := engine.Run(cfg.Port); err != nil {
		log.Fatal(err)
	}
}
