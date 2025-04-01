package dataset

import (
	"context"
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type CreateForm struct {
	handlers.App
	Err error

	authUser models.User
	team     models.Team
}

func (form *CreateForm) Pre(c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}
	form.authUser = *authUser

	team, err := form.DB().GetTeam(authUser, c.Params("team"))
	if err != nil {
		return handlers.GormToFiber(err)
	}
	form.team = *team

	if !authUser.CanWriteTeam(*team) {
		return fiber.ErrForbidden
	}

	form.Err, _ = c.Locals("Error").(error)

	return nil
}

func (form *CreateForm) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverLogin(c, err)
}

func (form *CreateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("dataset/create", handlers.Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Team":     form.team,
		"Error":    form.Err,
	}), "layouts/main")
}

type Create struct {
	handlers.App
	Err error

	authUser   models.User
	team       models.Team
	newDataset models.NewDataset
	dataset    *models.Dataset
}

func (page *Create) Pre(c *fiber.Ctx) (err error) {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}
	page.authUser = *authUser

	if err = c.BodyParser(&page.newDataset); err != nil {
		return fiber.ErrBadRequest
	}

	name := models.Slugify(page.newDataset.Name)
	if handlers.IsDisallowedName(name) {
		return fmt.Errorf("%w: that dataset name is already taken", fiber.ErrConflict)
	}

	team, err := page.DB().GetTeam(&page.authUser, c.Params("team"))
	if err != nil {
		return handlers.GormToFiber(err)
	}
	page.team = *team

	if team.ID != page.newDataset.TeamID {
		return fiber.ErrBadRequest
	}

	page.Err, _ = c.Locals("Error").(error)

	return nil
}

func (page *Create) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverForm(&CreateForm{
		App: page.App,
	}, c, err)
}

func (page *Create) Run() error {
	ctx, cancel := context.WithCancel(page.Dorothy())
	defer cancel()

	manifest, err := page.Dorothy().Ipfs.CreateEmptyManifest(ctx)
	if err != nil {
		return fmt.Errorf(
			"%w: We are having issues with IPFS at the moment. Try again later.",
			fiber.ErrInternalServerError,
		)
	}

	page.newDataset.Name, err = page.DB().CreateDataset(page.newDataset, manifest, &page.authUser)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	return nil
}

func (page *Create) Post(c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(page.Dorothy())
	defer cancel()

	var err error
	page.dataset, err = page.DB().GetDataset(&page.authUser, page.team.Name, page.newDataset.Name)
	if err != nil {
		return handlers.GormToFiber(err)
	}

	page.dataset.Manifest, err = page.Dorothy().Ipfs.GetManifest(ctx, page.dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	return nil
}

func (page *Create) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/" + page.team.Name + "/" + page.dataset.Name)
}

func (page *Create) RenderJson(c *fiber.Ctx) error {
	return c.JSON(page.dataset)
}

func (page *Create) RenderText(c *fiber.Ctx) error {
	return c.JSON(page.dataset.Name)
}
