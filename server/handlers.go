package server

import (
	"fmt"
	"maps"
	"time"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/core/model"
	"github.com/39alpha/dorothy/server/auth"
	"github.com/gofiber/fiber/v2"
)

func bind(c *fiber.Ctx, local fiber.Map) fiber.Map {
	bind := fiber.Map{}

	state, ok := c.Locals("State").(fiber.Map)
	if !ok || state != nil {
		maps.Copy(bind, state)
	}
	maps.Copy(bind, local)

	return bind
}

func addState(c *fiber.Ctx, key string, value interface{}) {
	state, ok := c.Locals("State").(fiber.Map)
	if !ok || state == nil {
		state = fiber.Map{
			key: value,
		}
	} else {
		state[key] = value
	}
	c.Locals("State", state)
	c.Locals(key, value)
}

func Index(c *fiber.Ctx) error {
	return c.Render("views/index", bind(c, fiber.Map{
		"AuthUser": c.Locals("AuthUser"),
	}), "views/layouts/main")
}

func RegistrationForm(c *fiber.Ctx) error {
	return c.Render("views/register", bind(c, fiber.Map{
		"AuthUser": c.Locals("AuthUser"),
	}), "views/layouts/main")
}

func Registration(db *core.DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var new_user model.NewUser
		if err := c.BodyParser(&new_user); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
		}

		if err := db.CreateUser(&new_user); err != nil {
			return c.Status(fiber.StatusInternalServerError).RedirectBack("register")
		}

		return c.Redirect("/login")
	}
}

func LoginForm(c *fiber.Ctx) error {
	if c.Locals("AuthUser") != nil {
		return c.Redirect("/")
	}

	bindings := bind(c, fiber.Map{
		"AuthUser": c.Locals("AuthUser"),
	})

	if c.Query("Redirect") != "" {
		bindings["Redirect"] = c.Query("Redirect")
	}

	return c.Render("views/login", bindings, "views/layouts/main")
}

func Login(jwtAuth *auth.Auth, db *core.DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var fields struct {
			Redirect string
		}
		c.BodyParser(&fields)

		var login model.UserLogin
		if err := c.BodyParser(&login); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
		}

		if err := db.ValidateCredentials(login.Email, login.Password); err != nil {
			return c.Status(fiber.StatusUnauthorized).RedirectBack("login")
		}

		user := &model.User{Email: login.Email}
		err := db.Select("id", "email", "name", "orcid").Where(user).First(user).Error
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).RedirectBack("login")
		}

		token, err := jwtAuth.MakeToken(user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).RedirectBack("login")
		}

		c.Cookie(&fiber.Cookie{
			Name:    "jwt",
			Value:   token,
			Expires: time.Now().Add(72 * time.Hour),
		})

		if fields.Redirect == "" {
			return c.Redirect("/")
		} else {
			return c.Redirect(fields.Redirect)
		}
	}
}

func Logout(c *fiber.Ctx) error {
	c.ClearCookie("jwt")
	return c.Redirect("/")
}

func GetOrganizations(db *core.DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgs := []model.Organization{}
		if err := db.Preload("Datasets.Organization").Find(&orgs).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).Redirect("/")
		}

		organizations := []model.Organization{}

		if c.Locals("AuthUser") == nil {
			for _, org := range orgs {
				if org.IsPrivate {
					continue
				}

				datasets := []model.Dataset{}
				for _, dataset := range org.Datasets {
					if !dataset.IsPrivate {
						datasets = append(datasets, dataset)
					}
				}
				org.Datasets = datasets

				organizations = append(organizations, org)
			}
		} else {
			user := c.Locals("AuthUser").(*model.User)
			for _, org := range orgs {
				if !user.CanReadOrganization(org) {
					continue
				}

				datasets := []model.Dataset{}
				for _, dataset := range org.Datasets {
					if user.CanReadDataset(dataset) {
						datasets = append(datasets, dataset)
					}
				}
				org.Datasets = datasets

				organizations = append(organizations, org)
			}
		}

		addState(c, "Organizations", organizations)

		return c.Next()
	}
}

func CreateOrganizationForm(c *fiber.Ctx) error {
	authUser := c.Locals("AuthUser")
	if authUser == nil {
		return c.Redirect("/login?Redirect=" + c.Path())
	} else {
		return c.Render("views/create-organization", bind(c, fiber.Map{
			"AuthUser": c.Locals("AuthUser"),
		}), "views/layouts/main")
	}
}

func CreateOrganization(db *core.DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authUser, ok := c.Locals("AuthUser").(*model.User)

		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).Redirect("/")
		}

		var neworg model.NewOrganization
		if err := c.BodyParser(&neworg); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
		}

		org := &model.Organization{
			Slug:      neworg.Slug,
			Name:      neworg.Name,
			Contact:   neworg.Contact,
			IsPrivate: neworg.IsPrivate,
		}
		if neworg.Description != nil {
			org.Description = *neworg.Description
		}
		if err := db.Save(org).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).Redirect("/")
		}

		db.Save(&model.UserOrganizationPrivilege{
			User:          authUser,
			Organization:  org,
			PrivilegeCode: "admin",
		})

		return c.Redirect("/" + org.Slug)
	}
}

func GetOrganization(db *core.DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		slug := c.Params("organization")
		if slug == "" {
			return c.Status(fiber.StatusNotFound).Redirect("/")
		}

		org := model.Organization{Slug: slug}
		if err := db.Preload("Datasets.Organization").Where(&org).First(&org).Error; err != nil {
			return c.Status(fiber.StatusNotFound).Redirect("/")
		}

		datasets := []model.Dataset{}
		if c.Locals("AuthUser") == nil {
			if org.IsPrivate {
				return c.Status(fiber.StatusUnauthorized).Redirect("/login?Redirect=" + c.Path())
			}
			for _, dataset := range org.Datasets {
				if !dataset.IsPrivate {
					datasets = append(datasets, dataset)
				}
			}
		} else {
			user := c.Locals("AuthUser").(*model.User)
			if !user.CanReadOrganization(org) {
				return c.Status(fiber.StatusForbidden).Redirect("/")
			}
			for _, dataset := range org.Datasets {
				if user.CanReadDataset(dataset) {
					datasets = append(datasets, dataset)
				}
			}
		}
		org.Datasets = datasets

		addState(c, "Organization", &org)

		return c.Next()
	}
}

func Organization(c *fiber.Ctx) error {
	return c.Render("views/organization", bind(c, fiber.Map{
		"AuthUser": c.Locals("AuthUser"),
	}), "views/layouts/main")
}

func CreateDatasetForm(c *fiber.Ctx) error {
	authUser := c.Locals("AuthUser")
	org, ok := c.Locals("Organization").(*model.Organization)
	if !ok || org == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	if authUser == nil {
		return c.Redirect("/login?Redirect=" + c.Path())
	} else {
		return c.Render("views/create-dataset", bind(c, fiber.Map{
			"AuthUser": c.Locals("AuthUser"),
		}), "views/layouts/main")
	}
}

func CreateDataset(db *core.DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authUser, ok := c.Locals("AuthUser").(*model.User)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).Redirect("/")
		}

		org, ok := c.Locals("Organization").(*model.Organization)
		if !ok || org == nil {
			return c.Status(fiber.StatusNotFound).Redirect("/")
		}

		var newdata model.NewDataset
		if err := c.BodyParser(&newdata); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
		}

		if org.ID != newdata.OrganizationID {
			return c.Redirect("/"+org.Slug+"/dataset/create", 400)
		}

		dataset := &model.Dataset{
			Slug:           newdata.Slug,
			Name:           newdata.Name,
			OrganizationID: newdata.OrganizationID,
			Contact:        newdata.Contact,
			IsPrivate:      newdata.IsPrivate,
		}
		if newdata.Description != nil {
			dataset.Description = *newdata.Description
		}
		if err := db.Save(dataset).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).Redirect("/")
		}

		db.Save(&model.UserDatasetPrivilege{
			User:          authUser,
			Dataset:       dataset,
			PrivilegeCode: "admin",
		})

		return c.Redirect("/" + org.Slug + "/" + dataset.Slug)
	}
}

func GetDataset(db *core.DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		org, ok := c.Locals("Organization").(*model.Organization)
		if !ok || org == nil {
			return c.Status(fiber.StatusNotFound).Redirect("/")
		}

		datasetSlug := c.Params("dataset")
		if datasetSlug == "" {
			return c.Status(fiber.StatusBadRequest).Redirect("/" + org.Slug)
		}

		dataset := model.Dataset{
			Slug:           datasetSlug,
			OrganizationID: org.ID,
		}
		if err := db.Preload("Organization").Where(&dataset).First(&dataset).Error; err != nil {
			return c.Status(fiber.StatusNotFound).Redirect("/")
		}

		if org.IsPrivate || dataset.IsPrivate {
			if c.Locals("AuthUser") == nil {
				return c.Status(fiber.StatusUnauthorized).Redirect("/login?Redirect=" + c.Path())
			}

			user := c.Locals("AuthUser").(*model.User)
			if !user.CanReadDataset(dataset) {
				return c.Status(fiber.StatusForbidden).Redirect("/" + org.Slug)
			}
		}

		addState(c, "Dataset", &dataset)

		return c.Next()
	}
}
func Dataset(c *fiber.Ctx) error {
	return c.Render("views/dataset", bind(c, fiber.Map{
		"AuthUser": c.Locals("AuthUser"),
	}), "views/layouts/main")
}
