package dataset

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type CreateForm struct {
	team  *models.Team
	teams []models.Team
}

func (form *CreateForm) Run(c *fiber.Ctx) error {
	db := handlers.GetDB(c)

	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	if c.Params("team") == "" {
		if authUser.HasAdminRole() {
			if err := db.Find(&form.teams).Error; err != nil {
				return handlers.GormToFiber(err)
			}
		} else {
			err := db.Select("`teams`.*").
				Joins("INNER JOIN `user_team_privileges` AS `utp` ON `utp`.`team_id` = `teams`.`id`").
				Where(
					"`utp`.`user_id` = ? AND `utp`.`privilege_code` IN (?)",
					authUser.ID,
					[]models.PrivilegeCode{
						models.WritePrivilege,
						models.AdminPrivilege,
					},
				).
				Order("`teams`.`name`").
				Find(&form.teams).
				Error

			if err != nil {
				return handlers.GormToFiber(err)
			}
		}
	} else {
		var err error
		form.team, err = db.GetTeam(authUser, c.Params("team"))
		if err != nil {
			return handlers.GormToFiber(err)
		}

		if !authUser.CanWriteTeam(*form.team) {
			return fiber.ErrForbidden
		}
	}

	return nil
}

func (form *CreateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("dataset/create", handlers.Bind(c, fiber.Map{
		"Team":  form.team,
		"Teams": form.teams,
	}), "layouts/main")
}

type Create struct {
	team    *models.Team
	dataset *models.Dataset
}

func (page *Create) Run(c *fiber.Ctx) (err error) {
	db := handlers.GetDB(c)
	ipfs := handlers.GetIpfs(c)

	ctx, cancel := handlers.GetContext(c)
	defer cancel()

	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	fmt.Println(string(c.BodyRaw()))

	newDataset := models.NewDataset{}
	if err = c.BodyParser(&newDataset); err != nil {
		return fiber.ErrBadRequest
	}

	name := models.Slugify(newDataset.Name)
	if handlers.IsDisallowedName(name) {
		return fmt.Errorf("%w: that dataset name is already taken", fiber.ErrConflict)
	}

	team, err := db.GetTeam(authUser, c.Params("team"))
	if err != nil {
		return handlers.GormToFiber(err)
	}
	page.team = team

	if team.ID != newDataset.TeamID {
		return fiber.ErrBadRequest
	}

	manifest, err := ipfs.CreateEmptyManifest(ctx)
	if err != nil {
		return fmt.Errorf(
			"%w: We are having issues with IPFS at the moment. Try again later.",
			fiber.ErrInternalServerError,
		)
	}

	newDataset.Name, err = db.CreateDataset(newDataset, manifest, authUser)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	page.dataset, err = db.GetDataset(authUser, page.team.Name, newDataset.Name)
	if err != nil {
		return handlers.GormToFiber(err)
	}

	page.dataset.Manifest, err = ipfs.GetManifest(ctx, page.dataset.ManifestHash)
	if err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	return nil
}

func (page *Create) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverForm(&CreateForm{}, c, err)
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
