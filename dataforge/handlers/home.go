package handlers

import (
	"math"

	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Home struct {
	ErrorHandler

	user     *models.User
	teams    []models.Team
	datasets []models.Dataset
}

func (page *Home) Pre(c *fiber.Ctx) error {
	page.user, _ = c.Locals("AuthUser").(*models.User)

	teams, err := page.DB().GetTeams(page.user, false)
	if err != nil {
		page.teams = []models.Team{}
	}
	teams = teams[0:int64(math.Min(float64(len(teams)), 6))]

	datasets, err := page.DB().GetDatasets(page.user)
	if err != nil {
		datasets = []models.Dataset{}
	}
	datasets = datasets[0:int64(math.Min(float64(len(datasets)), 6))]

	page.teams = teams
	page.datasets = datasets

	return nil
}

func (page *Home) RenderHtml(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).Render("index", Bind(c, fiber.Map{
		"AuthUser": page.user,
		"Teams":    page.teams,
		"Datasets": page.datasets,
	}), "layouts/main")
}

func (page *Home) RenderJson(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"AuthUser": page.user,
		"Teams":    page.teams,
		"Datasets": page.datasets,
	})
}

func (*Home) RenderText(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).SendString("Welcome to Dorothy")
}
