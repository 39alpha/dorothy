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
	logger := handlers.GetLogger(c).Trace().Handler("dataset.CreateForm").Send()

	db := handlers.GetDB(c)

	logger.Trace().Msg("Require Authentication")
	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		logger.Debug().Msg("Unauthenticated")
		return fiber.ErrUnauthorized
	}

	if c.Params("team") == "" {
		logger.Trace().Msg("Dataset Without Team")
		if authUser.HasAdminRole() {
			logger.Trace().Msg("Admin User")

			logger.Trace().Msg("Get Teams")
			if err := db.Find(&form.teams).Error; err != nil {
				logger.Error(err).Msg("Find Teams Failed")
				return handlers.GormToFiber(err)
			}
		} else {
			logger.Trace().Msg("Basic User")

			logger.Trace().Msg("Get Teams")
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
				logger.Error(err).Msg("Find Teams Failed")
				return handlers.GormToFiber(err)
			}
		}
	} else {
		logger.Trace().Msg("Dataset With Team")

		teamName := c.Params("team")
		logger.Trace().Str("team", teamName).Msg("Get Team")
		var err error
		form.team, err = db.GetTeam(authUser, teamName)
		if err != nil {
			logger.Error(err).Str("team", teamName).Msg("Get Team Failed")
			return handlers.GormToFiber(err)
		}

		logger.Trace().Msg("Write Access")
		if !authUser.CanWriteTeam(*form.team) {
			logger.Error(fiber.ErrForbidden).Msg("No Write Access")
			return fiber.ErrForbidden
		}
	}

	return nil
}

func (form *CreateForm) RenderHtml(c *fiber.Ctx) error {
	handlers.GetLogger(c).Trace().Renderer("dataset.CreateForm", "RenderHtml").Send()
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
	logger := handlers.GetLogger(c).Trace().Handler("dataset.Create").Send()

	db := handlers.GetDB(c)
	ipfs := handlers.GetIpfs(c)

	ctx, cancel := handlers.GetContext(c)
	defer cancel()

	logger.Trace().Msg("Require Authentication")
	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		logger.Trace().Msg("Unauthenticated")
		return fiber.ErrUnauthorized
	}

	logger.Trace().Msg("Parse Request Body")
	newDataset := models.NewDataset{}
	if err = c.BodyParser(&newDataset); err != nil {
		logger.Debug().Err(err).Str("body", string(c.BodyRaw())).Msg("Bad Request")
		return fiber.ErrBadRequest
	}

	logger.Trace().Msg("Is Valid Name")
	name := models.Slugify(newDataset.Name)
	if handlers.IsDisallowedName(name) {
		logger.Debug().Str("name", name).Msg("Disallowed Name")
		return fmt.Errorf("%w: that dataset name is already taken", fiber.ErrConflict)
	}

	teamName := c.Params("team")

	if teamName != "" {
		logger.Trace().Str("team", teamName).Msg("Get Team By Name")
		page.team, err = db.GetTeam(authUser, teamName)
		if err != nil {
			logger.Error(err).Str("team", teamName).Msg("Get Team By Name Failed")
			return handlers.GormToFiber(err)
		}

		logger.Trace().Msg("Check Team ID Consistency")
		if page.team.ID != newDataset.TeamID {
			logger.Error(fmt.Errorf("inconsistent team ids")).Msg("Bad Request")
			return fiber.ErrBadRequest
		}
	} else {
		logger.Trace().Uint("team_id", newDataset.TeamID).Msg("Get Team By ID")
		page.team, err = db.GetTeamById(authUser, newDataset.TeamID)
		if err != nil {
			logger.Error(err).Str("team", teamName).Msg("Get Team By Name Failed")
			return handlers.GormToFiber(err)
		}
	}

	logger.Trace().Msg("Create Empty Manifest")
	manifest, err := ipfs.CreateEmptyManifest(ctx)
	if err != nil {
		logger.Error(err).Msg("Create Empty Manifest Failed")
		return fmt.Errorf(
			"%w: We are having issues with IPFS at the moment. Try again later.",
			fiber.ErrInternalServerError,
		)
	}

	logger.Trace().Msg("Create Database")
	newDataset.Name, err = db.CreateDataset(newDataset, manifest, authUser)
	if err != nil {
		logger.Error(err).Msg("Create Database Failed")
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	logger.Trace().
		Str("team", page.team.Name).
		Str("dataset", newDataset.Name).
		Msg("Get Database")
	page.dataset, err = db.GetDataset(authUser, page.team.Name, newDataset.Name)
	if err != nil {
		logger.Error(err).
			Str("team", page.team.Name).
			Str("dataset", newDataset.Name).
			Msg("Get Database Failed")

		return handlers.GormToFiber(err)
	}

	logger.Trace().Msg("Get Manifest")
	page.dataset.Manifest, err = ipfs.GetManifest(ctx, page.dataset.ManifestHash)
	if err != nil {
		logger.Error(err).Msg("Get Manifest Failed")
		return fmt.Errorf("%w: %v", fiber.ErrInternalServerError, err)
	}

	return nil
}

func (page *Create) Recover(c *fiber.Ctx, err error) error {
	handlers.GetLogger(c).Trace().Recover("dataset.Create").Send()
	return handlers.RecoverForm(&CreateForm{}, c, err)
}

func (page *Create) RenderHtml(c *fiber.Ctx) error {
	handlers.GetLogger(c).Trace().Renderer("dataset.Create", "RenderHtml").Send()
	return c.Redirect("/" + page.team.Name + "/" + page.dataset.Name)
}

func (page *Create) RenderJson(c *fiber.Ctx) error {
	handlers.GetLogger(c).Trace().Renderer("dataset.Create", "RenderJson").Send()
	return c.JSON(page.dataset)
}

func (page *Create) RenderText(c *fiber.Ctx) error {
	handlers.GetLogger(c).Trace().Renderer("dataset.Create", "RenderText").Send()
	return c.SendString(page.dataset.Name)
}
