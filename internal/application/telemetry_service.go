package application

import (
	"context"

	"github.com/drivervision/api/internal/domain"
	"github.com/google/uuid"
)

type TelemetryRepository interface {
	SaveTelemetry(context.Context, domain.DeviceTelemetry) (domain.DeviceTelemetry, error)
	ListLatestTelemetry(context.Context, int) ([]domain.DeviceTelemetry, error)
}

type TelemetryService struct{ repository TelemetryRepository }

func NewTelemetryService(repository TelemetryRepository) *TelemetryService {
	return &TelemetryService{repository: repository}
}

func (service *TelemetryService) Report(ctx context.Context, telemetry domain.DeviceTelemetry) (domain.DeviceTelemetry, error) {
	if telemetry.ID == "" {
		telemetry.ID = uuid.NewString()
	}
	if err := telemetry.Validate(); err != nil {
		return domain.DeviceTelemetry{}, err
	}
	return service.repository.SaveTelemetry(ctx, telemetry)
}

func (service *TelemetryService) Latest(ctx context.Context, limit int) ([]domain.DeviceTelemetry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return service.repository.ListLatestTelemetry(ctx, limit)
}
