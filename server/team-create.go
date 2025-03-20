package server

import (
	"errors"
	"fmt"

	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CreateTeamForm struct {
	authUser models.User
	err      error
}

func (form *CreateTeamForm) Preprocess(d *Server, c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}
	form.authUser = *authUser

	form.err, _ = c.Locals("Error").(error)
	return nil
}

func (form *CreateTeamForm) HandleError(d *Server, c *fiber.Ctx, err error) error {
	var e *fiber.Error

	if errors.As(err, &e) {
		if e == fiber.ErrUnauthorized {
			return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
		}
	}

	return d.ErrorFallback(c, err)
}

func (form *CreateTeamForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("views/team/create", Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Error":    form.err,
	}), "views/layouts/main")
}

type CreateTeam struct {
	authUser models.User
	newTeam  models.NewTeam
	team     *models.Team
}

func (page *CreateTeam) Preprocess(d *Server, c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}
	page.authUser = *authUser

	if err := c.BodyParser(&page.newTeam); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	return nil
}

func (page *CreateTeam) Run(d *Server) error {
	team := &models.Team{
		Slug:      page.newTeam.Slug,
		Name:      page.newTeam.Name,
		Contact:   page.newTeam.Contact,
		IsPrivate: page.newTeam.IsPrivate,
	}
	if page.newTeam.Description != nil {
		team.Description = *page.newTeam.Description
	}

	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := d.db.Save(team).Error; err != nil {
			return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
		}

		err := d.db.Save(&models.UserTeamPrivilege{
			User:          &page.authUser,
			Team:          team,
			PrivilegeCode: "admin",
		}).Error

		if err != nil {
			return fmt.Errorf(
				"%w: We could not create your team at this time. Try again later?",
				fiber.ErrInternalServerError,
			)
		}

		page.team = team

		return nil
	})
}

func (page *CreateTeam) Postprocess(d *Server, c *fiber.Ctx) error {
	var err error
	page.team, err = d.db.GetTeam(&page.authUser, page.team.Slug)
	return GormToFiber(err)
}

func (page *CreateTeam) HandleError(d *Server, c *fiber.Ctx, err error) error {
	return d.HandleFormError(&CreateTeamForm{}, c, err)
}

func (page *CreateTeam) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/" + page.team.Slug)
}

func (page *CreateTeam) RenderJson(c *fiber.Ctx) error {
	return c.JSON(page.team)
}

func (page *CreateTeam) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
