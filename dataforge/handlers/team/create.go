package team

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type CreateForm struct {
	handlers.App
}

func (form *CreateForm) Run(c *fiber.Ctx) error {
	return handlers.RequireLogin(c)
}

func (form *CreateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("team/create", handlers.Bind(c), "layouts/main")
}

type Create struct {
	handlers.App

	team *models.Team
}

func (page *Create) Run(c *fiber.Ctx) error {
	db := handlers.GetDB(c)

	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	var newTeam models.NewTeam
	if err := c.BodyParser(&newTeam); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	name := models.Slugify(newTeam.Name)
	if handlers.IsDisallowedName(name) {
		return fmt.Errorf("%w: that team name is already taken", fiber.ErrConflict)
	}

	var err error
	newTeam.Name, err = db.CreateTeam(newTeam, authUser)
	if err != nil {
		return handlers.GormToFiber(err)
	}

	page.team, err = db.GetTeam(authUser, newTeam.Name)
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
