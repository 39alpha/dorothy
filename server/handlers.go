package server

import (
	"context"
	"fmt"
	"maps"
	"math"
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

func (d *Server) Index(c *fiber.Ctx) error {
	user, _ := c.Locals("AuthUser").(*model.User)

	teams, err := d.session.GetTeams(user, false)
	if err != nil {
		teams = []model.Team{}
	}
	teams = teams[0:int64(math.Min(float64(len(teams)), 6))]

	datasets, err := d.session.GetDatasets(user)
	if err != nil {
		datasets = []model.Dataset{}
	}
	datasets = datasets[0:int64(math.Min(float64(len(datasets)), 6))]

	return c.Render("views/index", bind(c, fiber.Map{
		"AuthUser": user,
		"Teams":    teams,
		"Datasets": datasets,
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

func GetTeams(db *DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, _ := c.Locals("AuthUser").(*model.User)

		teams, err := db.GetTeams(user, true)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).Redirect("/")
		}

		addState(c, "Teams", teams)

		return c.Next()
	}
}

func CreateTeamForm(c *fiber.Ctx) error {
	authUser := c.Locals("AuthUser")
	if authUser == nil {
		return c.Redirect("/login?Redirect=" + c.Path())
	} else {
		return c.Render("views/create-team", bind(c, fiber.Map{
			"AuthUser": c.Locals("AuthUser"),
			"Error":    c.Locals("Error"),
		}), "views/layouts/main")
	}
}

func CreateTeam(db *DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authUser, ok := c.Locals("AuthUser").(*model.User)

		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).Redirect("/")
		}

		var newteam model.NewTeam
		if err := c.BodyParser(&newteam); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
		}

		team := &model.Team{
			Slug:      newteam.Slug,
			Name:      newteam.Name,
			Contact:   newteam.Contact,
			IsPrivate: newteam.IsPrivate,
		}
		if newteam.Description != nil {
			team.Description = *newteam.Description
		}
		if err := db.Save(team).Error; err != nil {
			c.Locals("Error", "A team with slug \""+team.Slug+"\" already exists. Try a different name.")
			return CreateTeamForm(c)
		}

		db.Save(&model.UserTeamPrivilege{
			User:          authUser,
			Team:          team,
			PrivilegeCode: "admin",
		})

		return c.Redirect("/" + team.Slug)
	}
}

func GetTeam(db *DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		slug := c.Params("team")
		if slug == "" {
			return c.Status(fiber.StatusNotFound).Redirect("/")
		}

		team := model.Team{Slug: slug}
		if err := db.Preload("Datasets.Team").Where(&team).First(&team).Error; err != nil {
			return c.Status(fiber.StatusNotFound).Redirect("/")
		}

		datasets := []model.Dataset{}
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
			user := c.Locals("AuthUser").(*model.User)
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

		addState(c, "Team", &team)

		return c.Next()
	}
}

func Team(c *fiber.Ctx) error {
	return c.Render("views/team", bind(c, fiber.Map{
		"AuthUser": c.Locals("AuthUser"),
	}), "views/layouts/main")
}

func CreateDatasetForm(c *fiber.Ctx) error {
	authUser := c.Locals("AuthUser")
	team, ok := c.Locals("Team").(*model.Team)
	if !ok || team == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	if authUser == nil {
		return c.Redirect("/login?Redirect=" + c.Path())
	} else {
		return c.Render("views/create-dataset", bind(c, fiber.Map{
			"AuthUser": c.Locals("AuthUser"),
			"Error":    c.Locals("Error"),
		}), "views/layouts/main")
	}
}

func (d *Server) CreateDatasetHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authUser, ok := c.Locals("AuthUser").(*model.User)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).Redirect("/")
		}

		team, ok := c.Locals("Team").(*model.Team)
		if !ok || team == nil {
			return c.Status(fiber.StatusNotFound).Redirect("/")
		}

		var dataset model.NewDataset
		if err := c.BodyParser(&dataset); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
		}

		if team.ID != dataset.TeamID {
			return c.Redirect("/"+team.Slug+"/dataset/create", 400)
		}

		if err := d.CreateDataset(dataset, authUser); err != nil {
			c.Locals("Error", team.Name+" already has a dataset with slug \""+dataset.Slug+"\". Try a different name.")
			return CreateDatasetForm(c)
		}

		return c.Redirect("/" + team.Slug + "/" + dataset.Slug)
	}
}

func (d *Server) GetDataset() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		team, ok := c.Locals("Team").(*model.Team)
		if !ok || team == nil {
			return Redirect(c, fiber.StatusNotFound, "/", fiber.Map{
				"error": "not found",
			}, "not found")
		}

		datasetSlug := c.Params("dataset")
		if datasetSlug == "" {
			return Redirect(c, fiber.StatusNotFound, "/"+team.Slug, fiber.Map{
				"error": "bad request",
			}, "bad request")
		}

		dataset := model.Dataset{
			Slug:   datasetSlug,
			TeamID: team.ID,
		}
		if err := d.session.Preload("Team").Where(&dataset).First(&dataset).Error; err != nil {
			return Redirect(c, fiber.StatusNotFound, "/"+team.Slug, fiber.Map{
				"error": "not found",
			}, "not found")
		}

		if team.IsPrivate || dataset.IsPrivate {
			user, ok := c.Locals("AuthUser").(*model.User)
			if !ok {
				return Redirect(c, fiber.StatusUnauthorized, "/login?Redirect="+c.Path(), fiber.Map{
					"error": "unauthorized",
				}, "unauthorized")
			} else if !user.CanReadDataset(dataset) {
				return Redirect(c, fiber.StatusForbidden, "/"+team.Slug, fiber.Map{
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
		team, ok := c.Locals("Team").(*model.Team)
		if !ok || team == nil {
			return Redirect(c, fiber.StatusNotFound, "/", fiber.Map{
				"error": "not found",
			}, "not found")
		}

		dataset, ok := c.Locals("Dataset").(*model.Dataset)
		if !ok || dataset == nil || dataset.Manifest == nil {
			return Redirect(c, fiber.StatusNotFound, "/", fiber.Map{
				"error": "not found",
			}, "not found")
		}
		old := dataset.Manifest

		var payload sdk.Payload
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "recieved invalid manifest",
			})
		}

		user, ok := c.Locals("AuthUser").(*model.User)
		if !ok {
			return Redirect(c, fiber.StatusUnauthorized, "/login?Redirect="+c.Path(), fiber.Map{
				"error": "unauthorized",
			}, "unauthorized")
		} else if !user.CanWriteDataset(*dataset) {
			return Redirect(c, fiber.StatusForbidden, "/"+team.Slug+"/"+dataset.Slug, fiber.Map{
				"error": "forbidden",
			}, "forbidden")
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

func DatasetSettingsForm(c *fiber.Ctx) error {
	team, ok := c.Locals("Team").(*model.Team)
	if !ok || team == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	dataset, ok := c.Locals("Dataset").(*model.Dataset)
	if !ok || dataset == nil || dataset.Manifest == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	user, ok := c.Locals("AuthUser").(*model.User)
	if !ok {
		return Redirect(c, fiber.StatusUnauthorized, "/login?Redirect="+c.Path(), fiber.Map{
			"error": "unauthorized",
		}, "unauthorized")
	} else if !user.CanWriteDataset(*dataset) {
		return Redirect(c, fiber.StatusForbidden, "/"+team.Slug+"/"+dataset.Slug, fiber.Map{
			"error": "forbidden",
		}, "forbidden")
	}

	return c.Render("views/edit-dataset", bind(c, fiber.Map{
		"AuthUser": user,
		"Error":    c.Locals("Error"),
	}), "views/layouts/main")
}

func (d *Server) DatasetSettingsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authUser, ok := c.Locals("AuthUser").(*model.User)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).Redirect("/")
		}

		team, ok := c.Locals("Team").(*model.Team)
		if !ok || team == nil {
			return c.Status(fiber.StatusNotFound).Redirect("/")
		}

		dataset, ok := c.Locals("Dataset").(*model.Dataset)
		if !ok || dataset == nil || dataset.Manifest == nil {
			return c.Status(fiber.StatusNotFound).Redirect("/")
		}

		user, ok := c.Locals("AuthUser").(*model.User)
		if !ok {
			return Redirect(c, fiber.StatusUnauthorized, "/login?Redirect="+c.Path(), fiber.Map{
				"error": "unauthorized",
			}, "unauthorized")
		} else if !user.CanManageDataset(*dataset) {
			return Redirect(c, fiber.StatusForbidden, "/"+team.Slug+"/"+dataset.Slug, fiber.Map{
				"error": "forbidden",
			}, "forbidden")
		}

		var updated model.UpdateDataset
		if err := c.BodyParser(&updated); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
		}

		if team.ID != updated.TeamID || team.ID != dataset.TeamID || dataset.ID != updated.ID {
			return c.Redirect("/"+team.Slug+"/"+dataset.Slug+"/settings", 400)
		}

		if err := d.UpdateDataset(updated); err != nil {
			c.Locals("Error", team.Name+" already has a dataset with slug \""+dataset.Slug+"\". Try a different name.")
			return CreateDatasetForm(c)
		}

		return c.Redirect("/" + team.Slug + "/" + updated.Slug)
	}
}
