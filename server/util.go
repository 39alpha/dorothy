package server

import (
	"maps"

	"github.com/gofiber/fiber/v2"
)

func Bind(c *fiber.Ctx, local ...fiber.Map) fiber.Map {
	bind := fiber.Map{}

	state, ok := c.Locals("State").(fiber.Map)
	if !ok || state != nil {
		maps.Copy(bind, state)
	}
	for _, l := range local {
		maps.Copy(bind, l)
	}

	return bind
}

func AcceptsHtml(c *fiber.Ctx) bool {
	return c.Accepts("text/html") != ""
}

func AcceptsJson(c *fiber.Ctx) bool {
	return c.Accepts("application/json") != ""
}

func AcceptsText(c *fiber.Ctx) bool {
	return c.Accepts("text/plain") != ""
}

func Redirectable(c *fiber.Ctx) bool {
	return AcceptsHtml(c) || (!AcceptsJson(c) && !AcceptsText(c))
}
