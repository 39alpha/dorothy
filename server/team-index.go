package server

import (
	"errors"

	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
)

type GetTeam struct {
	authUser *models.User
	team     *models.Team

	canRead   bool
	canWrite  bool
	canManage bool
}

func (page *GetTeam) Preprocess(d *Server, c *fiber.Ctx) (err error) {
	if authUser, ok := c.Locals("AuthUser").(*models.User); ok {
		page.authUser = authUser
	}

	page.team, err = d.db.GetTeam(page.authUser, c.Params("team"))
	if err != nil {
		return GormToFiber(err)
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

func (page *GetTeam) HandleError(d *Server, c *fiber.Ctx, err error) error {
	var e *fiber.Error
	if errors.As(err, &e) && e == fiber.ErrUnauthorized {
		return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
	}

	return d.ErrorFallback(c, err)
}

func (page *GetTeam) RenderHtml(c *fiber.Ctx) error {
	return c.Render("views/team/index", Bind(c, fiber.Map{
		"AuthUser":  page.authUser,
		"CanRead":   page.canRead,
		"CanWrite":  page.canWrite,
		"CanManage": page.canManage,
		"Team":      page.team,
	}), "views/layouts/main")
}

func (page *GetTeam) RenderJson(c *fiber.Ctx) error {
	return c.JSON(Bind(c, fiber.Map{
		"AuthUser":  page.authUser,
		"CanRead":   page.canRead,
		"CanWrite":  page.canWrite,
		"CanManage": page.canManage,
		"Team":      page.team,
	}))
}
