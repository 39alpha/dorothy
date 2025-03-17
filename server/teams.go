package server

import (
	"fmt"

	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
)

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

func CreateTeamForm(c *fiber.Ctx) error {
	authUser := c.Locals("AuthUser")
	if authUser == nil {
		return c.Redirect("/login?Redirect=" + c.Path())
	} else {
		return c.Render("views/create-team", Bind(c, fiber.Map{
			"AuthUser": c.Locals("AuthUser"),
			"Error":    c.Locals("Error"),
		}), "views/layouts/main")
	}
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

func (d *Server) CreateTeam(c *fiber.Ctx) error {
	authUser, ok := c.Locals("AuthUser").(*models.User)

	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).Redirect("/")
	}

	var newteam models.NewTeam
	if err := c.BodyParser(&newteam); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
	}

	team := &models.Team{
		Slug:      newteam.Slug,
		Name:      newteam.Name,
		Contact:   newteam.Contact,
		IsPrivate: newteam.IsPrivate,
	}
	if newteam.Description != nil {
		team.Description = *newteam.Description
	}
	if err := d.db.Save(team).Error; err != nil {
		c.Locals("Error", "A team with slug \""+team.Slug+"\" already exists. Try a different name.")
		return CreateTeamForm(c)
	}

	d.db.Save(&models.UserTeamPrivilege{
		User:          authUser,
		Team:          team,
		PrivilegeCode: "admin",
	})

	return c.Redirect("/" + team.Slug)
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
