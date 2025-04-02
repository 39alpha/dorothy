package profile

import (
	"fmt"
	"strings"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type ChangePassword struct {
	handlers.App
}

func (page *ChangePassword) Run(c *fiber.Ctx) error {
	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	var update models.ChangePassword
	if err := c.BodyParser(&update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	update.Password = strings.TrimSpace(update.Password)
	if len(update.Password) < 8 {
		return fmt.Errorf("%w: password must be at least 8 characters long", fiber.ErrBadRequest)
	}

	if authUser.ID != update.ID {
		return fiber.ErrForbidden
	}

	if err := page.DB().UserChangePassword(update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	return nil
}

func (page *ChangePassword) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverForm(&Form{
		App: page.App,
	}, c, err)
}

func (page *ChangePassword) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (page *ChangePassword) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
