package team

import (
	"errors"
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type UpdateForm struct {
	handlers.ErrorHandler

	authUser models.User
	team     models.Team
}

func (form *UpdateForm) Pre(c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}
	form.authUser = *authUser

	team, err := form.DB().GetTeam(authUser, c.Params("team"))
	if err != nil {
		return handlers.GormToFiber(err)
	}
	form.team = *team

	if !authUser.CanManageTeam(*team) {
		return fiber.ErrForbidden
	}

	form.Err, _ = c.Locals("Error").(error)

	return nil
}

func (form *UpdateForm) Recover(c *fiber.Ctx, err error) error {
	var e *fiber.Error

	if errors.As(err, &e) && e == fiber.ErrUnauthorized && handlers.Redirectable(c) {
		return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
	}

	return form.ErrorFallback(c, err)
}

func (form *UpdateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("team/settings", handlers.Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Team":     form.team,
		"Error":    form.Err,
	}), "layouts/main")
}

type Update struct {
	handlers.ErrorHandler

	authUser *models.User
	team     *models.Team
	update   models.UpdateTeam
}

func (page *Update) Pre(c *fiber.Ctx) error {
	authUser, ok := c.Locals("AuthUser").(*models.User)
	if !ok {
		return fiber.ErrUnauthorized
	}

	if err := c.BodyParser(&page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	team, err := page.DB().GetTeam(authUser, c.Params("team"))
	if err != nil {
		return handlers.GormToFiber(err)
	}

	if !authUser.CanManageTeam(*team) {
		return fiber.ErrForbidden
	}

	page.team = team
	page.authUser = authUser

	return nil
}

func (page *Update) Run() error {
	if err := page.DB().UpdateTeam(page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	return nil
}

func (page *Update) Post(c *fiber.Ctx) error {
	var err error
	page.team, err = page.DB().GetTeam(page.authUser, page.update.Name)
	return handlers.GormToFiber(err)
}

func (page *Update) Recover(c *fiber.Ctx, err error) error {
	return page.HandleFormError(&UpdateForm{}, c, err)
}

func (page *Update) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/" + page.update.Name)
}

func (page *Update) RenderJson(c *fiber.Ctx) error {
	return c.JSON(page.team)
}

func (page *Update) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
