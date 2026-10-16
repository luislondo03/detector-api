package application

import (
	"context"
	"testing"
)

type stubEnterpriseRepository struct {
	EnterpriseRepository
	captured FleetEventFilter
}

func (stub *stubEnterpriseRepository) ListEvents(_ context.Context, filter FleetEventFilter) ([]FleetEvent, int, error) {
	stub.captured = filter
	return []FleetEvent{}, 0, nil
}

func TestEnterpriseEventsClampsPagination(t *testing.T) {
	repository := &stubEnterpriseRepository{}
	service := NewEnterpriseService(repository)

	if _, _, err := service.Events(context.Background(), FleetEventFilter{Limit: 0, Offset: -3}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repository.captured.Limit != 50 || repository.captured.Offset != 0 {
		t.Fatalf("expected default limit 50 and offset 0, got limit=%d offset=%d", repository.captured.Limit, repository.captured.Offset)
	}

	if _, _, err := service.Events(context.Background(), FleetEventFilter{Limit: 500, Offset: 20}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repository.captured.Limit != 50 || repository.captured.Offset != 20 {
		t.Fatalf("expected over-limit clamped to 50 with offset 20, got limit=%d offset=%d", repository.captured.Limit, repository.captured.Offset)
	}
}
