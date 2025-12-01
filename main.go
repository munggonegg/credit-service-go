package main

import (
	"log"
	"munggonegg/credit-service-go/pkg/config"
	"munggonegg/credit-service-go/pkg/database"
	"munggonegg/credit-service-go/pkg/router"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Load config
	config.LoadConfig()

	// Connect to Database
	database.Connect()

	// Initialize Fiber app
	app := fiber.New()

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())

	// Setup Routes
	router.SetupRoutes(app)

	// Start server
	log.Fatal(app.Listen(":3000"))
}
