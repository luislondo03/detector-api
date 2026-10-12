package httpadapter

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/drivervision/api/internal/application"
	"github.com/drivervision/api/internal/domain"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service           *application.IncidentService
	telemetryService  *application.TelemetryService
	enterpriseService *application.EnterpriseService
	adminBootstrapKey string
}

func NewHandler(service *application.IncidentService, telemetryService *application.TelemetryService, enterpriseService *application.EnterpriseService, adminBootstrapKey string) *Handler {
	return &Handler{service: service, telemetryService: telemetryService, enterpriseService: enterpriseService, adminBootstrapKey: adminBootstrapKey}
}

func (handler *Handler) CreateTelemetry(c *fiber.Ctx) error {
	var telemetry domain.DeviceTelemetry
	if err := c.BodyParser(&telemetry); err != nil {
		return badRequest(c, "invalid JSON body")
	}
	saved, err := handler.telemetryService.Report(context.Background(), telemetry)
	if err != nil {
		return badRequest(c, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(saved)
}

func (handler *Handler) ListLatestTelemetry(c *fiber.Ctx) error {
	items, err := handler.telemetryService.Latest(context.Background(), c.QueryInt("limit", 50))
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": items})
}

func (handler *Handler) CreateIncident(c *fiber.Ctx) error {
	var incident domain.Incident
	if err := c.BodyParser(&incident); err != nil {
		return badRequest(c, "invalid JSON body")
	}
	saved, err := handler.service.Report(context.Background(), incident)
	if err != nil {
		return badRequest(c, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(saved)
}

func (handler *Handler) GetIncident(c *fiber.Ctx) error {
	incident, err := handler.service.Get(context.Background(), c.Params("incidentId"))
	if errors.Is(err, application.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "incident not found"})
	}
	if err != nil {
		return err
	}
	return c.JSON(incident)
}

func (handler *Handler) ListIncidents(c *fiber.Ctx) error {
	filter := application.IncidentFilter{Type: c.Query("type"), Limit: c.QueryInt("limit", 30), Offset: c.QueryInt("offset", 0)}
	var err error
	if filter.From, err = parseOptionalTime(c.Query("from")); err != nil {
		return badRequest(c, "from must be RFC3339")
	}
	if filter.To, err = parseOptionalTime(c.Query("to")); err != nil {
		return badRequest(c, "to must be RFC3339")
	}
	items, total, err := handler.service.List(context.Background(), filter)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": items, "total": total, "limit": filter.Limit, "offset": filter.Offset})
}

func (handler *Handler) UploadEvidence(c *fiber.Ctx) error {
	consent, err := strconv.ParseBool(c.FormValue("consent"))
	if err != nil || !consent {
		return badRequest(c, "consent=true is required to store a frame")
	}
	file, err := c.FormFile("file")
	if err != nil {
		return badRequest(c, "file is required")
	}
	if file.Size <= 0 || file.Size > 5<<20 {
		return badRequest(c, "file must be between 1 byte and 5 MiB")
	}
	content, err := file.Open()
	if err != nil {
		return err
	}
	defer content.Close()
	prefix := make([]byte, 512)
	read, readErr := io.ReadFull(content, prefix)
	if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) && !errors.Is(readErr, io.EOF) {
		return badRequest(c, "could not read evidence file")
	}
	mediaType := http.DetectContentType(prefix[:read])
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		return badRequest(c, "evidence must be a JPEG or PNG image")
	}
	evidence, err := handler.service.AttachEvidence(
		context.Background(), c.Params("incidentId"), consent, mediaType,
		io.MultiReader(bytes.NewReader(prefix[:read]), content),
	)
	if errors.Is(err, application.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "incident not found"})
	}
	if err != nil {
		return badRequest(c, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(evidence)
}

func parseOptionalTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func badRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": message})
}
