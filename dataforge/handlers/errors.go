package handlers

import (
	"errors"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func GormToFiber(err error) error {
	if err == nil {
		return err
	}

	mapping := map[error]error{
		gorm.ErrRecordNotFound: fiber.ErrNotFound,
		gorm.ErrInvalidValue:   fiber.ErrBadRequest,
	}

	for g, f := range mapping {
		if errors.Is(err, g) {
			return f
		}
	}

	if errors.Is(err, &fiber.Error{}) {
		return err
	}

	return fiber.ErrInternalServerError
}

type MergeConflict struct {
	error
	Conflicts []core.Conflict
}

func NewMergeConflict(err error, conflicts []core.Conflict) *MergeConflict {
	return &MergeConflict{err, conflicts}
}

func BestErrorCode(err error) int {
	var e *fiber.Error
	if errors.As(err, &e) {
		return e.Code
	} else {
		return fiber.StatusInternalServerError
	}
}

func RecoverLogin(c *fiber.Ctx, err error) error {
	var e *fiber.Error
	if errors.As(err, &e) && e == fiber.ErrUnauthorized && Redirectable(c) {
		return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
	}
	return err
}

func RecoverForm(form Handler, c *fiber.Ctx, err error) error {
	c.Locals("Error", err)
	return ToFiberHandler(form)(c)
}

func ErrorMiddleware(app App) fiber.Handler {
	handleError := func(c *fiber.Ctx, err error) error {
		page := "error"

		var e *fiber.Error
		if errors.As(err, &e) {
			c.Status(e.Code)
			switch e {
			case fiber.ErrNotFound:
				page = "404"
			case fiber.ErrForbidden:
				page = "405"
			}
		} else {
			c.Status(fiber.StatusInternalServerError)
		}

		if AcceptsHtml(c) {
			authUser, _ := c.Locals("AuthUser").(*models.User)

			return c.Render(page, Bind(c, fiber.Map{
				"Error":    err,
				"AuthUser": authUser,
			}), "layouts/main")
		} else if AcceptsJson(c) {
			return c.JSON(fiber.Map{
				"error": err,
			})
		} else if AcceptsText(c) {
			return c.SendString(err.Error())
		}

		return c.SendStatus(fiber.StatusNoContent)
	}

	return func(c *fiber.Ctx) error {
		var err error

		defer func() {
			if e := recover(); e != nil {
				err = handleError(c, e.(error))
			}
		}()

		err = c.Next()
		if err != nil {
			err = handleError(c, err)
		}

		return err
	}
}
