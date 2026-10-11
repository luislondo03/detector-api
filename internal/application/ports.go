package application

import (
	"context"
	"io"
	"time"

	"github.com/drivervision/api/internal/domain"
)

type IncidentFilter struct {
	Type   string
	From   *time.Time
	To     *time.Time
	Limit  int
	Offset int
}

type IncidentRepository interface {
	Save(context.Context, domain.Incident) (domain.Incident, error)
	FindByID(context.Context, string) (domain.Incident, error)
	List(context.Context, IncidentFilter) ([]domain.Incident, int, error)
	AddEvidence(context.Context, domain.Evidence) error
}

type EvidenceStore interface {
	Save(context.Context, string, string, io.Reader) (string, error)
}
