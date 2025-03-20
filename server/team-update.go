package server

import (
	"errors"
	"fmt"

	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
)

type UpdateTeamForm struct {
	authUser models.User
	team     models.Team
	err      error
}

func (form *UpdateTeamForm) Preprocess(d *Server, c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}
	form.authUser = *authUser

	team, err := d.db.GetTeam(authUser, c.Params("team"))
	if err != nil {
		return GormToFiber(err)
	}
	form.team = *team

	if !authUser.CanManageTeam(*team) {
		return fiber.ErrForbidden
	}

	form.err, _ = c.Locals("Error").(error)

	return nil
}

func (form *UpdateTeamForm) HandleError(d *Server, c *fiber.Ctx, err error) error {
	var e *fiber.Error

	if errors.As(err, &e) && e == fiber.ErrUnauthorized && Redirectable(c) {
		return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
	}

	return d.ErrorFallback(c, err)
}

func (form *UpdateTeamForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("views/team/settings", Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Team":     form.team,
		"Error":    form.err,
	}), "views/layouts/main")
}

type UpdateTeam struct {
	authUser *models.User
	team     *models.Team
	update   models.UpdateTeam
}

func (page *UpdateTeam) Preprocess(d *Server, c *fiber.Ctx) error {
	authUser, ok := c.Locals("AuthUser").(*models.User)
	if !ok {
		return fiber.ErrUnauthorized
	}

	if err := c.BodyParser(&page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	team, err := d.db.GetTeam(authUser, c.Params("team"))
	if err != nil {
		return GormToFiber(err)
	}

	if !authUser.CanManageTeam(*team) {
		return fiber.ErrForbidden
	}

	page.team = team
	page.authUser = authUser

	return nil
}

func (page *UpdateTeam) Run(d *Server) error {
	if err := d.db.UpdateTeam(page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	return nil
}

func (page *UpdateTeam) Postprocess(d *Server, c *fiber.Ctx) error {
	var err error
	page.team, err = d.db.GetTeam(page.authUser, page.update.Slug)
	return GormToFiber(err)
}

func (page *UpdateTeam) HandleError(d *Server, c *fiber.Ctx, err error) error {
	return d.HandleFormError(&UpdateTeamForm{}, c, err)
}

func (page *UpdateTeam) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/" + page.update.Slug)
}

func (page *UpdateTeam) RenderJson(c *fiber.Ctx) error {
	return c.JSON(page.team)
}

func (page *UpdateTeam) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
