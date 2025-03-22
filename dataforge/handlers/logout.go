package handlers

import (
	"github.com/gofiber/fiber/v2"
)

type Logout struct {
	ErrorHandler
}

func (*Logout) Post(c *fiber.Ctx) error {
	c.ClearCookie("jwt")
	return nil
}

func (*Logout) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/")
}

func (*Logout) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (*Logout) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
