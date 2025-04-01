package team

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type UpdateForm struct {
	handlers.App

	authUser models.User
	team     models.Team
	users    []models.User
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

	form.users, _ = form.DB().GetUsersWithTeamAccess(form.team)

	return nil
}

func (form *UpdateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("team/settings", handlers.Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Team":     form.team,
		"Users":    form.users,
	}), "layouts/main")
}

type Update struct {
	handlers.App

	authUser *models.User
	team     *models.Team
	update   models.UpdateTeam
}

func (page *Update) Pre(c *fiber.Ctx) error {
	authUser, ok := c.Locals("AuthUser").(*models.User)
	if !ok {
		return fiber.ErrUnauthorized
	}

	team, err := page.DB().GetTeam(authUser, c.Params("team"))
	if err != nil {
		return handlers.GormToFiber(err)
	}

	if !authUser.CanManageTeam(*team) {
		return fiber.ErrForbidden
	}

	if err := c.BodyParser(&page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	name := models.Slugify(page.update.Name)
	if handlers.IsDisallowedName(name) {
		return fmt.Errorf("%w: that dataset name is already taken", fiber.ErrConflict)
	}

	page.team = team
	page.authUser = authUser

	return nil
}

func (page *Update) Run() error {
	var err error
	page.update.Name, err = page.DB().UpdateTeam(page.update)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	return nil
}

func (page *Update) Post(c *fiber.Ctx) error {
	var err error
	page.team, err = page.DB().GetTeamById(page.authUser, page.team.ID)
	return handlers.GormToFiber(err)
}

func (page *Update) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverForm(&UpdateForm{
		App: page.App,
	}, c, err)
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
