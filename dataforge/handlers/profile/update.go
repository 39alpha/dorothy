package profile

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Update struct {
	handlers.App

	update models.UpdateUser
	user   *models.User
}

func (page *Update) Pre(c *fiber.Ctx) error {
	authUser, ok := c.Locals("AuthUser").(*models.User)
	if !ok {
		return fiber.ErrUnauthorized
	}

	if err := c.BodyParser(&page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	if authUser.ID != page.update.ID {
		return fiber.ErrForbidden
	}

	return nil
}

func (page *Update) Run() error {
	if err := page.DB().UpdateUser(page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	return nil
}

func (page *Update) Post(c *fiber.Ctx) error {
	var err error
	page.user, err = page.DB().GetUserById(page.update.ID)
	return handlers.GormToFiber(err)
}

func (page *Update) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverForm(&Form{
		App: page.App,
	}, c, err)
}

func (page *Update) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/profile")
}

func (page *Update) RenderJson(c *fiber.Ctx) error {
	return c.JSON(page.user)
}

func (page *Update) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
