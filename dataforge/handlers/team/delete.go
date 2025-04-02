package team

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/gofiber/fiber/v2"
)

type Delete struct {
	handlers.App
}

func (page *Delete) Run(c *fiber.Ctx) error {
	db := handlers.GetDB(c)
	ipfs := handlers.GetIpfs(c)

	ctx, cancel := handlers.GetContext(c)
	defer cancel()

	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	team, err := db.GetTeam(authUser, c.Params("team"))
	if err != nil {
		return err
	}

	if !authUser.CanManageTeam(*team) {
		return fiber.ErrForbidden
	}

	errs := []error{}
	for _, dataset := range team.Datasets {
		if err := db.DeleteDataset(&dataset); err != nil {
			errs = append(errs, err)
			continue
		}

		if dataset.Manifest == nil && dataset.ManifestHash != "" {
			var err error
			dataset.Manifest, err = ipfs.GetManifest(ctx, dataset.ManifestHash)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}

		if err := ipfs.UnpinManifest(ctx, dataset.Manifest, true); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return fmt.Errorf(
			"%w: We could not delete the datasets owned by the team for some reason. Try again later?",
			fiber.ErrInternalServerError,
		)
	}

	if err := db.DeleteTeam(team); err != nil {
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
