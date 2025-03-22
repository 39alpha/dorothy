package team

import (
	"errors"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Team struct {
	handlers.ErrorHandler

	authUser *models.User
	team     *models.Team

	canRead   bool
	canWrite  bool
	canManage bool
}

func (page *Team) Pre(c *fiber.Ctx) (err error) {
	if authUser, ok := c.Locals("AuthUser").(*models.User); ok {
		page.authUser = authUser
	}

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

func (page *Team) Recover(c *fiber.Ctx, err error) error {
	var e *fiber.Error
	if errors.As(err, &e) && e == fiber.ErrUnauthorized && handlers.Redirectable(c) {
		return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
	}

	return page.ErrorFallback(c, err)
}

func (page *Team) RenderHtml(c *fiber.Ctx) error {
	return c.Render("team/index", handlers.Bind(c, fiber.Map{
		"AuthUser":  page.authUser,
		"CanRead":   page.canRead,
		"CanWrite":  page.canWrite,
		"CanManage": page.canManage,
		"Team":      page.team,
	}), "layouts/main")
}

func (page *Team) RenderJson(c *fiber.Ctx) error {
	return c.JSON(handlers.Bind(c, fiber.Map{
		"AuthUser":  page.authUser,
		"CanRead":   page.canRead,
		"CanWrite":  page.canWrite,
		"CanManage": page.canManage,
		"Team":      page.team,
	}))
}
