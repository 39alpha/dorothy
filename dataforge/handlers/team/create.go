package team

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type CreateForm struct {
	handlers.App
	Err error

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

func (form *CreateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("team/create", handlers.Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Error":    form.Err,
	}), "layouts/main")
}

type Create struct {
	handlers.App

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

	name := models.Slugify(page.newTeam.Name)
	if handlers.IsDisallowedName(name) {
		return fmt.Errorf("%w: that team name is already taken", fiber.ErrConflict)
	}

	return nil
}

func (page *Create) Run() error {
	var err error
	page.newTeam.Name, err = page.DB().CreateTeam(page.newTeam, &page.authUser)
	if err != nil {
		return handlers.GormToFiber(err)
	}
	return nil
}

func (page *Create) Post(c *fiber.Ctx) error {
	var err error
	page.team, err = page.DB().GetTeam(&page.authUser, page.newTeam.Name)
	return handlers.GormToFiber(err)
}

func (page *Create) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverForm(&CreateForm{
		App: page.App,
	}, c, err)
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
