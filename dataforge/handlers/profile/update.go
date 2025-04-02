package profile

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Update struct {
	handlers.App

	user *models.User
}

func (page *Update) Run(c *fiber.Ctx) error {
	db := handlers.GetDB(c)

	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	var update models.UpdateUser
	if err := c.BodyParser(&update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	if authUser.ID != update.ID {
		return fiber.ErrForbidden
	}

	if err := db.UpdateUser(update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	var err error
	page.user, err = db.GetUserById(update.ID)
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
