package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/drivervision/api/internal/domain"
	"github.com/google/uuid"
)

type EnterpriseRepository interface {
	CreateCompany(context.Context, domain.Company) (domain.Company, error)
	CreateVehicle(context.Context, domain.Vehicle) (domain.Vehicle, error)
	CreateDevice(context.Context, domain.Device) (domain.Device, error)
	CreateEnrollmentCode(context.Context, domain.EnrollmentCode) error
	RedeemEnrollment(context.Context, string, string, time.Time) (domain.EnrollmentResult, error)
	RedeemVehiclePairing(context.Context, string, string, time.Time) (domain.EnrollmentResult, error)
	ListAdminVehicles(context.Context) ([]domain.Vehicle, error)
	SetVehiclePairingCode(context.Context, string, string) error
	AuthenticateDevice(context.Context, string) (domain.DevicePrincipal, error)
	RevokeDevice(context.Context, string, time.Time) error
	RecordEvent(context.Context, domain.DevicePrincipal, domain.DeviceEvent) (domain.BatchAck, error)
	RecordSession(context.Context, domain.DevicePrincipal, domain.MonitoringSession) (domain.BatchAck, error)
	RecordCoverage(context.Context, domain.DevicePrincipal, domain.CoverageInterval) (domain.BatchAck, error)
	RecordTelemetry(context.Context, domain.DevicePrincipal, domain.TelemetryMessage) (domain.BatchAck, error)
	RecordHeartbeat(context.Context, domain.DevicePrincipal, domain.DeviceHeartbeat) (domain.BatchAck, error)
	ListFleet(context.Context) ([]FleetVehicle, error)
	ListEvents(context.Context, FleetEventFilter) ([]FleetEvent, int, error)
}

var (
	ErrEnrollmentInvalid  = errors.New("enrollment code is invalid, expired or already used")
	ErrDeviceUnauthorized = errors.New("device credential is invalid or revoked")
)

type FleetVehicle struct {
	VehicleID       string     `json:"vehicleId"`
	Plate           string     `json:"plate"`
	InternalCode    string     `json:"internalCode"`
	DeviceID        string     `json:"deviceId"`
	DeviceLabel     string     `json:"deviceLabel"`
	LastContactAt   *time.Time `json:"lastContactAt,omitempty"`
	OperationStatus string     `json:"operationStatus,omitempty"`
	Latitude        *float64   `json:"latitude,omitempty"`
	Longitude       *float64   `json:"longitude,omitempty"`
	LocationAt      *time.Time `json:"locationAt,omitempty"`
	BatteryPercent  *int       `json:"batteryPercent,omitempty"`
}

// FleetEventFilter narrows the enterprise detection events exposed to the dashboard.
type FleetEventFilter struct {
	Type      string
	Severity  string
	VehicleID string
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}

// FleetEvent is a read projection of an enterprise detection event enriched with
// the vehicle plate and terminal label so the dashboard can render it directly.
type FleetEvent struct {
	EventID              string     `json:"eventId"`
	SessionID            string     `json:"sessionId"`
	Type                 string     `json:"type"`
	Lifecycle            string     `json:"lifecycle"`
	Severity             string     `json:"severity"`
	StartedAt            time.Time  `json:"startedAt"`
	EndedAt              *time.Time `json:"endedAt,omitempty"`
	DurationMilliseconds *int64     `json:"durationMs,omitempty"`
	Confidence           *float32   `json:"confidence,omitempty"`
	ObservationQuality   *float32   `json:"observationQuality,omitempty"`
	DetectorVersion      string     `json:"detectorVersion"`
	ModelVersion         string     `json:"modelVersion"`
	CompanyID            string     `json:"companyId"`
	VehicleID            string     `json:"vehicleId"`
	VehiclePlate         string     `json:"vehiclePlate"`
	DeviceID             string     `json:"deviceId"`
	DeviceLabel          string     `json:"deviceLabel"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

type EnterpriseService struct {
	repository EnterpriseRepository
	now        func() time.Time
}

func NewEnterpriseService(repository EnterpriseRepository) *EnterpriseService {
	return &EnterpriseService{repository: repository, now: time.Now}
}

func (service *EnterpriseService) CreateCompany(ctx context.Context, name, timezone string) (domain.Company, error) {
	if name == "" {
		return domain.Company{}, errors.New("name is required")
	}
	if timezone == "" {
		timezone = "America/Bogota"
	}
	return service.repository.CreateCompany(ctx, domain.Company{ID: uuid.NewString(), Name: name, Timezone: timezone, CreatedAt: service.now().UTC()})
}

func (service *EnterpriseService) CreateVehicle(ctx context.Context, companyID, plate, internalCode string) (domain.Vehicle, error) {
	if companyID == "" || plate == "" {
		return domain.Vehicle{}, errors.New("companyId and plate are required")
	}
	pairingCode, err := randomSecret()
	if err != nil {
		return domain.Vehicle{}, err
	}
	vehicle, err := service.repository.CreateVehicle(ctx, domain.Vehicle{
		ID: uuid.NewString(), CompanyID: companyID, Plate: plate, InternalCode: internalCode,
		Status: "ACTIVE", PairingCode: pairingCode, CreatedAt: service.now().UTC(),
	})
	if err != nil {
		return domain.Vehicle{}, err
	}
	vehicle.QRPayload = domain.VehicleQRPayload(pairingCode)
	return vehicle, nil
}

func (service *EnterpriseService) ListVehicles(ctx context.Context) ([]domain.Vehicle, error) {
	items, err := service.repository.ListAdminVehicles(ctx)
	if err != nil {
		return nil, err
	}
	for index := range items {
		if items[index].PairingCode != "" {
			items[index].QRPayload = domain.VehicleQRPayload(items[index].PairingCode)
			continue
		}
		pairingCode, err := randomSecret()
		if err != nil {
			return nil, err
		}
		if err := service.repository.SetVehiclePairingCode(ctx, items[index].ID, pairingCode); err != nil {
			// Another request may have filled it; re-list single vehicle state via full list refresh below.
			refreshed, listErr := service.repository.ListAdminVehicles(ctx)
			if listErr != nil {
				return nil, err
			}
			return service.withQRPayloads(refreshed), nil
		}
		items[index].PairingCode = pairingCode
		items[index].QRPayload = domain.VehicleQRPayload(pairingCode)
	}
	return items, nil
}

func (service *EnterpriseService) withQRPayloads(items []domain.Vehicle) []domain.Vehicle {
	for index := range items {
		items[index].QRPayload = domain.VehicleQRPayload(items[index].PairingCode)
	}
	return items
}

func (service *EnterpriseService) CreateDevice(ctx context.Context, companyID, label, hardware string) (domain.Device, error) {
	if companyID == "" {
		return domain.Device{}, errors.New("companyId is required")
	}
	return service.repository.CreateDevice(ctx, domain.Device{ID: uuid.NewString(), CompanyID: companyID, Label: label, Hardware: hardware})
}

func (service *EnterpriseService) CreateEnrollment(ctx context.Context, companyID, vehicleID, deviceID string, expiresIn time.Duration) (domain.EnrollmentCode, error) {
	if companyID == "" || vehicleID == "" || deviceID == "" {
		return domain.EnrollmentCode{}, errors.New("companyId, vehicleId and deviceId are required")
	}
	if expiresIn <= 0 || expiresIn > 24*time.Hour {
		expiresIn = 15 * time.Minute
	}
	code, err := randomSecret()
	if err != nil {
		return domain.EnrollmentCode{}, err
	}
	// Device is bound by creating the active assignment before the one-time credential is redeemed.
	if err := service.repository.CreateEnrollmentCode(ctx, domain.EnrollmentCode{ID: uuid.NewString(), CompanyID: companyID, VehicleID: vehicleID, DeviceID: deviceID, Code: hashSecret(code), ExpiresAt: service.now().UTC().Add(expiresIn)}); err != nil {
		return domain.EnrollmentCode{}, err
	}
	return domain.EnrollmentCode{CompanyID: companyID, VehicleID: vehicleID, DeviceID: deviceID, Code: code, QRPayload: "DRIVER_VISION_ENROLL:" + code, ExpiresAt: service.now().UTC().Add(expiresIn)}, nil
}

func (service *EnterpriseService) Redeem(ctx context.Context, code string) (domain.EnrollmentResult, error) {
	if code == "" {
		return domain.EnrollmentResult{}, ErrEnrollmentInvalid
	}
	token, err := randomSecret()
	if err != nil {
		return domain.EnrollmentResult{}, err
	}
	now := service.now().UTC()
	credentialHash := hashSecret(token)

	// Prefer legacy one-time enrollments; fall back to durable vehicle pairing QR.
	result, err := service.repository.RedeemEnrollment(ctx, hashSecret(code), credentialHash, now)
	if errors.Is(err, ErrEnrollmentInvalid) {
		result, err = service.repository.RedeemVehiclePairing(ctx, code, credentialHash, now)
	}
	if err != nil {
		return domain.EnrollmentResult{}, err
	}
	result.DeviceToken = token
	result.TelemetryEverySeconds = 60
	return result, nil
}

func (service *EnterpriseService) Authenticate(ctx context.Context, token string) (domain.DevicePrincipal, error) {
	if token == "" {
		return domain.DevicePrincipal{}, ErrDeviceUnauthorized
	}
	return service.repository.AuthenticateDevice(ctx, hashSecret(token))
}

func (service *EnterpriseService) Revoke(ctx context.Context, deviceID string) error {
	return service.repository.RevokeDevice(ctx, deviceID, service.now().UTC())
}
func (service *EnterpriseService) Fleet(ctx context.Context) ([]FleetVehicle, error) {
	return service.repository.ListFleet(ctx)
}
func (service *EnterpriseService) Events(ctx context.Context, filter FleetEventFilter) ([]FleetEvent, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return service.repository.ListEvents(ctx, filter)
}
func (service *EnterpriseService) Event(ctx context.Context, principal domain.DevicePrincipal, value domain.DeviceEvent) (domain.BatchAck, error) {
	if err := value.Validate(); err != nil {
		return rejected(value.MessageID, err), nil
	}
	return service.repository.RecordEvent(ctx, principal, value)
}
func (service *EnterpriseService) Session(ctx context.Context, principal domain.DevicePrincipal, value domain.MonitoringSession) (domain.BatchAck, error) {
	if err := value.SyncMessage.Validate(); err != nil || value.SessionID == "" {
		return rejected(value.MessageID, errors.New("invalid session")), nil
	}
	return service.repository.RecordSession(ctx, principal, value)
}
func (service *EnterpriseService) Coverage(ctx context.Context, principal domain.DevicePrincipal, value domain.CoverageInterval) (domain.BatchAck, error) {
	if err := value.SyncMessage.Validate(); err != nil || value.SessionID == "" || value.StartedAt.IsZero() || value.EndedAt.IsZero() {
		return rejected(value.MessageID, errors.New("invalid coverage")), nil
	}
	return service.repository.RecordCoverage(ctx, principal, value)
}
func (service *EnterpriseService) Telemetry(ctx context.Context, principal domain.DevicePrincipal, value domain.TelemetryMessage) (domain.BatchAck, error) {
	if err := value.Validate(); err != nil {
		return rejected(value.MessageID, err), nil
	}
	return service.repository.RecordTelemetry(ctx, principal, value)
}
func (service *EnterpriseService) Heartbeat(ctx context.Context, principal domain.DevicePrincipal, value domain.DeviceHeartbeat) (domain.BatchAck, error) {
	if err := value.SyncMessage.Validate(); err != nil || value.OccurredAt.IsZero() {
		return rejected(value.MessageID, errors.New("invalid heartbeat")), nil
	}
	return service.repository.RecordHeartbeat(ctx, principal, value)
}

func rejected(messageID string, err error) domain.BatchAck {
	return domain.BatchAck{MessageID: messageID, Status: "rejected", Code: err.Error()}
}
func hashSecret(value string) string {
	hash := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
func randomSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
