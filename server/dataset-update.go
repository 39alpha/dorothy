package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
)

type UpdateDatasetForm struct {
	authUser models.User
	dataset  models.Dataset
	err      error
}

func (form *UpdateDatasetForm) Preprocess(d *Server, c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}
	form.authUser = *authUser

	dataset, err := d.db.GetDataset(authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return GormToFiber(err)
	}

	dataset.Manifest, err = d.Ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	form.dataset = *dataset

	if !authUser.CanManageDataset(*dataset) {
		return fiber.ErrForbidden
	}

	form.err, _ = c.Locals("Error").(error)

	return nil
}

func (form *UpdateDatasetForm) HandleError(d *Server, c *fiber.Ctx, err error) error {
	var e *fiber.Error

	if errors.As(err, &e) && e == fiber.ErrUnauthorized && Redirectable(c) {
		return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
	}

	return d.ErrorFallback(c, err)
}

func (form *UpdateDatasetForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("dataset/settings", Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Dataset":  form.dataset,
		"Error":    form.err,
	}), "layouts/main")
}

type UpdateDataset struct {
	authUser models.User
	dataset  models.Dataset
	update   models.UpdateDataset
}

func (page *UpdateDataset) Preprocess(d *Server, c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}
	page.authUser = *authUser

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

	if err := c.BodyParser(&page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	if dataset.Team.ID != page.update.TeamID || dataset.ID != page.update.ID {
		return fmt.Errorf("%w: There is some inconsistency in your request.", fiber.ErrBadRequest)
	}

	return nil
}

func (page *UpdateDataset) Run(d *Server) error {
	if err := d.db.UpdateDataset(page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}
	return nil
}

func (page *UpdateDataset) Postprocess(d *Server, c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataset, err := d.db.GetDatasetById(&page.authUser, page.dataset.ID)
	if err != nil {
		return err
	}

	dataset.Manifest, err = d.Ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	page.dataset = *dataset

	return nil
}

func (page *UpdateDataset) HandleError(d *Server, c *fiber.Ctx, err error) error {
	return d.HandleFormError(&UpdateDatasetForm{}, c, err)
}

func (page *UpdateDataset) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/" + page.dataset.Team.Name + "/" + page.dataset.Name)
}

func (page *UpdateDataset) RenderJson(c *fiber.Ctx) error {
	return c.JSON(page.dataset)
}

func (page *UpdateDataset) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
