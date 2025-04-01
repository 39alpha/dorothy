package team

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Available struct {
	handlers.App

	payload struct {
		Name string
	}

	name         string
	isAvailable  bool
	isDisallowed bool
}

func (page *Available) Pre(c *fiber.Ctx) error {
	if err := handlers.RequireLogin(c); err != nil {
		return err
	}

	if err := c.BodyParser(&page.payload); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	page.name = models.Slugify(page.payload.Name)
	page.isDisallowed = handlers.IsDisallowedName(page.name)

	return nil
}

func (page *Available) Run() error {
	if !page.isDisallowed {
		var err error
		page.isAvailable, err = page.DB().IsTeamNameAvailable(page.name)
		if err != nil {
			return fmt.Errorf(
				"%w: cannot check availability at this time",
				fiber.ErrInternalServerError,
			)
		}
	}

	return nil
}

func (page *Available) RenderJson(c *fiber.Ctx) error {
	message := fiber.Map{
		"needsRewrite":  page.payload.Name != page.name,
		"name":          page.payload.Name,
		"rewrittenName": page.name,
	}

	if !page.isAvailable || page.isDisallowed {
		c.Status(fiber.StatusConflict)
		message["error"] = fmt.Sprintf("%q is already taken", page.name)
	} else {
		message["message"] = fmt.Sprintf("%q is available", page.name)
	}

	return c.JSON(message)
}
