package dataset

import (
	"context"
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type UpdateForm struct {
	handlers.App

	authUser *models.User
	dataset  models.Dataset
	users    []models.User
}

func (form *UpdateForm) Pre(c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(form.Dorothy())
	defer cancel()

	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}
	form.authUser = authUser

	dataset, err := form.DB().GetDataset(authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return handlers.GormToFiber(err)
	}

	dataset.Manifest, err = form.Dorothy().Ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	form.dataset = *dataset

	if !authUser.CanManageDataset(*dataset) {
		return fiber.ErrForbidden
	}

	form.users, _ = form.DB().GetUsersWithDatasetAccess(form.dataset)

	return nil
}

func (form *UpdateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("dataset/settings", handlers.Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Dataset":  form.dataset,
		"Users":    form.users,
	}), "layouts/main")
}

type Update struct {
	handlers.App

	authUser models.User
	dataset  models.Dataset
	update   models.UpdateDataset
}

func (page *Update) Pre(c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(page.Dorothy())
	defer cancel()

	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}
	page.authUser = *authUser

	dataset, err := page.DB().GetDataset(authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return err
	}

	dataset.Manifest, err = page.Dorothy().Ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	if !authUser.CanManageDataset(*dataset) {
		return fiber.ErrForbidden
	}

	if err := c.BodyParser(&page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	name := models.Slugify(page.update.Name)
	if handlers.IsDisallowedName(name) {
		return fmt.Errorf("%w: that dataset name is already taken", fiber.ErrConflict)
	}

	if dataset.Team.ID != page.update.TeamID || dataset.ID != page.update.ID {
		return fmt.Errorf("%w: There is some inconsistency in your request.", fiber.ErrBadRequest)
	}

	page.dataset = *dataset

	return nil
}

func (page *Update) Run() error {
	if _, err := page.DB().UpdateDataset(page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}
	return nil
}

func (page *Update) Post(c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataset, err := page.DB().GetDatasetById(&page.authUser, page.dataset.ID)
	if err != nil {
		return err
	}

	dataset.Manifest, err = page.Dorothy().Ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	page.dataset = *dataset

	return nil
}

func (page *Update) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverForm(&UpdateForm{
		App: page.App,
	}, c, err)
}

func (page *Update) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/" + page.dataset.Team.Name + "/" + page.dataset.Name)
}

func (page *Update) RenderJson(c *fiber.Ctx) error {
	return c.JSON(page.dataset)
}

func (*Update) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
