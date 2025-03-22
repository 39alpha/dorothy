package team

import (
	"errors"
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CreateForm struct {
	handlers.ErrorHandler

	authUser models.User
}

func (form *CreateForm) Pre(c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}
	form.authUser = *authUser

	form.Err, _ = c.Locals("Error").(error)
	return nil
}

func (form *CreateForm) Recover(c *fiber.Ctx, err error) error {
	var e *fiber.Error

	if errors.As(err, &e) && e == fiber.ErrUnauthorized && handlers.Redirectable(c) {
		return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
	}

	return form.ErrorFallback(c, err)
}

func (form *CreateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("team/create", handlers.Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Error":    form.Err,
	}), "layouts/main")
}

type Create struct {
	handlers.ErrorHandler

	authUser models.User
	newTeam  models.NewTeam
	team     *models.Team
}

func (page *Create) Pre(c *fiber.Ctx) error {
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

func (page *Create) Run() error {
	team := &models.Team{
		Name:      models.Slugify(page.newTeam.Name),
		Contact:   page.newTeam.Contact,
		IsPrivate: page.newTeam.IsPrivate,
	}
	if page.newTeam.Description != nil {
		team.Description = *page.newTeam.Description
	}

	return page.DB().Transaction(func(tx *gorm.DB) error {
		if err := page.DB().Save(team).Error; err != nil {
			return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
		}

		err := page.DB().Save(&models.UserTeamPrivilege{
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

func (page *Create) Post(c *fiber.Ctx) error {
	var err error
	page.team, err = page.DB().GetTeam(&page.authUser, page.team.Name)
	return handlers.GormToFiber(err)
}

func (page *Create) Recover(c *fiber.Ctx, err error) error {
	return page.HandleFormError(&CreateForm{}, c, err)
}

func (page *Create) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/" + page.team.Name)
}

func (page *Create) RenderJson(c *fiber.Ctx) error {
	return c.JSON(page.team)
}

func (*Create) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
