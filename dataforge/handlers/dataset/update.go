package dataset

import (
	"context"
	"errors"
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type UpdateForm struct {
	handlers.ErrorHandler

	authUser *models.User
	dataset  models.Dataset
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

	form.Err, _ = c.Locals("Error").(error)

	return nil
}

func (form *UpdateForm) Recover(c *fiber.Ctx, err error) error {
	var e *fiber.Error

	if errors.As(err, &e) && e == fiber.ErrUnauthorized && handlers.Redirectable(c) {
		return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
	}

	return form.ErrorFallback(c, err)
}

func (form *UpdateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("dataset/settings", handlers.Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Dataset":  form.dataset,
		"Error":    form.Err,
	}), "layouts/main")
}

type Update struct {
	handlers.ErrorHandler

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

	page.dataset = *dataset

	if !authUser.CanManageDataset(*dataset) {
		return fiber.ErrForbidden
	}

	if err := c.BodyParser(&page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	if dataset.Team.ID != page.update.TeamID || dataset.ID != page.update.ID {
		return fmt.Errorf("%w: There is some inconsistency in your request.", fiber.ErrBadRequest)
	}

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
	return page.HandleFormError(&UpdateForm{}, c, err)
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
