package handlers

import (
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Home struct {
	App

	datasets []models.Dataset
}

func (page *Home) Pre(c *fiber.Ctx) error {
	page.datasets, _ = page.DB().GetHotDatasets(GetAuthUser(c))

	return nil
}

func (page *Home) RenderHtml(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).Render("index", Bind(c, fiber.Map{
		"Datasets": page.datasets,
	}), "layouts/main")
}

func (page *Home) RenderJson(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"Datasets": page.datasets,
	})
}

func (*Home) RenderText(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).SendString("Welcome to Dorothy")
}
