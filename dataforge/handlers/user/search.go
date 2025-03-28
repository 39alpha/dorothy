package user

import (
	"fmt"
	"strconv"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Search struct {
	handlers.ErrorHandler

	pattern string
	limit   int
	users   []models.User
}

func (page *Search) Pre(c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrForbidden
	}

	queries := c.Queries()

	pattern, ok := queries["q"]
	if !ok || pattern == "" {
		return fmt.Errorf("%w: invalid pattern (%q) provided", fiber.ErrBadRequest, pattern)
	}
	page.pattern = pattern

	limit, ok := queries["limit"]
	if !ok || limit == "" {
		page.limit = -1
	} else {
		var err error
		page.limit, err = strconv.Atoi(limit)
		if err != nil || page.limit < 1 {
			return fmt.Errorf("%w: invalid limit (%q) provided", fiber.ErrBadRequest, limit)
		}
	}

	return nil
}

func (page *Search) Run() error {
	var err error
	page.users, err = page.DB().SearchUsers(page.pattern, page.limit)
	return handlers.GormToFiber(err)
}

func (page *Search) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"users": page.users,
	})
}
