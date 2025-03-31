package handlers

import (
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Home struct {
	ErrorHandler

	user     *models.User
	datasets []models.Dataset
}

func (page *Home) Pre(c *fiber.Ctx) error {
	page.user, _ = c.Locals("AuthUser").(*models.User)

	page.datasets, _ = page.DB().GetHotDatasets(page.user)

	return nil
}

func (page *Home) RenderHtml(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).Render("index", Bind(c, fiber.Map{
		"AuthUser": page.user,
		"Datasets": page.datasets,
	}), "layouts/main")
}

func (page *Home) RenderJson(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"AuthUser": page.user,
		"Datasets": page.datasets,
	})
}

func (*Home) RenderText(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).SendString("Welcome to Dorothy")
}
