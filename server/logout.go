package server

import (
	"github.com/gofiber/fiber/v2"
)

type Logout struct{}

func (page *Logout) Postprocess(d *Server, c *fiber.Ctx) error {
	c.ClearCookie("jwt")
	return nil
}

func (page *Logout) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/")
}

func (page *Logout) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (page *Logout) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
