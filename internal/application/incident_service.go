package application

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/drivervision/api/internal/domain"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("incident not found")

type IncidentService struct {
	repository IncidentRepository
	files      EvidenceStore
	now        func() time.Time
}

func NewIncidentService(repository IncidentRepository, files EvidenceStore) *IncidentService {
	return &IncidentService{repository: repository, files: files, now: time.Now}
}

func (service *IncidentService) Report(ctx context.Context, incident domain.Incident) (domain.Incident, error) {
	if err := incident.Validate(); err != nil {
		return domain.Incident{}, err
	}
	if len(incident.Details) == 0 {
		incident.Details = []byte("{}")
	}
	return service.repository.Save(ctx, incident)
}

func (service *IncidentService) Get(ctx context.Context, incidentID string) (domain.Incident, error) {
	return service.repository.FindByID(ctx, incidentID)
}

func (service *IncidentService) List(ctx context.Context, filter IncidentFilter) ([]domain.Incident, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 30
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return service.repository.List(ctx, filter)
}

func (service *IncidentService) AttachEvidence(ctx context.Context, incidentID string, consent bool, mediaType string, content io.Reader) (domain.Evidence, error) {
	if !consent {
		return domain.Evidence{}, errors.New("evidence consent is required")
	}
	if _, err := service.repository.FindByID(ctx, incidentID); err != nil {
		return domain.Evidence{}, err
	}
	evidenceID := uuid.NewString()
	path, err := service.files.Save(ctx, evidenceID, mediaType, content)
	if err != nil {
		return domain.Evidence{}, err
	}
	evidence := domain.Evidence{
		ID: evidenceID, IncidentID: incidentID, MediaType: mediaType,
		URL: "/evidence/" + path, CreatedAt: service.now().UTC(),
	}
	if err := service.repository.AddEvidence(ctx, evidence); err != nil {
		return domain.Evidence{}, err
	}
	return evidence, nil
}
