package dataforge

import (
	"context"
	"errors"
	"fmt"

	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type CreateDatasetForm struct {
	authUser models.User
	team     models.Team
	err      error
}

func (form *CreateDatasetForm) Preprocess(d *Server, c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}
	form.authUser = *authUser

	team, err := d.db.GetTeam(authUser, c.Params("team"))
	if err != nil {
		return GormToFiber(err)
	}
	form.team = *team

	if !authUser.CanWriteTeam(*team) {
		return fiber.ErrForbidden
	}

	form.err, _ = c.Locals("Error").(error)

	return nil
}

func (form *CreateDatasetForm) HandleError(d *Server, c *fiber.Ctx, err error) error {
	var e *fiber.Error

	if errors.As(err, &e) && e == fiber.ErrUnauthorized && Redirectable(c) {
		return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
	}

	return d.ErrorFallback(c, err)
}

func (form *CreateDatasetForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("dataset/create", Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Team":     form.team,
		"Error":    form.err,
	}), "layouts/main")
}

type CreateDataset struct {
	authUser   models.User
	team       models.Team
	newDataset models.NewDataset
	dataset    *models.Dataset
	err        error
}

func (page *CreateDataset) Preprocess(d *Server, c *fiber.Ctx) (err error) {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}
	page.authUser = *authUser

	if err = c.BodyParser(&page.newDataset); err != nil {
		return fiber.ErrBadRequest
	}

	team, err := d.db.GetTeam(&page.authUser, c.Params("team"))
	if err != nil {
		return GormToFiber(err)
	}
	page.team = *team

	if team.ID != page.newDataset.TeamID {
		return fiber.ErrBadRequest
	}

	page.err, _ = c.Locals("Error").(error)

	return nil
}

func (page *CreateDataset) HandleError(d *Server, c *fiber.Ctx, err error) error {
	return d.HandleFormError(&CreateDatasetForm{}, c, err)
}

func (page *CreateDataset) Run(d *Server) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manifest, err := d.Ipfs.CreateEmptyManifest(ctx)
	if err != nil {
		return fmt.Errorf(
			"%w: We are having issues with IPFS at the moment. Try again later.",
			fiber.ErrInternalServerError,
		)
	}

	if err := d.db.CreateDataset(page.newDataset, manifest, &page.authUser); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	return nil
}

func (page *CreateDataset) Postprocess(d *Server, c *fiber.Ctx) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var err error
	page.dataset, err = d.db.GetDataset(&page.authUser, page.team.Name, page.newDataset.Name)
	if err != nil {
		return GormToFiber(err)
	}

	page.dataset.Manifest, err = d.Ipfs.GetManifest(ctx, page.dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	return nil
}

func (page *CreateDataset) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/" + page.team.Name + "/" + page.dataset.Name)
}

func (page *CreateDataset) RenderJson(c *fiber.Ctx) error {
	return c.JSON(page.dataset)
}

func (page *CreateDataset) RenderText(c *fiber.Ctx) error {
	return c.JSON(page.dataset.Name)
}
