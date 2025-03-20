package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/39alpha/dorothy/sdk"
	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
	"github.com/libp2p/go-libp2p/core/peer"
)

type ReceiveDataset struct {
	dataset  models.Dataset
	identity peer.ID
	payload  sdk.Payload
}

func (page *ReceiveDataset) Preprocess(d *Server, c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	authUser, _ := c.Locals("AuthUser").(*models.User)

	dataset, err := d.db.GetDataset(authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return GormToFiber(err)
	}

	dataset.Manifest, err = d.Ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	page.dataset = *dataset

	if err := c.BodyParser(&page.payload); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	if authUser == nil {
		return fiber.ErrUnauthorized
	} else if !authUser.CanWriteDataset(*dataset) {
		return fiber.ErrForbidden
	}

	page.identity = d.Ipfs.Identity

	return nil
}

func (page *ReceiveDataset) Run(d *Server) error {
	ctx, cancel := context.WithTimeout(d, 10*time.Second)
	defer cancel()

	err := d.Ipfs.ConnectToPeerById(ctx, page.payload.PeerIdentity)
	if err != nil {
		if ctx.Err() == nil {
			return fmt.Errorf("%w: failed to connect to ipfs peer", fiber.ErrInternalServerError)
		} else {
			return fmt.Errorf("%w: attempt to connect to ipfs peer timed out", fiber.ErrRequestTimeout)
		}
	}

	manifest, conflicts, err := d.Receive(page.dataset.Manifest, page.payload.Hash)
	if len(conflicts) != 0 {
		return &MergeConflict{
			error:     fiber.ErrBadRequest,
			conflicts: conflicts,
		}
	} else if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	page.dataset.ManifestHash = manifest.Hash
	if err := d.db.Save(&page.dataset).Error; err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	return nil
}

func (page *ReceiveDataset) HandleError(d *Server, c *fiber.Ctx, err error) error {
	handler := ErrorHandler{err}
	handler.Preprocess(d, c)

	var conflict *MergeConflict
	if errors.As(err, &conflict) {
		if c.Accepts("application/json") != "" {
			return c.JSON(fiber.Map{
				"conflicts": conflict.conflicts,
				"error":     "merge failed with conflicts",
			})
		} else if c.Accepts("text/plain") != "" {
			msg := fmt.Sprintf("merge failed with %d conflicts", len(conflict.conflicts))
			return c.SendString(msg)
		}
	}

	return d.ErrorFallback(c, err)
}

func (page *ReceiveDataset) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/" + page.dataset.Team.Slug + "/" + page.dataset.Slug)
}

func (page *ReceiveDataset) RenderJson(c *fiber.Ctx) error {
	return c.JSON(sdk.Payload{
		Hash:         page.dataset.ManifestHash,
		PeerIdentity: page.identity,
	})
}

func (page *ReceiveDataset) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
