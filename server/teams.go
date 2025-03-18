package server

import (
	"fmt"

	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CreateTeamForm struct {
	authUser *models.User
}

func (form *CreateTeamForm) Preprocess(c *fiber.Ctx) error {
	form.authUser = c.Locals("AuthUser").(*models.User)
	return nil
}

func (form *CreateTeamForm) RenderHtml(c *fiber.Ctx) error {
	if form.authUser == nil {
		return c.Redirect("/login?Redirect=" + c.Path())
	} else {
		return c.Render("views/create-team", Bind(c, fiber.Map{
			"AuthUser": c.Locals("AuthUser"),
			"Error":    c.Locals("Error"),
		}), "views/layouts/main")
	}
}

type CreateTeam struct {
	authUser models.User
	newTeam  models.NewTeam
	team     *models.Team
}

func (page *CreateTeam) Preprocess(c *fiber.Ctx) error {
	authUser, ok := c.Locals("AuthUser").(*models.User)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).Redirect("/")
	}

	var newTeam models.NewTeam
	if err := c.BodyParser(&newTeam); err != nil {
		// return fmt.Errorf()
		// return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
	}

	page.authUser = *authUser
	page.newTeam = newTeam

	return nil
}

func (page *CreateTeam) Run(d *Server) error {
	team := &models.Team{
		Slug:      page.newTeam.Slug,
		Name:      page.newTeam.Name,
		Contact:   page.newTeam.Contact,
		IsPrivate: page.newTeam.IsPrivate,
	}
	if page.newTeam.Description != nil {
		team.Description = *page.newTeam.Description
	}

	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := d.db.Save(team).Error; err != nil {
			return fmt.Errorf(
				"%w: A team with slug \"%s\" already exists. Try a different name.",
				fiber.ErrBadRequest,
				team.Slug,
			)
		}

		err := d.db.Save(&models.UserTeamPrivilege{
			User:          &page.authUser,
			Team:          team,
			PrivilegeCode: "admin",
		})

		if err != nil {
			return fmt.Errorf(
				"%w: We could not create you team at this time. Try again later?",
				fiber.ErrInternalServerError,
			)
		}

		page.team = team

		return nil
	})
}

func (page *CreateTeam) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/" + page.team.Slug)
}

func (page *CreateTeam) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (page *CreateTeam) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}

func GetTeam(c *fiber.Ctx) error {
	team, ok := c.Locals("Team").(*models.Team)
	if !ok {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	canRead := false
	canWrite := false
	canManage := false

	authUser, ok := c.Locals("AuthUser").(*models.User)
	if ok {
		canRead = authUser.CanReadTeam(*team)
		canWrite = authUser.CanManageTeam(*team)
		canManage = authUser.CanManageTeam(*team)
	}

	return c.Render("views/team", Bind(c, fiber.Map{
		"AuthUser":  authUser,
		"CanRead":   canRead,
		"CanWrite":  canWrite,
		"CanManage": canManage,
	}), "views/layouts/main")
}

func TeamSettingsForm(c *fiber.Ctx) error {
	team, ok := c.Locals("Team").(*models.Team)
	if !ok || team == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	user, ok := c.Locals("AuthUser").(*models.User)
	if !ok {
		return Redirect(c, fiber.StatusUnauthorized, "/login?Redirect="+c.Path(), fiber.Map{
			"error": "unauthorized",
		}, "unauthorized")
	} else if !user.CanManageTeam(*team) {
		return Redirect(c, fiber.StatusForbidden, "/"+team.Slug, fiber.Map{
			"error": "forbidden",
		}, "forbidden")
	}

	return c.Render("views/edit-team", Bind(c, fiber.Map{
		"AuthUser": user,
		"Error":    c.Locals("Error"),
	}), "views/layouts/main")
}

func (d *Server) LoadTeams(c *fiber.Ctx) error {
	user, _ := c.Locals("AuthUser").(*models.User)

	teams, err := d.db.GetTeams(user, true)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).Redirect("/")
	}

	AddState(c, "Teams", teams)

	return c.Next()
}

func (d *Server) LoadTeam(c *fiber.Ctx) error {
	slug := c.Params("team")
	if slug == "" {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	team := models.Team{Slug: slug}
	if err := d.db.Preload("Datasets.Team").Where(&team).First(&team).Error; err != nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	datasets := []models.Dataset{}
	if c.Locals("AuthUser") == nil {
		if team.IsPrivate {
			return c.Status(fiber.StatusUnauthorized).Redirect("/login?Redirect=" + c.Path())
		}
		for _, dataset := range team.Datasets {
			if !dataset.IsPrivate {
				datasets = append(datasets, dataset)
			}
		}
	} else {
		user := c.Locals("AuthUser").(*models.User)
		if !user.CanReadTeam(team) {
			return c.Status(fiber.StatusForbidden).Redirect("/")
		}
		for _, dataset := range team.Datasets {
			if user.CanReadDataset(dataset) {
				datasets = append(datasets, dataset)
			}
		}
	}
	team.Datasets = datasets

	AddState(c, "Team", &team)

	return c.Next()
}

func (d *Server) UpdateTeamSettings(c *fiber.Ctx) error {
	authUser, ok := c.Locals("AuthUser").(*models.User)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).Redirect("/")
	}

	team, ok := c.Locals("Team").(*models.Team)
	if !ok || team == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	user, ok := c.Locals("AuthUser").(*models.User)
	if !ok {
		return Redirect(c, fiber.StatusUnauthorized, "/login?Redirect="+c.Path(), fiber.Map{
			"error": "unauthorized",
		}, "unauthorized")
	} else if !user.CanWriteTeam(*team) {
		return Redirect(c, fiber.StatusForbidden, "/"+team.Slug, fiber.Map{
			"error": "forbidden",
		}, "forbidden")
	}

	var updated models.UpdateTeam
	if err := c.BodyParser(&updated); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
	}

	if err := d.db.UpdateTeam(updated); err != nil {
		c.Locals("Error", "A team already has slag \""+updated.Slug+"\". Try a different name.")
		return CreateDatasetForm(c)
	}

	return c.Redirect("/" + updated.Slug)
}
