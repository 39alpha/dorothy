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
	page.authUser, _ = c.Locals("AuthUser").(*models.User)
	if page.authUser == nil {
		return fiber.ErrUnauthorized
	} else if !page.authUser.HasAdminRole() {
		return fiber.ErrForbidden
	}
	return nil
}

func (page *Dashboard) RenderHtml(c *fiber.Ctx) error {
	return c.Render("admin/dashboard", handlers.Bind(c, fiber.Map{
		"AuthUser": page.authUser,
	}), "layouts/main")
}
