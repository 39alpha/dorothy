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
	logger := GetLogger(c).Trace(c).Handler("Error").Send()

	c.Locals("Error", err)

	page := "error"
	var e *fiber.Error
	if errors.As(err, &e) {
		logger.Trace(c).Err(err).Bool("is_fiber", true).Int("code", e.Code).Msg("Handling Error")

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

		logger.Trace(c).Str("error_page", page).Msg("Error Page Selected")
	} else {
		logger.Trace(c).Err(err).Bool("is_fiber", false).Msg("Handling Error")

		c.Status(fiber.StatusInternalServerError)
	}

	if AcceptsHtml(c) {
		logger.Trace(c).Msg("Responding with HTML")
		return c.Render(page, Bind(c), "layouts/main")
	} else if AcceptsJson(c) {
		logger.Trace(c).Msg("Responding with JSON")
		return c.JSON(fiber.Map{
			"error":      err.Error(),
			"request_id": RequestID(c),
		})
	} else if AcceptsText(c) {
		logger.Trace(c).Msg("Responding with Text")
		return c.SendString(err.Error())
	}

	logger.Trace(c).Msg("Responding No Content")
	return c.SendStatus(fiber.StatusNoContent)
}
