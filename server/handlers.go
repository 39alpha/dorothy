package server

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/39alpha/dorothy/sdk"
	"github.com/39alpha/dorothy/server/model"
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

func Redirect(c *fiber.Ctx, status int, path string, json any, text string) error {
	if c.Accepts("text/html") != "" {
		return c.Status(status).Redirect(path)
	} else if c.Accepts("application/json") != "" {
		return c.Status(status).JSON(json)
	} else if c.Accepts("text/plain") != "" {
		return c.Status(status).SendString(text)
	}

	return c.Status(status).Redirect(path)
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

func Registration(db *DatabaseSession) fiber.Handler {
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

func Login(jwtAuth *Auth, db *DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var fields struct {
			Redirect string
		}
		c.BodyParser(&fields)

		var login model.UserLogin
		if err := c.BodyParser(&login); err != nil {
			return Redirect(c, fiber.StatusBadRequest, "/login", fiber.Map{
				"error": "bad request",
			}, "bad request")
		}

		if err := db.ValidateCredentials(login.Email, login.Password); err != nil {
			return Redirect(c, fiber.StatusUnauthorized, "/login", fiber.Map{
				"error": "invalid login credentials",
			}, "invalid logic credentials")
		}

		user := &model.User{Email: login.Email}
		err := db.Select("id", "email", "name", "orcid").Where(user).First(user).Error
		if err != nil {
			return Redirect(c, fiber.StatusInternalServerError, "/login", fiber.Map{
				"error": "an unexpected error occurred",
			}, "an unexpected error occurred")
		}

		token, err := jwtAuth.MakeToken(user)
		if err != nil {
			return Redirect(c, fiber.StatusInternalServerError, "/login", fiber.Map{
				"error": "an unexpected error occurred",
			}, "an unexpected error occurred")
		}

		c.Cookie(&fiber.Cookie{
			Name:    "jwt",
			Value:   token,
			Expires: time.Now().Add(72 * time.Hour),
		})

		if c.Accepts("text/html") == "" {
			if c.Accepts("application/json") != "" {
				return c.JSON(fiber.Map{
					"message": "success",
				})
			} else if c.Accepts("text/plain") != "" {
				return c.SendString("success")
			}
		}

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

func GetOrganizations(db *DatabaseSession) fiber.Handler {
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

func CreateOrganization(db *DatabaseSession) fiber.Handler {
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

func GetOrganization(db *DatabaseSession) fiber.Handler {
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

func (d *Server) CreateDatasetHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authUser, ok := c.Locals("AuthUser").(*model.User)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).Redirect("/")
		}

		org, ok := c.Locals("Organization").(*model.Organization)
		if !ok || org == nil {
			return c.Status(fiber.StatusNotFound).Redirect("/")
		}

		var dataset model.NewDataset
		if err := c.BodyParser(&dataset); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
		}

		if org.ID != dataset.OrganizationID {
			return c.Redirect("/"+org.Slug+"/dataset/create", 400)
		}

		if err := d.CreateDataset(dataset, authUser); err != nil {
			return c.Status(fiber.StatusInternalServerError).Redirect("/")
		}

		return c.Redirect("/" + org.Slug + "/" + dataset.Slug)
	}
}

func (d *Server) GetDataset() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		org, ok := c.Locals("Organization").(*model.Organization)
		if !ok || org == nil {
			return Redirect(c, fiber.StatusNotFound, "/", fiber.Map{
				"error": "not found",
			}, "not found")
		}

		datasetSlug := c.Params("dataset")
		if datasetSlug == "" {
			return Redirect(c, fiber.StatusNotFound, "/"+org.Slug, fiber.Map{
				"error": "bad request",
			}, "bad request")
		}

		dataset := model.Dataset{
			Slug:           datasetSlug,
			OrganizationID: org.ID,
		}
		if err := d.session.Preload("Organization").Where(&dataset).First(&dataset).Error; err != nil {
			return Redirect(c, fiber.StatusNotFound, "/"+org.Slug, fiber.Map{
				"error": "not found",
			}, "not found")
		}

		if org.IsPrivate || dataset.IsPrivate {
			user, ok := c.Locals("AuthUser").(*model.User)
			if !ok {
				return Redirect(c, fiber.StatusUnauthorized, "/login?Redirect="+c.Path(), fiber.Map{
					"error": "unauthorized",
				}, "unauthorized")
			} else if !user.CanReadDataset(dataset) {
				return Redirect(c, fiber.StatusForbidden, "/"+org.Slug, fiber.Map{
					"error": "forbidden",
				}, "forbidden")
			}
		}

		var err error
		dataset.Manifest, err = d.Ipfs.GetManifest(ctx, dataset.ManifestHash)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get dataset manifest",
			})
		}

		addState(c, "Dataset", &dataset)

		return c.Next()
	}
}

func (d *Server) RecieveDataset() fiber.Handler {
	return func(c *fiber.Ctx) error {
		dataset, ok := c.Locals("Dataset").(*model.Dataset)
		if !ok || dataset == nil || dataset.Manifest == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to fetch dataset manifest",
			})
		}
		old := dataset.Manifest

		var payload sdk.Payload
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "recieved invalid manifest",
			})
		}

		ctx, cancel := context.WithTimeout(d, 10*time.Second)
		defer cancel()

		err := d.Ipfs.ConnectToPeerById(ctx, payload.PeerIdentity)
		if err != nil {
			if ctx.Err() == nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to connect to ipfs peer",
				})
			} else {
				return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
					"message": "attempt to connect to ipfs peer timed out",
				})
			}
		}

		manifest, conflicts, err := d.Recieve(old, payload.Hash)
		if len(conflicts) != 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"conflicts": conflicts,
				"error":     "merge failed with conflicts",
			})
		} else if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("merge failed: %v", err),
			})
		}

		dataset.ManifestHash = manifest.Hash
		if err := d.session.Save(dataset).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to save manifest",
			})
		}

		addState(c, "Dataset", dataset)

		return c.JSON(sdk.Payload{
			Hash:         dataset.ManifestHash,
			PeerIdentity: d.Ipfs.Identity,
		})
	}
}

func (d *Server) Dataset() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Accepts("text/html") != "" {
			return c.Render("views/dataset", bind(c, fiber.Map{
				"AuthUser": c.Locals("AuthUser"),
			}), "views/layouts/main")
		} else if c.Accepts("application/json") != "" {
			dataset, ok := c.Locals("Dataset").(*model.Dataset)
			if !ok || dataset == nil || dataset.Manifest == nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to fetch dataset manifest",
				})
			}
			return c.JSON(sdk.Payload{
				Hash:         dataset.ManifestHash,
				PeerIdentity: d.Ipfs.Identity,
			})
		} else if c.Accepts("text/plain") != "" {
			if dataset, ok := c.Locals("Dataset").(*model.Dataset); ok {
				msg := dataset.ManifestHash + "\n" + string(d.Ipfs.Identity)
				return c.SendString(msg)
			}
		}
		return c.Render("views/dataset", bind(c, fiber.Map{
			"AuthUser": c.Locals("AuthUser"),
		}), "views/layouts/main")
	}
}
