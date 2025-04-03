package handlers

import (
	"errors"
	"net/url"

	"github.com/39alpha/dorothy/core"
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

func RecoverForm(form Handler, c *fiber.Ctx, err error) error {
	c.Locals("Error", err)
	return ToFiberHandler(form)(c)
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	c.Locals("Error", err)

	page := "error"

	var e *fiber.Error
	if errors.As(err, &e) {
		c.Status(e.Code)

		switch e {
		case fiber.ErrUnauthorized:
			if Redirectable(c) {
				return c.Redirect("/login?Redirect=" + url.QueryEscape(c.Path()))
			}
		case fiber.ErrForbidden:
			page = "403"
		case fiber.ErrNotFound:
			page = "404"
		}
	} else {
		c.Status(fiber.StatusInternalServerError)
	}

	if AcceptsHtml(c) {
		return c.Render(page, Bind(c), "layouts/main")
	} else if AcceptsJson(c) {
		return c.JSON(fiber.Map{
			"error": err,
		})
	} else if AcceptsText(c) {
		return c.SendString(err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}
