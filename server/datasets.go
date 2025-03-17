package server

import (
	"context"
	"fmt"
	"time"

	"github.com/39alpha/dorothy/sdk"
	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
	"github.com/libp2p/go-libp2p/core/peer"
)

func GetDataset(identity peer.ID) fiber.Handler {
	return func(c *fiber.Ctx) error {
		dataset, ok := c.Locals("Dataset").(*models.Dataset)
		if dataset == nil || !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "cannot find dataset",
			})
		}

		if c.Accepts("text/html") != "" {
			canRead := false
			canWrite := false
			canManage := false

			authUser, ok := c.Locals("AuthUser").(*models.User)
			if ok {
				canRead = authUser.CanReadDataset(*dataset)
				canWrite = authUser.CanManageDataset(*dataset)
				canManage = authUser.CanManageDataset(*dataset)
			}

			return c.Render("views/dataset", Bind(c, fiber.Map{
				"AuthUser":  authUser,
				"CanRead":   canRead,
				"CanWrite":  canWrite,
				"CanManage": canManage,
			}), "views/layouts/main")
		} else if c.Accepts("application/json") != "" {
			if dataset.Manifest == nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to fetch dataset manifest",
				})
			}
			return c.JSON(sdk.Payload{
				Hash:         dataset.ManifestHash,
				PeerIdentity: identity,
			})
		} else if c.Accepts("text/plain") != "" {
			msg := dataset.ManifestHash + "\n" + string(identity)
			return c.SendString(msg)
		}
		return c.Render("views/dataset", Bind(c, fiber.Map{
			"AuthUser": c.Locals("AuthUser"),
		}), "views/layouts/main")
	}
}

func CreateDatasetForm(c *fiber.Ctx) error {
	authUser := c.Locals("AuthUser")
	team, ok := c.Locals("Team").(*models.Team)
	if !ok || team == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	if authUser == nil {
		return c.Redirect("/login?Redirect=" + c.Path())
	} else {
		return c.Render("views/create-dataset", Bind(c, fiber.Map{
			"AuthUser": c.Locals("AuthUser"),
			"Error":    c.Locals("Error"),
		}), "views/layouts/main")
	}
}

func DatasetSettingsForm(c *fiber.Ctx) error {
	team, ok := c.Locals("Team").(*models.Team)
	if !ok || team == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	dataset, ok := c.Locals("Dataset").(*models.Dataset)
	if !ok || dataset == nil || dataset.Manifest == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	user, ok := c.Locals("AuthUser").(*models.User)
	if !ok {
		return Redirect(c, fiber.StatusUnauthorized, "/login?Redirect="+c.Path(), fiber.Map{
			"error": "unauthorized",
		}, "unauthorized")
	} else if !user.CanManageDataset(*dataset) {
		return Redirect(c, fiber.StatusForbidden, "/"+team.Slug+"/"+dataset.Slug, fiber.Map{
			"error": "forbidden",
		}, "forbidden")
	}

	return c.Render("views/dataset-settings", Bind(c, fiber.Map{
		"AuthUser": user,
		"Error":    c.Locals("Error"),
	}), "views/layouts/main")
}

func (d *Server) LoadDataset(c *fiber.Ctx) error {
	team, ok := c.Locals("Team").(*models.Team)
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

	dataset := models.Dataset{
		Slug:   datasetSlug,
		TeamID: team.ID,
	}
	if err := d.session.Preload("Team").Where(&dataset).First(&dataset).Error; err != nil {
		return Redirect(c, fiber.StatusNotFound, "/"+team.Slug, fiber.Map{
			"error": "not found",
		}, "not found")
	}

	if team.IsPrivate || dataset.IsPrivate {
		user, ok := c.Locals("AuthUser").(*models.User)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var err error
	dataset.Manifest, err = d.Ipfs.GetManifest(ctx, dataset.ManifestHash)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get dataset manifest",
		})
	}

	AddState(c, "Dataset", &dataset)

	return c.Next()
}

func (d *Server) CreateDataset(c *fiber.Ctx) error {
	authUser, ok := c.Locals("AuthUser").(*models.User)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).Redirect("/")
	}

	team, ok := c.Locals("Team").(*models.Team)
	if !ok || team == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	var dataset models.NewDataset
	if err := c.BodyParser(&dataset); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
	}

	if team.ID != dataset.TeamID {
		return c.Redirect("/"+team.Slug+"/dataset/create", 400)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manifest, err := d.Ipfs.CreateEmptyManifest(ctx)
	if err != nil {
		c.Locals("Error", "We are having issues with IPFS at the moment. Try again later.")
		return nil
	}

	if err := d.session.CreateDataset(dataset, manifest, authUser); err != nil {
		c.Locals("Error", team.Name+" already has a dataset with slug \""+dataset.Slug+"\". Try a different name.")
		return CreateDatasetForm(c)
	}

	return c.Redirect("/" + team.Slug + "/" + dataset.Slug)
}

func (d *Server) RecieveDataset(c *fiber.Ctx) error {
	team, ok := c.Locals("Team").(*models.Team)
	if !ok || team == nil {
		return Redirect(c, fiber.StatusNotFound, "/", fiber.Map{
			"error": "not found",
		}, "not found")
	}

	dataset, ok := c.Locals("Dataset").(*models.Dataset)
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

	user, ok := c.Locals("AuthUser").(*models.User)
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

	AddState(c, "Dataset", dataset)

	return c.JSON(sdk.Payload{
		Hash:         dataset.ManifestHash,
		PeerIdentity: d.Ipfs.Identity,
	})
}

func (d *Server) UpdateDatasetSettings(c *fiber.Ctx) error {
	team, ok := c.Locals("Team").(*models.Team)
	if !ok || team == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	dataset, ok := c.Locals("Dataset").(*models.Dataset)
	if !ok || dataset == nil || dataset.Manifest == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	if team.ID != dataset.TeamID {
		return c.Redirect("/"+team.Slug+"/dataset/create", 400)
	}

	user, ok := c.Locals("AuthUser").(*models.User)
	if !ok {
		return Redirect(c, fiber.StatusUnauthorized, "/login?Redirect="+c.Path(), fiber.Map{
			"error": "unauthorized",
		}, "unauthorized")
	} else if !user.CanManageDataset(*dataset) {
		return Redirect(c, fiber.StatusForbidden, "/"+team.Slug+"/"+dataset.Slug, fiber.Map{
			"error": "forbidden",
		}, "forbidden")
	}

	var updated models.UpdateDataset
	if err := c.BodyParser(&updated); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
	}

	if team.ID != updated.TeamID || team.ID != dataset.TeamID || dataset.ID != updated.ID {
		return c.Redirect("/"+team.Slug+"/"+dataset.Slug+"/settings", 400)
	}

	if err := d.session.UpdateDataset(updated); err != nil {
		c.Locals("Error", team.Name+" already has a dataset with slug \""+updated.Slug+"\". Try a different name.")
		return DatasetSettingsForm(c)
	}

	return c.Redirect("/" + team.Slug + "/" + updated.Slug)
}

func (d *Server) DeleteDataset(c *fiber.Ctx) error {
	team, ok := c.Locals("Team").(*models.Team)
	if !ok || team == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	dataset, ok := c.Locals("Dataset").(*models.Dataset)
	if !ok || dataset == nil || dataset.Manifest == nil {
		return c.Status(fiber.StatusNotFound).Redirect("/")
	}

	user, ok := c.Locals("AuthUser").(*models.User)
	if !ok {
		return Respond(c, fiber.StatusUnauthorized, fiber.Map{
			"error": "unauthorized",
		}, "unauthorized")
	} else if !user.CanManageDataset(*dataset) {
		return Respond(c, fiber.StatusForbidden, fiber.Map{
			"error": "forbidden",
		}, "forbidden")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := d.session.DeleteDataset(dataset); err != nil {
		message := "We couldn't delete the dataset for some reason. Try again later?"
		return Respond(c, fiber.StatusInternalServerError, fiber.Map{
			"error": message,
		}, message)
	}

	if err := d.Ipfs.UnpinManifest(ctx, dataset.Manifest, true); err != nil {
		message := "We couldn't delete the dataset for some reason. Try again later?"
		return Respond(c, fiber.StatusInternalServerError, fiber.Map{
			"error": message,
		}, message)
	}

	return Respond(c, fiber.StatusOK, fiber.Map{
		"message": "success",
	}, "success")
}
