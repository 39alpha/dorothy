package server

import (
	"context"
	"fmt"

	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
)

type DeleteDataset struct {
	dataset models.Dataset
}

func (page *DeleteDataset) Preprocess(d *Server, c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	authUser := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	dataset, err := d.db.GetDataset(authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return err
	}

	dataset.Manifest, err = d.Ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	page.dataset = *dataset

	if !authUser.CanManageDataset(*dataset) {
		return fiber.ErrForbidden
	}

	return nil
}

func (page *DeleteDataset) Run(d *Server) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := d.db.DeleteDataset(&page.dataset); err != nil {
		return fmt.Errorf(
			"%w: We couldn't delete the dataset for some reason. Try again later?",
			fiber.ErrInternalServerError,
		)
	}

	if err := d.Ipfs.UnpinManifest(ctx, page.dataset.Manifest, true); err != nil {
		return fmt.Errorf(
			"%w: We couldn't delete the dataset for some reason. Try again later?",
			fiber.ErrInternalServerError,
		)
	}

	return nil
}

func (page *DeleteDataset) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/" + page.dataset.Team.Slug)
}

func (page *DeleteDataset) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (page *DeleteDataset) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
