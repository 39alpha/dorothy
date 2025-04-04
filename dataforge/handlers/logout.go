package handlers

import (
	"github.com/gofiber/fiber/v2"
)

type Logout struct{}

func (*Logout) Run(c *fiber.Ctx) error {
	GetLogger(c).Trace().Handler("Logout").Send()
	c.ClearCookie("jwt")
	return nil
}

func (*Logout) RenderHtml(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("Logout", "RenderHtml").Send()
	return c.Redirect("/")
}

func (*Logout) RenderJson(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("Logout", "RenderJson").Send()
	return c.JSON(fiber.Map{"message": "success"})
}

func (*Logout) RenderText(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("Logout", "RenderText").Send()
	return c.SendString("success")
}
