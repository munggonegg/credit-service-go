package router

import (
	"munggonegg/credit-service-go/pkg/handler"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {

	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Root route
	v1.Get("/", handler.GetRoot)

	// Token Used route
	v1.Post("/token_used", handler.RecordTokenUsed)
}
