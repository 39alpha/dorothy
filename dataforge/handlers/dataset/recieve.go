package dataset

import (
	"context"
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
	handlers.ErrorHandler

	dataset  models.Dataset
	identity peer.ID
	payload  sdk.Payload
}

func (page *Receive) Pre(c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	authUser, _ := c.Locals("AuthUser").(*models.User)

	dataset, err := page.DB().GetDataset(authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return handlers.GormToFiber(err)
	}

	dataset.Manifest, err = page.Dorothy().Ipfs.GetManifest(ctx, dataset.ManifestHash)
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

	page.identity = page.Dorothy().Ipfs.Identity

	return nil
}

func (page *Receive) Run() error {
	ctx, cancel := context.WithTimeout(page.Dorothy(), 10*time.Second)
	defer cancel()

	err := page.Dorothy().Ipfs.ConnectToPeerById(ctx, page.payload.PeerIdentity)
	if err != nil {
		if ctx.Err() == nil {
			return fmt.Errorf("%w: failed to connect to ipfs peer", fiber.ErrInternalServerError)
		} else {
			return fmt.Errorf("%w: attempt to connect to ipfs peer timed out", fiber.ErrRequestTimeout)
		}
	}

	manifest, conflicts, err := page.Dorothy().Receive(page.dataset.Manifest, page.payload.Hash)
	if len(conflicts) != 0 {
		return handlers.NewMergeConflict(fiber.ErrBadRequest, conflicts)
	} else if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	page.dataset.ManifestHash = manifest.Hash
	if err := page.DB().Save(&page.dataset).Error; err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	return nil
}

func (page *Receive) Recover(c *fiber.Ctx, err error) error {
	handler := &handlers.ErrorHandler{Err: err}
	_ = handler.Pre(c)

	var conflict *handlers.MergeConflict
	if errors.As(err, &conflict) {
		if c.Accepts("application/json") != "" {
			return c.JSON(fiber.Map{
				"conflicts": conflict.Conflicts,
				"error":     "merge failed with conflicts",
			})
		} else if c.Accepts("text/plain") != "" {
			msg := fmt.Sprintf("merge failed with %d conflicts", len(conflict.Conflicts))
			return c.SendString(msg)
		}
	}

	return page.ErrorFallback(c, err)
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
