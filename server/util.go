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

func AddState(c *fiber.Ctx, key string, value interface{}) {
	state, ok := c.Locals("State").(fiber.Map)
	if !ok || state == nil {
		state = fiber.Map{
			key: value,
		}
	} else {
		state[key] = value
	}
	c.Locals("State", state)
	c.Locals(key, value)
}

func Redirect(c *fiber.Ctx, status int, path string, json any, text string) error {
	if c.Accepts("text/html") != "" {
		return c.Status(status).Redirect(path)
	} else if c.Accepts("application/json") != "" {
		return c.Status(status).JSON(json)
	} else if c.Accepts("text/plain") != "" {
		return c.Status(status).SendString(text)
	}

	return c.Status(status).Redirect(path)
}

func Goto(path string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Redirect(path)
	}
}

func GotoWithErrorMessage(path, err string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("Error", err)
		return c.Redirect(path)
	}
}

func Respond(c *fiber.Ctx, status int, json any, text string) error {
	if c.Accepts("application/json") != "" {
		return c.Status(status).JSON(json)
	} else if c.Accepts("text/plain") != "" {
		return c.Status(status).SendString(text)
	}

	return c.Status(status).JSON(json)
}
