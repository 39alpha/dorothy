package profile

import (
	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/gofiber/fiber/v2"
)

type Form struct {
	handlers.App
}

func (form *Form) Run(c *fiber.Ctx) error {
	return handlers.RequireLogin(c)
}

func (form *Form) RenderHtml(c *fiber.Ctx) error {
	return c.Render("profile", handlers.Bind(c), "layouts/main")
}
