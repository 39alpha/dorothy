package server

import (
	"fmt"

	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
)

type DatasetAvailability struct {
	payload struct {
		Name string
	}

	teamName    string
	name        string
	isAvailable bool
}

func (page *DatasetAvailability) Preprocess(d *Server, c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrForbidden
	}

	if err := c.BodyParser(&page.payload); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	team, err := d.db.GetTeam(authUser, page.teamName)
	if err != nil {
		return GormToFiber(err)
	}

	if team.IsPrivate && !authUser.CanReadTeam(*team) {
		return fiber.ErrForbidden
	}

	return nil
}

func (page *DatasetAvailability) Run(d *Server) error {
	var err error
	page.name = models.Slugify(page.payload.Name)
	page.isAvailable, err = d.db.IsDatasetNameAvailable(page.teamName, page.name)
	if err != nil {
		return fmt.Errorf(
			"%w: cannot check availability at this time",
			fiber.ErrInternalServerError,
		)
	}

	return nil
}

func (page *DatasetAvailability) RenderJson(c *fiber.Ctx) error {
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
