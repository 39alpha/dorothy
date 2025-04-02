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

	team models.Team
}

func (form *CreateForm) Run(c *fiber.Ctx) error {
	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	team, err := form.DB().GetTeam(authUser, c.Params("team"))
	if err != nil {
		return handlers.GormToFiber(err)
	}
	form.team = *team

	if !authUser.CanWriteTeam(*team) {
		return fiber.ErrForbidden
	}

	return nil
}

func (form *CreateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("dataset/create", handlers.Bind(c, fiber.Map{
		"Team": form.team,
	}), "layouts/main")
}

type Create struct {
	handlers.App

	team    *models.Team
	dataset *models.Dataset
}

func (page *Create) Run(c *fiber.Ctx) (err error) {
	ctx, cancel := context.WithCancel(page.Dorothy())
	defer cancel()

	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	newDataset := models.NewDataset{}
	if err = c.BodyParser(&newDataset); err != nil {
		return fiber.ErrBadRequest
	}

	name := models.Slugify(newDataset.Name)
	if handlers.IsDisallowedName(name) {
		return fmt.Errorf("%w: that dataset name is already taken", fiber.ErrConflict)
	}

	team, err := page.DB().GetTeam(authUser, c.Params("team"))
	if err != nil {
		return handlers.GormToFiber(err)
	}
	page.team = team

	if team.ID != newDataset.TeamID {
		return fiber.ErrBadRequest
	}

	manifest, err := page.Dorothy().Ipfs.CreateEmptyManifest(ctx)
	if err != nil {
		return fmt.Errorf(
			"%w: We are having issues with IPFS at the moment. Try again later.",
			fiber.ErrInternalServerError,
		)
	}

	newDataset.Name, err = page.DB().CreateDataset(newDataset, manifest, authUser)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	page.dataset, err = page.DB().GetDataset(authUser, page.team.Name, newDataset.Name)
	if err != nil {
		return handlers.GormToFiber(err)
	}

	page.dataset.Manifest, err = page.Dorothy().Ipfs.GetManifest(ctx, page.dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	return nil
}

func (page *Create) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverForm(&CreateForm{
		App: page.App,
	}, c, err)
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
