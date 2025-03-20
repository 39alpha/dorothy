package server

import (
	"maps"

	"github.com/gofiber/fiber/v2"
)

func Bind(c *fiber.Ctx, local fiber.Map) fiber.Map {
	bind := fiber.Map{}

	state, ok := c.Locals("State").(fiber.Map)
	if !ok || state != nil {
		maps.Copy(bind, state)
	}
	maps.Copy(bind, local)

	return bind
}
