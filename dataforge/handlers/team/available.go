package team

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Available struct {
	handlers.App

	desiredName  string
	name         string
	isAvailable  bool
	isDisallowed bool
}

func (page *Available) Run(c *fiber.Ctx) error {
	if err := handlers.RequireLogin(c); err != nil {
		return err
	}

	var payload struct {
		Name string
	}
	if err := c.BodyParser(&payload); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	page.desiredName = payload.Name
	page.name = models.Slugify(page.desiredName)
	page.isDisallowed = handlers.IsDisallowedName(page.name)

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
		"needsRewrite":  page.desiredName != page.name,
		"name":          page.desiredName,
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
