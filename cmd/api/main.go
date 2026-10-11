package main

import (
	"context"
	"log"

	"github.com/drivervision/api/internal/adapters/filesystem"
	httpadapter "github.com/drivervision/api/internal/adapters/http"
	"github.com/drivervision/api/internal/adapters/postgres"
	"github.com/drivervision/api/internal/application"
	"github.com/drivervision/api/internal/platform/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	repository, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer repository.Close()
	store, err := filesystem.NewEvidenceStore(cfg.EvidenceDirectory)
	if err != nil {
		log.Fatal(err)
	}
	service := application.NewIncidentService(repository, store)
	app := httpadapter.NewApp(httpadapter.NewHandler(service), cfg.AllowedOrigin, cfg.EvidenceDirectory)
	log.Fatal(app.Listen(":" + cfg.Port))
}
