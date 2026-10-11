package postgres

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/drivervision/api/internal/application"
	"github.com/drivervision/api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_initial.sql
var initialSchema string

type Repository struct{ pool *pgxpool.Pool }

func New(ctx context.Context, databaseURL string) (*Repository, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if _, err := pool.Exec(ctx, initialSchema); err != nil {
		pool.Close()
		return nil, err
	}
	return &Repository{pool: pool}, nil
}

func (repository *Repository) Close() { repository.pool.Close() }

func (repository *Repository) Save(ctx context.Context, incident domain.Incident) (domain.Incident, error) {
	row := repository.pool.QueryRow(ctx, `
		INSERT INTO incidents (event_id, session_id, type, lifecycle, severity, started_at, ended_at, duration_ms, confidence, observation_quality, detector_version, model_version, threshold_profile_id, details, alert_triggered)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (event_id) DO UPDATE SET lifecycle=EXCLUDED.lifecycle, severity=EXCLUDED.severity, ended_at=EXCLUDED.ended_at, duration_ms=EXCLUDED.duration_ms, confidence=EXCLUDED.confidence, observation_quality=EXCLUDED.observation_quality, details=EXCLUDED.details, alert_triggered=EXCLUDED.alert_triggered, updated_at=now()
		RETURNING event_id, session_id, type, lifecycle, severity, started_at, ended_at, duration_ms, confidence, observation_quality, detector_version, model_version, threshold_profile_id, details, alert_triggered, created_at, updated_at`,
		incident.EventID, incident.SessionID, incident.Type, incident.Lifecycle, incident.Severity, incident.StartedAt, incident.EndedAt, incident.DurationMilliseconds, incident.Confidence, incident.ObservationQuality, incident.DetectorVersion, incident.ModelVersion, incident.ThresholdProfileID, incident.Details, incident.AlertTriggered,
	)
	return scanIncident(row)
}

func (repository *Repository) FindByID(ctx context.Context, incidentID string) (domain.Incident, error) {
	row := repository.pool.QueryRow(ctx, `SELECT event_id, session_id, type, lifecycle, severity, started_at, ended_at, duration_ms, confidence, observation_quality, detector_version, model_version, threshold_profile_id, details, alert_triggered, created_at, updated_at FROM incidents WHERE event_id=$1`, incidentID)
	incident, err := scanIncident(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Incident{}, application.ErrNotFound
	}
	if err != nil {
		return domain.Incident{}, err
	}
	evidence, err := repository.evidenceFor(ctx, incidentID)
	if err != nil {
		return domain.Incident{}, err
	}
	incident.Evidence = evidence
	return incident, nil
}

func (repository *Repository) List(ctx context.Context, filter application.IncidentFilter) ([]domain.Incident, int, error) {
	clauses := []string{"TRUE"}
	args := []any{}
	if filter.Type != "" {
		args = append(args, filter.Type)
		clauses = append(clauses, fmt.Sprintf("type=$%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		clauses = append(clauses, fmt.Sprintf("started_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		clauses = append(clauses, fmt.Sprintf("started_at <= $%d", len(args)))
	}
	where := strings.Join(clauses, " AND ")
	var total int
	if err := repository.pool.QueryRow(ctx, "SELECT count(*) FROM incidents WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, filter.Limit, filter.Offset)
	query := "SELECT event_id, session_id, type, lifecycle, severity, started_at, ended_at, duration_ms, confidence, observation_quality, detector_version, model_version, threshold_profile_id, details, alert_triggered, created_at, updated_at FROM incidents WHERE " + where + fmt.Sprintf(" ORDER BY started_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := repository.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	incidents := make([]domain.Incident, 0)
	for rows.Next() {
		incident, err := scanIncident(rows)
		if err != nil {
			return nil, 0, err
		}
		incidents = append(incidents, incident)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	evidenceByIncident, err := repository.evidenceForIncidents(ctx, incidents)
	if err != nil {
		return nil, 0, err
	}
	for index := range incidents {
		incidents[index].Evidence = evidenceByIncident[incidents[index].EventID]
	}
	return incidents, total, nil
}

func (repository *Repository) AddEvidence(ctx context.Context, evidence domain.Evidence) error {
	_, err := repository.pool.Exec(ctx, `INSERT INTO incident_evidence (id, incident_id, media_type, url, created_at) VALUES ($1,$2,$3,$4,$5)`, evidence.ID, evidence.IncidentID, evidence.MediaType, evidence.URL, evidence.CreatedAt)
	return err
}

func (repository *Repository) evidenceFor(ctx context.Context, incidentID string) ([]domain.Evidence, error) {
	rows, err := repository.pool.Query(ctx, `SELECT id, incident_id, media_type, url, created_at FROM incident_evidence WHERE incident_id=$1 ORDER BY created_at`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	evidence := make([]domain.Evidence, 0)
	for rows.Next() {
		var item domain.Evidence
		if err := rows.Scan(&item.ID, &item.IncidentID, &item.MediaType, &item.URL, &item.CreatedAt); err != nil {
			return nil, err
		}
		evidence = append(evidence, item)
	}
	return evidence, rows.Err()
}

// evidenceForIncidents avoids one database query per row in the list endpoint.
func (repository *Repository) evidenceForIncidents(ctx context.Context, incidents []domain.Incident) (map[string][]domain.Evidence, error) {
	evidenceByIncident := make(map[string][]domain.Evidence, len(incidents))
	incidentIDs := make([]string, 0, len(incidents))
	for _, incident := range incidents {
		evidenceByIncident[incident.EventID] = make([]domain.Evidence, 0)
		incidentIDs = append(incidentIDs, incident.EventID)
	}
	if len(incidentIDs) == 0 {
		return evidenceByIncident, nil
	}

	rows, err := repository.pool.Query(ctx, `
		SELECT id, incident_id, media_type, url, created_at
		FROM incident_evidence
		WHERE incident_id = ANY($1)
		ORDER BY created_at`, incidentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item domain.Evidence
		if err := rows.Scan(&item.ID, &item.IncidentID, &item.MediaType, &item.URL, &item.CreatedAt); err != nil {
			return nil, err
		}
		evidenceByIncident[item.IncidentID] = append(evidenceByIncident[item.IncidentID], item)
	}
	return evidenceByIncident, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scanIncident(row rowScanner) (domain.Incident, error) {
	incident := domain.Incident{Evidence: make([]domain.Evidence, 0)}
	var details json.RawMessage
	err := row.Scan(&incident.EventID, &incident.SessionID, &incident.Type, &incident.Lifecycle, &incident.Severity, &incident.StartedAt, &incident.EndedAt, &incident.DurationMilliseconds, &incident.Confidence, &incident.ObservationQuality, &incident.DetectorVersion, &incident.ModelVersion, &incident.ThresholdProfileID, &details, &incident.AlertTriggered, &incident.CreatedAt, &incident.UpdatedAt)
	incident.Details = details
	return incident, err
}
