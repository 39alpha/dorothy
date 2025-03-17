package server

import (
	"math"

	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
)

func (d *Server) Index(c *fiber.Ctx) error {
	user, _ := c.Locals("AuthUser").(*models.User)

	teams, err := d.db.GetTeams(user, false)
	if err != nil {
		teams = []models.Team{}
	}
	teams = teams[0:int64(math.Min(float64(len(teams)), 6))]

	datasets, err := d.db.GetDatasets(user)
	if err != nil {
		datasets = []models.Dataset{}
	}
	datasets = datasets[0:int64(math.Min(float64(len(datasets)), 6))]

	return c.Render("views/index", Bind(c, fiber.Map{
		"AuthUser": user,
		"Teams":    teams,
		"Datasets": datasets,
	}), "views/layouts/main")
}
