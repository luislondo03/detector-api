package main

import (
	"context"
	"log"
	"time"

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
	go runTelemetryRetention(repository, cfg.TelemetryRetentionDays)
	store, err := filesystem.NewEvidenceStore(cfg.EvidenceDirectory)
	if err != nil {
		log.Fatal(err)
	}
	service := application.NewIncidentService(repository, store)
	telemetryService := application.NewTelemetryService(repository)
	enterpriseService := application.NewEnterpriseService(repository)
	app := httpadapter.NewApp(httpadapter.NewHandler(service, telemetryService, enterpriseService, cfg.AdminBootstrapKey), cfg.AllowedOrigin, cfg.EvidenceDirectory)
	log.Fatal(app.Listen(":" + cfg.Port))
}

func runTelemetryRetention(repository *postgres.Repository, days int) {
	if days <= 0 {
		return
	}
	purge := func() {
		if err := repository.PurgeTelemetryBefore(context.Background(), time.Now().UTC().AddDate(0, 0, -days)); err != nil {
			log.Printf("telemetry retention cleanup failed: %v", err)
		}
	}
	purge()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		purge()
	}
}
