package admin

import (
	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Dashboard struct {
	handlers.App

	authUser *models.User
}

func (page *Dashboard) Pre(c *fiber.Ctx) error {
	return RequireAdmin(c)
}

func (page *Dashboard) RenderHtml(c *fiber.Ctx) error {
	return c.Render("admin/dashboard", handlers.Bind(c), "layouts/main")
}

func RequireAdmin(c *fiber.Ctx) error {
	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	} else if !authUser.HasAdminRole() {
		return fiber.ErrForbidden
	}
	return nil
}
