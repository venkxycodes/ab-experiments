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
	repository, closeRepository := buildRepository(cfg)
	defer closeRepository()

	experimentService := service.NewExperimentService(repository)
	experimentHandler := handler.NewExperimentHandler(experimentService)
	engine := router.New(experimentHandler)

	log.Printf("experiment service listening on %s using %s storage", cfg.Port, cfg.StorageBackend)
	if err := engine.Run(cfg.Port); err != nil {
		log.Fatal(err)
	}
}

func buildRepository(cfg config.Config) (store.ExperimentStore, func()) {
	switch cfg.StorageBackend {
	case "memory":
		return store.NewInMemoryExperimentStore(), func() {}
	case "postgres":
		repository, err := store.OpenPostgresExperimentStore(cfg.DatabaseURL, store.PostgresConfig{
			MaxOpenConns: cfg.DBMaxOpenConns,
			MaxIdleConns: cfg.DBMaxIdleConns,
		})
		if err != nil {
			log.Fatalf("open postgres store: %v", err)
		}
		return repository, func() {
			if err := repository.Close(); err != nil {
				log.Printf("close postgres store: %v", err)
			}
		}
	default:
		log.Fatalf("unsupported STORAGE_BACKEND %q; expected memory or postgres", cfg.StorageBackend)
		return nil, func() {}
	}
}
