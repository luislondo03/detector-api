package httpadapter

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/drivervision/api/internal/application"
	"github.com/drivervision/api/internal/domain"
	"github.com/gofiber/fiber/v2"
)

type batchRequest[T any] struct {
	Messages []T `json:"messages"`
}

func (handler *Handler) RequireAdmin(c *fiber.Ctx) error {
	// Pilot / pruebas: administración abierta sin clave.
	// Cuando quieras asegurar el entorno, vuelve a exigir ADMIN_BOOTSTRAP_KEY aquí.
	_ = handler.adminBootstrapKey
	return c.Next()
}

func (handler *Handler) RequireDevice(c *fiber.Ctx) error {
	token := strings.TrimSpace(strings.TrimPrefix(c.Get("Authorization"), "Bearer"))
	principal, err := handler.enterpriseService.Authenticate(context.Background(), token)
	if errors.Is(err, application.ErrDeviceUnauthorized) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "device authorization required"})
	}
	if err != nil {
		return err
	}
	c.Locals("devicePrincipal", principal)
	return c.Next()
}

func (handler *Handler) CreateCompany(c *fiber.Ctx) error {
	var request struct {
		Name     string `json:"name"`
		Timezone string `json:"timezone"`
	}
	if err := c.BodyParser(&request); err != nil {
		return badRequest(c, "invalid JSON body")
	}
	value, err := handler.enterpriseService.CreateCompany(context.Background(), request.Name, request.Timezone)
	if err != nil {
		return badRequest(c, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(value)
}
func (handler *Handler) CreateVehicle(c *fiber.Ctx) error {
	var request struct {
		CompanyID    string `json:"companyId"`
		Plate        string `json:"plate"`
		InternalCode string `json:"internalCode"`
	}
	if err := c.BodyParser(&request); err != nil {
		return badRequest(c, "invalid JSON body")
	}
	value, err := handler.enterpriseService.CreateVehicle(context.Background(), request.CompanyID, request.Plate, request.InternalCode)
	if err != nil {
		return badRequest(c, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(value)
}

func (handler *Handler) ListAdminVehicles(c *fiber.Ctx) error {
	items, err := handler.enterpriseService.ListVehicles(context.Background())
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": items})
}
func (handler *Handler) CreateDevice(c *fiber.Ctx) error {
	var request struct {
		CompanyID string `json:"companyId"`
		Label     string `json:"label"`
		Hardware  string `json:"hardware"`
	}
	if err := c.BodyParser(&request); err != nil {
		return badRequest(c, "invalid JSON body")
	}
	value, err := handler.enterpriseService.CreateDevice(context.Background(), request.CompanyID, request.Label, request.Hardware)
	if err != nil {
		return badRequest(c, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(value)
}
func (handler *Handler) CreateEnrollment(c *fiber.Ctx) error {
	var request struct {
		CompanyID        string `json:"companyId"`
		VehicleID        string `json:"vehicleId"`
		DeviceID         string `json:"deviceId"`
		ExpiresInMinutes int    `json:"expiresInMinutes"`
	}
	if err := c.BodyParser(&request); err != nil {
		return badRequest(c, "invalid JSON body")
	}
	value, err := handler.enterpriseService.CreateEnrollment(context.Background(), request.CompanyID, request.VehicleID, request.DeviceID, time.Duration(request.ExpiresInMinutes)*time.Minute)
	if err != nil {
		return badRequest(c, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(value)
}
func (handler *Handler) RevokeDevice(c *fiber.Ctx) error {
	if err := handler.enterpriseService.Revoke(context.Background(), c.Params("deviceId")); err != nil {
		return badRequest(c, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (handler *Handler) RedeemEnrollment(c *fiber.Ctx) error {
	var request struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&request); err != nil {
		return badRequest(c, "invalid JSON body")
	}
	value, err := handler.enterpriseService.Redeem(context.Background(), request.Code)
	if errors.Is(err, application.ErrEnrollmentInvalid) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "enrollment code is invalid, expired or already used"})
	}
	if err != nil {
		return err
	}
	return c.JSON(value)
}
func (handler *Handler) ListFleetVehicles(c *fiber.Ctx) error {
	items, err := handler.enterpriseService.Fleet(context.Background())
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": items})
}

func (handler *Handler) ListFleetEvents(c *fiber.Ctx) error {
	filter := application.FleetEventFilter{Type: c.Query("type"), Severity: c.Query("severity"), VehicleID: c.Query("vehicleId"), Limit: c.QueryInt("limit", 50), Offset: c.QueryInt("offset", 0)}
	var err error
	if filter.From, err = parseOptionalTime(c.Query("from")); err != nil {
		return badRequest(c, "from must be RFC3339")
	}
	if filter.To, err = parseOptionalTime(c.Query("to")); err != nil {
		return badRequest(c, "to must be RFC3339")
	}
	items, total, err := handler.enterpriseService.Events(context.Background(), filter)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": items, "total": total, "limit": filter.Limit, "offset": filter.Offset})
}

func devicePrincipal(c *fiber.Ctx) domain.DevicePrincipal {
	return c.Locals("devicePrincipal").(domain.DevicePrincipal)
}
func (handler *Handler) BatchEvents(c *fiber.Ctx) error {
	var request batchRequest[domain.DeviceEvent]
	if err := c.BodyParser(&request); err != nil {
		return badRequest(c, "invalid JSON body")
	}
	acks := make([]domain.BatchAck, 0, len(request.Messages))
	for _, value := range request.Messages {
		ack, err := handler.enterpriseService.Event(context.Background(), devicePrincipal(c), value)
		if err != nil {
			return err
		}
		acks = append(acks, ack)
	}
	return c.JSON(fiber.Map{"acks": acks})
}
func (handler *Handler) BatchSessions(c *fiber.Ctx) error {
	var request batchRequest[domain.MonitoringSession]
	if err := c.BodyParser(&request); err != nil {
		return badRequest(c, "invalid JSON body")
	}
	acks := make([]domain.BatchAck, 0, len(request.Messages))
	for _, value := range request.Messages {
		ack, err := handler.enterpriseService.Session(context.Background(), devicePrincipal(c), value)
		if err != nil {
			return err
		}
		acks = append(acks, ack)
	}
	return c.JSON(fiber.Map{"acks": acks})
}
func (handler *Handler) BatchCoverage(c *fiber.Ctx) error {
	var request batchRequest[domain.CoverageInterval]
	if err := c.BodyParser(&request); err != nil {
		return badRequest(c, "invalid JSON body")
	}
	acks := make([]domain.BatchAck, 0, len(request.Messages))
	for _, value := range request.Messages {
		ack, err := handler.enterpriseService.Coverage(context.Background(), devicePrincipal(c), value)
		if err != nil {
			return err
		}
		acks = append(acks, ack)
	}
	return c.JSON(fiber.Map{"acks": acks})
}
func (handler *Handler) BatchTelemetry(c *fiber.Ctx) error {
	var request batchRequest[domain.TelemetryMessage]
	if err := c.BodyParser(&request); err != nil {
		return badRequest(c, "invalid JSON body")
	}
	acks := make([]domain.BatchAck, 0, len(request.Messages))
	for _, value := range request.Messages {
		ack, err := handler.enterpriseService.Telemetry(context.Background(), devicePrincipal(c), value)
		if err != nil {
			return err
		}
		acks = append(acks, ack)
	}
	return c.JSON(fiber.Map{"acks": acks})
}
func (handler *Handler) Heartbeat(c *fiber.Ctx) error {
	var request domain.DeviceHeartbeat
	if err := c.BodyParser(&request); err != nil {
		return badRequest(c, "invalid JSON body")
	}
	ack, err := handler.enterpriseService.Heartbeat(context.Background(), devicePrincipal(c), request)
	if err != nil {
		return err
	}
	return c.JSON(ack)
}
