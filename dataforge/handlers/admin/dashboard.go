package admin

import (
	"errors"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Dashboard struct {
	handlers.ErrorHandler

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

func (page *Dashboard) Recover(c *fiber.Ctx, err error) error {
	var e *fiber.Error

	if errors.As(err, &e) && e == fiber.ErrUnauthorized && handlers.Redirectable(c) {
		return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
	}

	return page.ErrorFallback(c, err)
}

func (page *Dashboard) RenderHtml(c *fiber.Ctx) error {
	return c.Render("admin/dashboard", handlers.Bind(c, fiber.Map{
		"AuthUser": page.authUser,
	}), "layouts/main")
}
