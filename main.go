package main

import (
	"log"
	"munggonegg/credit-service-go/pkg/router"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {

	app := fiber.New()

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())

	// Setup Routes
	router.SetupRoutes(app)

	// Start server
	log.Fatal(app.Listen(":3000"))
}
