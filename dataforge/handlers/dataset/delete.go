package dataset

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type Delete struct {
	handlers.App

	dataset models.Dataset
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

	dataset, err := db.GetDataset(authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return err
	}

	dataset.Manifest, err = ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	page.dataset = *dataset

	if !authUser.CanManageDataset(*dataset) {
		return fiber.ErrForbidden
	}

	if err := db.DeleteDataset(&page.dataset); err != nil {
		return fmt.Errorf(
			"%w: We couldn't delete the dataset for some reason. Try again later?",
			fiber.ErrInternalServerError,
		)
	}

	if err := ipfs.UnpinManifest(ctx, page.dataset.Manifest, true); err != nil {
		return fmt.Errorf(
			"%w: We couldn't delete the dataset for some reason. Try again later?",
			fiber.ErrInternalServerError,
		)
	}

	return nil
}

func (page *Delete) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/" + page.dataset.Team.Name)
}

func (page *Delete) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (page *Delete) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
