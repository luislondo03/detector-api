package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/drivervision/api/internal/application"
	"github.com/drivervision/api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (repository *Repository) CreateCompany(ctx context.Context, company domain.Company) (domain.Company, error) {
	_, err := repository.pool.Exec(ctx, `INSERT INTO companies(id,name,timezone,created_at) VALUES ($1,$2,$3,$4)`, company.ID, company.Name, company.Timezone, company.CreatedAt)
	return company, err
}

func (repository *Repository) CreateVehicle(ctx context.Context, vehicle domain.Vehicle) (domain.Vehicle, error) {
	_, err := repository.pool.Exec(ctx, `INSERT INTO vehicles(id,company_id,plate,internal_code,status,created_at) VALUES ($1,$2,$3,$4,$5,$6)`, vehicle.ID, vehicle.CompanyID, vehicle.Plate, vehicle.InternalCode, vehicle.Status, vehicle.CreatedAt)
	return vehicle, err
}

func (repository *Repository) CreateDevice(ctx context.Context, device domain.Device) (domain.Device, error) {
	_, err := repository.pool.Exec(ctx, `INSERT INTO devices(id,company_id,label,hardware) VALUES ($1,$2,$3,$4)`, device.ID, device.CompanyID, device.Label, device.Hardware)
	return device, err
}

func (repository *Repository) CreateEnrollmentCode(ctx context.Context, code domain.EnrollmentCode) error {
	tag, err := repository.pool.Exec(ctx, `
		INSERT INTO enrollment_codes(id,company_id,vehicle_id,device_id,code_hash,expires_at)
		SELECT $1,$2,$3,$4,$5,$6 FROM devices d
		JOIN vehicles v ON v.id=$3 AND v.company_id=d.company_id
		WHERE d.id=$4 AND d.company_id=$2`, code.ID, code.CompanyID, code.VehicleID, code.DeviceID, code.Code, code.ExpiresAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("device and vehicle must belong to the company")
	}
	return nil
}

func (repository *Repository) RedeemEnrollment(ctx context.Context, codeHash, credentialHash string, now time.Time) (domain.EnrollmentResult, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.EnrollmentResult{}, err
	}
	defer tx.Rollback(ctx)
	var result domain.EnrollmentResult
	err = tx.QueryRow(ctx, `
		UPDATE enrollment_codes SET used_at=$2
		WHERE code_hash=$1 AND used_at IS NULL AND expires_at > $2
		RETURNING company_id, vehicle_id, device_id`, codeHash, now).Scan(&result.CompanyID, &result.VehicleID, &result.DeviceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.EnrollmentResult{}, application.ErrEnrollmentInvalid
	}
	if err != nil {
		return domain.EnrollmentResult{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE devices SET credential_hash=$2,enrolled_at=$3,revoked_at=NULL WHERE id=$1 AND company_id=$4`, result.DeviceID, credentialHash, now, result.CompanyID); err != nil {
		return domain.EnrollmentResult{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE device_vehicle_assignments SET ends_at=$2 WHERE device_id=$1 AND ends_at IS NULL`, result.DeviceID, now); err != nil {
		return domain.EnrollmentResult{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO device_vehicle_assignments(id,device_id,vehicle_id,starts_at) VALUES ($1,$2,$3,$4)`, uuid.NewString(), result.DeviceID, result.VehicleID, now); err != nil {
		return domain.EnrollmentResult{}, err
	}
	if err = tx.QueryRow(ctx, `SELECT plate FROM vehicles WHERE id=$1 AND company_id=$2`, result.VehicleID, result.CompanyID).Scan(&result.VehiclePlate); err != nil {
		return domain.EnrollmentResult{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_logs(id,company_id,device_id,action,resource_type,resource_id) VALUES($1,$2,$3,'DEVICE_ENROLLED','device',$3)`, uuid.NewString(), result.CompanyID, result.DeviceID); err != nil {
		return domain.EnrollmentResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.EnrollmentResult{}, err
	}
	return result, nil
}

func (repository *Repository) AuthenticateDevice(ctx context.Context, credentialHash string) (domain.DevicePrincipal, error) {
	var principal domain.DevicePrincipal
	err := repository.pool.QueryRow(ctx, `
		SELECT d.id,d.company_id,a.vehicle_id FROM devices d
		JOIN device_vehicle_assignments a ON a.device_id=d.id AND a.ends_at IS NULL
		WHERE d.credential_hash=$1 AND d.revoked_at IS NULL`, credentialHash).Scan(&principal.DeviceID, &principal.CompanyID, &principal.VehicleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DevicePrincipal{}, application.ErrDeviceUnauthorized
	}
	return principal, err
}

func (repository *Repository) RevokeDevice(ctx context.Context, deviceID string, now time.Time) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var companyID string
	if err = tx.QueryRow(ctx, `UPDATE devices SET revoked_at=$2,credential_hash=NULL WHERE id=$1 RETURNING company_id`, deviceID, now).Scan(&companyID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_logs(id,company_id,device_id,action,resource_type,resource_id) VALUES($1,$2,$3,'DEVICE_REVOKED','device',$3)`, uuid.NewString(), companyID, deviceID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repository *Repository) RecordEvent(ctx context.Context, principal domain.DevicePrincipal, event domain.DeviceEvent) (domain.BatchAck, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.BatchAck{}, err
	}
	defer tx.Rollback(ctx)
	if duplicate, err := claimMessage(ctx, tx, principal.DeviceID, event.MessageID); err != nil || duplicate {
		if err != nil {
			return domain.BatchAck{}, err
		}
		return duplicateAck(event.MessageID), tx.Commit(ctx)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO monitoring_sessions(session_id,company_id,device_id,vehicle_id,started_at) VALUES($1,$2,$3,$4,$5) ON CONFLICT(session_id) DO NOTHING`, event.SessionID, principal.CompanyID, principal.DeviceID, principal.VehicleID, event.StartedAt); err != nil {
		return domain.BatchAck{}, err
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO detection_events(event_id,company_id,device_id,vehicle_id,session_id,revision,type,lifecycle,severity,occurred_at,started_at,ended_at,duration_ms,confidence,observation_quality,detector_version,model_version)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT(event_id) DO UPDATE SET revision=EXCLUDED.revision,lifecycle=EXCLUDED.lifecycle,severity=EXCLUDED.severity,occurred_at=EXCLUDED.occurred_at,ended_at=EXCLUDED.ended_at,duration_ms=EXCLUDED.duration_ms,confidence=EXCLUDED.confidence,observation_quality=EXCLUDED.observation_quality,updated_at=now()
		WHERE EXCLUDED.revision > detection_events.revision`, event.EventID, principal.CompanyID, principal.DeviceID, principal.VehicleID, event.SessionID, event.EventRevision, event.Type, event.Lifecycle, event.Severity, event.OccurredAt, event.StartedAt, event.EndedAt, event.DurationMilliseconds, event.Confidence, event.ObservationQuality, event.DetectorVersion, event.ModelVersion)
	if err != nil {
		return domain.BatchAck{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.BatchAck{}, err
	}
	if tag.RowsAffected() == 0 {
		return domain.BatchAck{MessageID: event.MessageID, Status: "rejected", Code: "STALE_REVISION"}, nil
	}
	return acceptedAck(event.MessageID), nil
}

func (repository *Repository) RecordSession(ctx context.Context, principal domain.DevicePrincipal, session domain.MonitoringSession) (domain.BatchAck, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.BatchAck{}, err
	}
	defer tx.Rollback(ctx)
	if duplicate, err := claimMessage(ctx, tx, principal.DeviceID, session.MessageID); err != nil || duplicate {
		if err != nil {
			return domain.BatchAck{}, err
		}
		return duplicateAck(session.MessageID), tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `INSERT INTO monitoring_sessions(session_id,company_id,device_id,vehicle_id,started_at,ended_at,close_reason) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(session_id) DO UPDATE SET ended_at=COALESCE(EXCLUDED.ended_at,monitoring_sessions.ended_at),close_reason=CASE WHEN EXCLUDED.close_reason<>'' THEN EXCLUDED.close_reason ELSE monitoring_sessions.close_reason END,updated_at=now()`, session.SessionID, principal.CompanyID, principal.DeviceID, principal.VehicleID, session.StartedAt, session.EndedAt, session.CloseReason)
	if err != nil {
		return domain.BatchAck{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.BatchAck{}, err
	}
	return acceptedAck(session.MessageID), nil
}

func (repository *Repository) RecordCoverage(ctx context.Context, principal domain.DevicePrincipal, coverage domain.CoverageInterval) (domain.BatchAck, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.BatchAck{}, err
	}
	defer tx.Rollback(ctx)
	if duplicate, err := claimMessage(ctx, tx, principal.DeviceID, coverage.MessageID); err != nil || duplicate {
		if err != nil {
			return domain.BatchAck{}, err
		}
		return duplicateAck(coverage.MessageID), tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `INSERT INTO coverage_intervals(id,company_id,device_id,vehicle_id,session_id,started_at,ended_at,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, uuid.NewString(), principal.CompanyID, principal.DeviceID, principal.VehicleID, coverage.SessionID, coverage.StartedAt, coverage.EndedAt, coverage.Status)
	if err != nil {
		return domain.BatchAck{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.BatchAck{}, err
	}
	return acceptedAck(coverage.MessageID), nil
}

func (repository *Repository) RecordTelemetry(ctx context.Context, principal domain.DevicePrincipal, telemetry domain.TelemetryMessage) (domain.BatchAck, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.BatchAck{}, err
	}
	defer tx.Rollback(ctx)
	if duplicate, err := claimMessage(ctx, tx, principal.DeviceID, telemetry.MessageID); err != nil || duplicate {
		if err != nil {
			return domain.BatchAck{}, err
		}
		return duplicateAck(telemetry.MessageID), tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `INSERT INTO device_telemetry(id,device_id,session_id,captured_at,latitude,longitude,accuracy_meters,speed_kilometers_per_hour,bearing_degrees,provider,battery_percent,charging,manufacturer,model,android_version,app_version,company_id,vehicle_id,enterprise_device_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`, uuid.NewString(), principal.DeviceID, telemetry.SessionID, telemetry.OccurredAt, telemetry.Latitude, telemetry.Longitude, telemetry.AccuracyMeters, telemetry.SpeedKilometersPerHour, telemetry.BearingDegrees, telemetry.Provider, telemetry.BatteryPercent, telemetry.Charging, telemetry.Manufacturer, telemetry.Model, telemetry.AndroidVersion, telemetry.AppVersion, principal.CompanyID, principal.VehicleID, principal.DeviceID)
	if err != nil {
		return domain.BatchAck{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.BatchAck{}, err
	}
	return acceptedAck(telemetry.MessageID), nil
}

func (repository *Repository) RecordHeartbeat(ctx context.Context, principal domain.DevicePrincipal, heartbeat domain.DeviceHeartbeat) (domain.BatchAck, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.BatchAck{}, err
	}
	defer tx.Rollback(ctx)
	if duplicate, err := claimMessage(ctx, tx, principal.DeviceID, heartbeat.MessageID); err != nil || duplicate {
		if err != nil {
			return domain.BatchAck{}, err
		}
		return duplicateAck(heartbeat.MessageID), tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `INSERT INTO device_heartbeats(id,company_id,device_id,vehicle_id,occurred_at,operation_status,quality_status,camera_status,app_version,model_version,pending_messages,storage_available_bytes) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, uuid.NewString(), principal.CompanyID, principal.DeviceID, principal.VehicleID, heartbeat.OccurredAt, heartbeat.OperationStatus, heartbeat.QualityStatus, heartbeat.CameraStatus, heartbeat.AppVersion, heartbeat.ModelVersion, heartbeat.PendingMessages, heartbeat.StorageAvailable)
	if err != nil {
		return domain.BatchAck{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.BatchAck{}, err
	}
	return acceptedAck(heartbeat.MessageID), nil
}

func (repository *Repository) ListFleet(ctx context.Context) ([]application.FleetVehicle, error) {
	rows, err := repository.pool.Query(ctx, `SELECT v.id,v.plate,v.internal_code,d.id,d.label,h.occurred_at,h.operation_status,t.latitude,t.longitude,t.captured_at,t.battery_percent FROM vehicles v LEFT JOIN device_vehicle_assignments a ON a.vehicle_id=v.id AND a.ends_at IS NULL LEFT JOIN devices d ON d.id=a.device_id LEFT JOIN LATERAL (SELECT occurred_at,operation_status FROM device_heartbeats WHERE device_id=d.id ORDER BY occurred_at DESC LIMIT 1) h ON TRUE LEFT JOIN LATERAL (SELECT latitude,longitude,captured_at,battery_percent FROM device_telemetry WHERE enterprise_device_id=d.id ORDER BY captured_at DESC LIMIT 1) t ON TRUE ORDER BY v.plate`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]application.FleetVehicle, 0)
	for rows.Next() {
		var item application.FleetVehicle
		if err := rows.Scan(&item.VehicleID, &item.Plate, &item.InternalCode, &item.DeviceID, &item.DeviceLabel, &item.LastContactAt, &item.OperationStatus, &item.Latitude, &item.Longitude, &item.LocationAt, &item.BatteryPercent); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repository *Repository) PurgeTelemetryBefore(ctx context.Context, cutoff time.Time) error {
	_, err := repository.pool.Exec(ctx, `DELETE FROM device_telemetry WHERE captured_at < $1`, cutoff)
	return err
}

func claimMessage(ctx context.Context, tx pgx.Tx, deviceID, messageID string) (bool, error) {
	tag, err := tx.Exec(ctx, `INSERT INTO received_device_messages(device_id,message_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, deviceID, messageID)
	return tag.RowsAffected() == 0, err
}
func acceptedAck(messageID string) domain.BatchAck {
	return domain.BatchAck{MessageID: messageID, Status: "accepted"}
}
func duplicateAck(messageID string) domain.BatchAck {
	return domain.BatchAck{MessageID: messageID, Status: "duplicate"}
}
