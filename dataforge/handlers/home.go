package handlers

import (
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Home struct {
	datasets []models.Dataset
}

func (page *Home) Run(c *fiber.Ctx) error {
	logger := GetLogger(c).Trace().Handler("Home").Send()
	authUser := GetAuthUser(c)

	logger.Trace().Msg("Get Hot Dataset")
	var err error
	page.datasets, err = GetDB(c).GetHotDatasets(authUser)
	if err != nil {
		event := logger.Error(err)
		if authUser != nil {
			event.Bool("authenticated", true).Uint("user_id", authUser.ID)
		} else {
			event.Bool("authenticated", false)
		}
		event.Msg("Get Hot Datasets Failed")
	}

	return nil
}

func (page *Home) RenderHtml(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("Home", "RenderHtml").Send()

	return c.Status(fiber.StatusOK).Render("index", Bind(c, fiber.Map{
		"Datasets": page.datasets,
	}), "layouts/main")
}

func (page *Home) RenderJson(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("Home", "RenderJson").Send()

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"Datasets": page.datasets,
	})
}

func (*Home) RenderText(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("Home", "RenderText").Send()

	return c.Status(fiber.StatusOK).SendString("Welcome to Dorothy")
}
