package dataset

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Available struct {
	handlers.ErrorHandler

	payload struct {
		Name string
	}

	teamName    string
	name        string
	isAvailable bool
}

func (page *Available) Pre(c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrForbidden
	}

	if err := c.BodyParser(&page.payload); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	team, err := page.DB().GetTeam(authUser, page.teamName)
	if err != nil {
		return handlers.GormToFiber(err)
	}

	if team.IsPrivate && !authUser.CanReadTeam(*team) {
		return fiber.ErrForbidden
	}

	return nil
}

func (page *Available) Run() error {
	var err error
	page.name = models.Slugify(page.payload.Name)
	page.isAvailable, err = page.DB().IsDatasetNameAvailable(page.teamName, page.name)
	if err != nil {
		return fmt.Errorf(
			"%w: cannot check availability at this time",
			fiber.ErrInternalServerError,
		)
	}

	return nil
}

func (page *Available) RenderJson(c *fiber.Ctx) error {
	message := fiber.Map{
		"needsRewrite":  page.payload.Name != page.name,
		"name":          page.payload.Name,
		"rewrittenName": page.name,
	}

	if !page.isAvailable {
		c.Status(fiber.StatusConflict)
		message["error"] = fmt.Sprintf("%q is already taken", page.name)
	} else {
		message["message"] = fmt.Sprintf("%q is available", page.name)
	}

	return c.JSON(message)
}
