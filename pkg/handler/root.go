package handler

import (
	"github.com/gofiber/fiber/v2"
)

func GetRoot(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "This is Credit Service API.",
	})
}
