package team

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type UpdateForm struct {
	team  models.Team
	users []models.User
}

func (form *UpdateForm) Run(c *fiber.Ctx) error {
	db := handlers.GetDB(c)

	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	team, err := db.GetTeam(authUser, c.Params("team"))
	if err != nil {
		return handlers.GormToFiber(err)
	}

	if !authUser.CanManageTeam(*team) {
		return fiber.ErrForbidden
	}

	form.team = *team
	form.users, _ = db.GetUsersWithTeamAccess(*team)

	return nil
}

func (form *UpdateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("team/settings", handlers.Bind(c, fiber.Map{
		"Team":  form.team,
		"Users": form.users,
	}), "layouts/main")
}

type Update struct {
	team *models.Team
}

func (page *Update) Run(c *fiber.Ctx) error {
	db := handlers.GetDB(c)

	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	team, err := db.GetTeam(authUser, c.Params("team"))
	if err != nil {
		return handlers.GormToFiber(err)
	}

	if !authUser.CanManageTeam(*team) {
		return fiber.ErrForbidden
	}

	var update models.UpdateTeam
	if err := c.BodyParser(&update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	name := models.Slugify(update.Name)
	if handlers.IsDisallowedName(name) {
		return fmt.Errorf("%w: that dataset name is already taken", fiber.ErrConflict)
	}

	_, err = db.UpdateTeam(update)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	page.team, err = db.GetTeamById(authUser, team.ID)
	return handlers.GormToFiber(err)
}

func (page *Update) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverForm(&UpdateForm{}, c, err)
}

func (page *Update) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect(page.team.Path())
}

func (page *Update) RenderJson(c *fiber.Ctx) error {
	return c.JSON(page.team)
}

func (page *Update) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
