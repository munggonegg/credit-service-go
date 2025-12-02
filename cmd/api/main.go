package main

import (
	"log"
	"munggonegg/credit-service-go/internal/config"
	"munggonegg/credit-service-go/internal/adapter/repository/mongodb"
	"munggonegg/credit-service-go/internal/adapter/handler/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {

	config.LoadConfig()
	mongodb.Connect()

	app := fiber.New()

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())

	// Setup Routes
	http.SetupRoutes(app)

	// Start server
	log.Fatal(app.Listen(":3000"))
}
