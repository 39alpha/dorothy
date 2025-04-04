package team

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type CreateForm struct{}

func (form *CreateForm) Run(c *fiber.Ctx) error {
	handlers.GetLogger(c).Trace().Handler("team.CreateForm").Send()
	return handlers.RequireLogin(c)
}

func (form *CreateForm) RenderHtml(c *fiber.Ctx) error {
	handlers.GetLogger(c).Trace().Renderer("team.CreateForm", "RenderHtml").Send()
	return c.Render("team/create", handlers.Bind(c), "layouts/main")
}

type Create struct {
	team *models.Team
}

func (page *Create) Run(c *fiber.Ctx) error {
	logger := handlers.GetLogger(c).Trace().Handler("team.Create").Send()

	db := handlers.GetDB(c)

	logger.Trace().Msg("Require Authentication")
	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		logger.Debug().Msg("Unauthenticated")
		return fiber.ErrUnauthorized
	}

	logger.Trace().Msg("Parse Request Body")
	var newTeam models.NewTeam
	if err := c.BodyParser(&newTeam); err != nil {
		logger.Debug().Str("body", string(c.BodyRaw())).Msg("Bad Request")
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	logger.Trace().Msg("Is Valid Name")
	name := models.Slugify(newTeam.Name)
	if handlers.IsDisallowedName(name) {
		logger.Debug().Str("name", name).Msg("Disallowed Name")
		return fmt.Errorf("%w: that team name is already taken", fiber.ErrConflict)
	}

	logger.Trace().Msg("Create Team")
	var err error
	newTeam.Name, err = db.CreateTeam(newTeam, authUser)
	if err != nil {
		logger.Error(err).Msg("Create Team Failed")
		return handlers.GormToFiber(err)
	}

	logger.Trace().Msg("Get Team")
	page.team, err = db.GetTeam(authUser, newTeam.Name)
	if err != nil {
		logger.Error(err).Str("team", newTeam.Name).Msg("Get Team Failed")
		return handlers.GormToFiber(err)
	}
	return nil
}

func (page *Create) Recover(c *fiber.Ctx, err error) error {
	handlers.GetLogger(c).Trace().Recover("team.Create").Send()
	return handlers.RecoverForm(&CreateForm{}, c, err)
}

func (page *Create) RenderHtml(c *fiber.Ctx) error {
	handlers.GetLogger(c).Trace().Renderer("team.Create", "RenderHtml").Send()
	return c.Redirect("/" + page.team.Name)
}

func (page *Create) RenderJson(c *fiber.Ctx) error {
	handlers.GetLogger(c).Trace().Renderer("team.Create", "RenderJson").Send()
	return c.JSON(page.team)
}

func (*Create) RenderText(c *fiber.Ctx) error {
	handlers.GetLogger(c).Trace().Renderer("team.Create", "RenderText").Send()
	return c.SendString("success")
}
