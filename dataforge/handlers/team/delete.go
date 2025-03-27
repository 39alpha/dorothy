package team

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Delete struct {
	handlers.ErrorHandler

	team *models.Team
}

func (page *Delete) Pre(c *fiber.Ctx) error {
	authUser := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	team, err := page.DB().GetTeam(authUser, c.Params("team"))
	if err != nil {
		return err
	}
	page.team = team

	if !authUser.CanManageTeam(*team) {
		return fiber.ErrForbidden
	}

	return nil
}

func (page *Delete) Run() error {
	if err := page.DB().DeleteTeam(page.team); err != nil {
		return fmt.Errorf(
			"%w: We couldn't delete the team for some reason. Try again later?",
			fiber.ErrInternalServerError,
		)
	}

	return nil
}

func (page *Delete) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/")
}

func (page *Delete) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (page *Delete) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
