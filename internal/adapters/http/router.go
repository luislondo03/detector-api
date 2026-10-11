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
	app.Use(cors.New(cors.Config{AllowOrigins: allowedOrigin, AllowMethods: "GET,POST,OPTIONS"}))
	app.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Static("/evidence", evidenceDirectory, fiber.Static{MaxAge: 3600})

	api := app.Group("/api/v1")
	api.Get("/incidents", handler.ListIncidents)
	api.Post("/incidents", handler.CreateIncident)
	api.Get("/incidents/:incidentId", handler.GetIncident)
	api.Post("/incidents/:incidentId/evidence", handler.UploadEvidence)
	return app
}
