package admin

import (
	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/gofiber/fiber/v2"
)

func RequireAdmin(c *fiber.Ctx) error {
	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	} else if !authUser.HasAdminRole() {
		return fiber.ErrForbidden
	}
	return nil
}

type Dashboard struct{}

func (page *Dashboard) Run(c *fiber.Ctx) error {
	return RequireAdmin(c)
}

func (page *Dashboard) RenderHtml(c *fiber.Ctx) error {
	return c.Render("admin/dashboard", handlers.Bind(c), "layouts/main")
}
