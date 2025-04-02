package dataset

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type UpdateForm struct {
	dataset models.Dataset
	users   []models.User
}

func (form *UpdateForm) Run(c *fiber.Ctx) error {
	db := handlers.GetDB(c)
	ipfs := handlers.GetIpfs(c)

	ctx, cancel := handlers.GetContext(c)
	defer cancel()

	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	dataset, err := db.GetDataset(authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return handlers.GormToFiber(err)
	}

	dataset.Manifest, err = ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	if !authUser.CanManageDataset(*dataset) {
		return fiber.ErrForbidden
	}

	form.dataset = *dataset
	form.users, _ = db.GetUsersWithDatasetAccess(*dataset)

	return nil
}

func (form *UpdateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("dataset/settings", handlers.Bind(c, fiber.Map{
		"Dataset": form.dataset,
		"Users":   form.users,
	}), "layouts/main")
}

type Update struct {
	dataset models.Dataset
}

func (page *Update) Run(c *fiber.Ctx) error {
	db := handlers.GetDB(c)
	ipfs := handlers.GetIpfs(c)

	ctx, cancel := handlers.GetContext(c)
	defer cancel()

	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	dataset, err := db.GetDataset(authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return err
	}

	dataset.Manifest, err = ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	if !authUser.CanManageDataset(*dataset) {
		return fiber.ErrForbidden
	}

	var update models.UpdateDataset
	if err := c.BodyParser(&update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	name := models.Slugify(update.Name)
	if handlers.IsDisallowedName(name) {
		return fmt.Errorf("%w: that dataset name is already taken", fiber.ErrConflict)
	}

	if dataset.Team.ID != update.TeamID || dataset.ID != update.ID {
		return fmt.Errorf("%w: There is some inconsistency in your request.", fiber.ErrBadRequest)
	}

	if _, err := db.UpdateDataset(update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	dataset, err = db.GetDatasetById(authUser, dataset.ID)
	if err != nil {
		return err
	}

	dataset.Manifest, err = ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	page.dataset = *dataset

	return nil
}

func (page *Update) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverForm(&UpdateForm{}, c, err)
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
