package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/39alpha/dorothy/sdk"
	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
	"github.com/libp2p/go-libp2p/core/peer"
)

type GetDataset struct {
	authUser  *models.User
	dataset   models.Dataset
	identity  peer.ID
	canRead   bool
	canWrite  bool
	canManage bool
}

func (page *GetDataset) Preprocess(d *Server, c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	page.authUser, _ = c.Locals("AuthUser").(*models.User)

	dataset, err := d.db.GetDataset(page.authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return GormToFiber(err)
	}

	dataset.Manifest, err = d.Ipfs.GetManifest(ctx, dataset.ManifestHash)
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

	page.identity = d.Ipfs.Identity

	return nil
}

func (page *GetDataset) HandleError(d *Server, c *fiber.Ctx, err error) error {
	var e *fiber.Error
	if errors.As(err, &e) && e == fiber.ErrUnauthorized && Redirectable(c) {
		return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
	}

	return d.ErrorFallback(c, err)
}

func (page *GetDataset) RenderHtml(c *fiber.Ctx) error {
	return c.Render("views/dataset/index", Bind(c, fiber.Map{
		"AuthUser":  page.authUser,
		"Dataset":   page.dataset,
		"CanRead":   page.canRead,
		"CanWrite":  page.canWrite,
		"CanManage": page.canManage,
	}), "views/layouts/main")
}

func (page *GetDataset) RenderJson(c *fiber.Ctx) error {
	return c.JSON(sdk.Payload{
		Hash:         page.dataset.ManifestHash,
		PeerIdentity: page.identity,
	})
}

func (page *GetDataset) RenderText(c *fiber.Ctx) error {
	return c.SendString(page.dataset.ManifestHash + "\n" + string(page.identity))
}
