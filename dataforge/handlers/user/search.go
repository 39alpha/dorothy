package user

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Search struct {
	handlers.App

	users []models.User
}

func (page *Search) Run(c *fiber.Ctx) error {
	if err := handlers.RequireLogin(c); err != nil {
		return err
	}

	pattern := c.Query("q")
	if pattern == "" {
		return fmt.Errorf("%w: invalid pattern (%q) provided", fiber.ErrBadRequest, pattern)
	}

	limit := c.QueryInt("limit", 20)
	if limit < 1 {
		return fmt.Errorf("%w: invalid limit (%q) provided", fiber.ErrBadRequest, limit)
	}

	var err error
	page.users, err = page.DB().SearchUsers(pattern, limit)
	return handlers.GormToFiber(err)
}

func (page *Search) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"users": page.users,
	})
}
