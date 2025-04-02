package team

import (
	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Team struct {
	handlers.App

	authUser *models.User
	team     *models.Team

	canRead   bool
	canWrite  bool
	canManage bool
}

func (page *Team) Run(c *fiber.Ctx) (err error) {
	page.authUser = handlers.GetAuthUser(c)

	page.team, err = page.DB().GetTeam(page.authUser, c.Params("team"))
	if err != nil {
		return handlers.GormToFiber(err)
	}

	if page.authUser != nil {
		page.canRead = page.authUser.CanReadTeam(*page.team)
		page.canWrite = page.authUser.CanManageTeam(*page.team)
		page.canManage = page.authUser.CanManageTeam(*page.team)
	}

	if page.authUser == nil && page.team.IsPrivate {
		err = fiber.ErrUnauthorized
	} else if page.authUser != nil && !page.canRead {
		err = fiber.ErrForbidden
	}

	return
}

func (page *Team) RenderHtml(c *fiber.Ctx) error {
	return c.Render("team/index", handlers.Bind(c, fiber.Map{
		"CanRead":   page.canRead,
		"CanWrite":  page.canWrite,
		"CanManage": page.canManage,
		"Team":      page.team,
	}), "layouts/main")
}

func (page *Team) RenderJson(c *fiber.Ctx) error {
	return c.JSON(handlers.Bind(c, fiber.Map{
		"CanRead":   page.canRead,
		"CanWrite":  page.canWrite,
		"CanManage": page.canManage,
		"Team":      page.team,
	}))
}
