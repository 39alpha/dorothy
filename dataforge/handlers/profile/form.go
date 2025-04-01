package profile

import (
	"errors"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/gofiber/fiber/v2"
)

type Form struct {
	handlers.App
}

func (form *Form) Pre(c *fiber.Ctx) error {
	return handlers.RequireLogin(c)
}

func (form *Form) Recover(c *fiber.Ctx, err error) error {
	var e *fiber.Error
	if errors.As(err, &e) && e == fiber.ErrUnauthorized && handlers.Redirectable(c) {
		return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
	}
	return err
}

func (form *Form) RenderHtml(c *fiber.Ctx) error {
	return c.Render("profile", handlers.Bind(c), "layouts/main")
}
