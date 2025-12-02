package http

import (


	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {

	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Root route
	app.Get("/", GetRoot)

	// Token Used route
	v1.Post("/token_used", RecordTokenUsed)
}
