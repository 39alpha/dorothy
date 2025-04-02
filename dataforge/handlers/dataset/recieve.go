package dataset

import (
	"errors"
	"fmt"
	"time"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/39alpha/dorothy/sdk"
	"github.com/gofiber/fiber/v2"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Receive struct {
	handlers.App

	dataset  models.Dataset
	identity peer.ID
}

func (page *Receive) Run(c *fiber.Ctx) error {
	db := handlers.GetDB(c)
	dorothy := handlers.GetDorothy(c)

	ctx, cancel := handlers.GetContext(c, 15*time.Second)
	defer cancel()

	authUser := handlers.GetAuthUser(c)

	dataset, err := db.GetDataset(authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return handlers.GormToFiber(err)
	}

	dataset.Manifest, err = dorothy.Ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	page.dataset = *dataset

	var payload sdk.Payload
	if err := c.BodyParser(&payload); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	if authUser == nil {
		return fiber.ErrUnauthorized
	} else if !authUser.CanWriteDataset(*dataset) {
		return fiber.ErrForbidden
	}

	page.identity = dorothy.Ipfs.Identity

	if err := dorothy.Ipfs.ConnectToPeerById(ctx, payload.PeerIdentity); err != nil {
		if ctx.Err() == nil {
			return fmt.Errorf("%w: failed to connect to ipfs peer", fiber.ErrInternalServerError)
		} else {
			return fmt.Errorf("%w: attempt to connect to ipfs peer timed out", fiber.ErrRequestTimeout)
		}
	}

	manifest, conflicts, err := dorothy.Receive(page.dataset.Manifest, payload.Hash)
	if len(conflicts) != 0 {
		return handlers.NewMergeConflict(fiber.ErrBadRequest, conflicts)
	} else if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	page.dataset.ManifestHash = manifest.Hash
	if err := db.Save(&page.dataset).Error; err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	return nil
}

func (page *Receive) Recover(c *fiber.Ctx, err error) error {
	var conflict *handlers.MergeConflict
	if errors.As(err, &conflict) {
		c.Status(fiber.StatusBadRequest)
		if handlers.AcceptsJson(c) {
			return c.JSON(fiber.Map{
				"conflicts": conflict.Conflicts,
				"error":     "merge failed with conflicts",
			})
		} else if c.Accepts("text/plain") != "" {
			msg := fmt.Sprintf("merge failed with %d conflicts", len(conflict.Conflicts))
			return c.SendString(msg)
		}
	}
	return err
}

func (page *Receive) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/" + page.dataset.Team.Name + "/" + page.dataset.Name)
}

func (page *Receive) RenderJson(c *fiber.Ctx) error {
	return c.JSON(sdk.Payload{
		Hash:         page.dataset.ManifestHash,
		PeerIdentity: page.identity,
	})
}

func (*Receive) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
