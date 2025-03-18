package server

import (
	"math"

	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
)

type Index struct {
	user     *models.User
	teams    []models.Team
	datasets []models.Dataset
}

func (index *Index) Preprocess(c *fiber.Ctx) error {
	index.user, _ = c.Locals("AuthUser").(*models.User)
	return nil
}

func (index *Index) Run(d *Server) error {
	teams, err := d.db.GetTeams(index.user, false)
	if err != nil {
		index.teams = []models.Team{}
	}
	teams = teams[0:int64(math.Min(float64(len(teams)), 6))]

	datasets, err := d.db.GetDatasets(index.user)
	if err != nil {
		datasets = []models.Dataset{}
	}
	datasets = datasets[0:int64(math.Min(float64(len(datasets)), 6))]

	index.teams = teams
	index.datasets = datasets

	return nil
}

func (index *Index) RenderHtml(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).Render("views/index", Bind(c, fiber.Map{
		"AuthUser": index.user,
		"Teams":    index.teams,
		"Datasets": index.datasets,
	}), "views/layouts/main")
}

func (index *Index) RenderJson(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"AuthUser": index.user,
		"Teams":    index.teams,
		"Datasets": index.datasets,
	})
}

func (index *Index) RenderText(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).SendString("Welcome to Dorothy")
}
