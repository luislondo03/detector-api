package httpadapter

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func NewApp(handler *Handler, allowedOrigin string, evidenceDirectory string) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "driver-vision-api",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if fiberError, ok := err.(*fiber.Error); ok {
				return c.Status(fiberError.Code).JSON(fiber.Map{"error": fiberError.Message})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		},
	})
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{AllowOrigins: allowedOrigin, AllowMethods: "GET,POST,OPTIONS", AllowHeaders: "Content-Type, Authorization, X-Admin-Bootstrap-Key"}))
	app.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Static("/evidence", evidenceDirectory, fiber.Static{MaxAge: 3600})

	api := app.Group("/api/v1")
	api.Get("/incidents", handler.ListIncidents)
	api.Post("/incidents", handler.CreateIncident)
	api.Get("/incidents/:incidentId", handler.GetIncident)
	api.Post("/incidents/:incidentId/evidence", handler.UploadEvidence)
	api.Post("/telemetry", handler.CreateTelemetry)
	api.Get("/fleet/latest", handler.ListLatestTelemetry)
	api.Get("/fleet/vehicles", handler.ListFleetVehicles)
	api.Get("/fleet/events", handler.ListFleetEvents)
	api.Post("/device-enrollment/redeem", handler.RedeemEnrollment)
	admin := api.Group("/admin", handler.RequireAdmin)
	admin.Post("/companies", handler.CreateCompany)
	admin.Get("/vehicles", handler.ListAdminVehicles)
	admin.Post("/vehicles", handler.CreateVehicle)
	admin.Post("/devices", handler.CreateDevice)
	admin.Post("/enrollments", handler.CreateEnrollment)
	admin.Post("/devices/:deviceId/revoke", handler.RevokeDevice)
	device := api.Group("/device", handler.RequireDevice)
	device.Post("/events/batch", handler.BatchEvents)
	device.Post("/sessions/batch", handler.BatchSessions)
	device.Post("/coverage/batch", handler.BatchCoverage)
	device.Post("/telemetry/batch", handler.BatchTelemetry)
	device.Post("/heartbeat", handler.Heartbeat)
	return app
}
