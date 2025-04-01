package dataset

import (
	"context"
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/39alpha/dorothy/sdk"
	"github.com/gofiber/fiber/v2"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Dataset struct {
	handlers.App

	authUser *models.User
	dataset  models.Dataset
	identity peer.ID

	canRead   bool
	canWrite  bool
	canManage bool
}

func (page *Dataset) Pre(c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(page.Dorothy())
	defer cancel()

	page.authUser, _ = c.Locals("AuthUser").(*models.User)

	dataset, err := page.DB().GetDataset(page.authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return handlers.GormToFiber(err)
	}

	dataset.Manifest, err = page.Dorothy().Ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	} else if dataset.Manifest == nil {
		return fmt.Errorf("%w: failed to fetch dataset manifest", fiber.ErrInternalServerError)
	}

	page.dataset = *dataset

	if page.authUser != nil {
		page.canRead = page.authUser.CanReadDataset(*dataset)
		page.canWrite = page.authUser.CanManageDataset(*dataset)
		page.canManage = page.authUser.CanManageDataset(*dataset)
	}

	if page.authUser == nil && (dataset.IsPrivate || dataset.Team.IsPrivate) {
		return fiber.ErrUnauthorized
	} else if page.authUser != nil && !page.canRead {
		return fiber.ErrForbidden
	}

	page.identity = page.Dorothy().Ipfs.Identity

	return nil
}

func (page *Dataset) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverLogin(c, err)
}

func (page *Dataset) RenderHtml(c *fiber.Ctx) error {
	return c.Render("dataset/index", handlers.Bind(c, fiber.Map{
		"AuthUser":  page.authUser,
		"Dataset":   page.dataset,
		"CanRead":   page.canRead,
		"CanWrite":  page.canWrite,
		"CanManage": page.canManage,
	}), "layouts/main")
}

func (page *Dataset) RenderJson(c *fiber.Ctx) error {
	return c.JSON(sdk.Payload{
		Hash:         page.dataset.ManifestHash,
		PeerIdentity: page.identity,
	})
}

func (page *Dataset) RenderText(c *fiber.Ctx) error {
	return c.SendString(page.dataset.ManifestHash + "\n" + string(page.identity))
}
