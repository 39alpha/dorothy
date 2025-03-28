package handlers

import (
	"errors"

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

type ErrorHandler struct {
	App

	Err error
}

func (handler *ErrorHandler) Pre(c *fiber.Ctx) error {
	c.Locals("Error", handler.Err)

	var e *fiber.Error
	if errors.As(handler.Err, &e) {
		c.Status(e.Code)
	} else {
		c.Status(fiber.StatusInternalServerError)
	}

	return nil
}

func (handler *ErrorHandler) RenderHtml(c *fiber.Ctx) error {
	var err *fiber.Error
	if errors.As(handler.Err, &err) {
		switch err {
		case fiber.ErrNotFound:
			return c.Render("404", Bind(c), "layouts/main")
		case fiber.ErrForbidden:
			return c.Render("403", Bind(c), "layouts/main")
		}
	}

	return c.Render("error", Bind(c, fiber.Map{
		"Error": handler.Err,
	}), "layouts/main")
}

func (handler *ErrorHandler) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"error": handler.Err.Error(),
	})
}

func (handler *ErrorHandler) RenderText(c *fiber.Ctx) error {
	return c.SendString(handler.Err.Error())
}

func (h *ErrorHandler) ErrorFallback(c *fiber.Ctx, err error) error {
	return ToFiberHandler(&ErrorHandler{App: h, Err: err})(c)
}

func (h *ErrorHandler) HandleFormError(page Handler, c *fiber.Ctx, err error) error {
	handler := &ErrorHandler{App: h, Err: err}
	_ = handler.Pre(c)

	if c.Accepts("text/html") != "" {
		return ToFiberHandler(page)(c)
	} else if c.Accepts("application/json") != "" || c.Accepts("text/plain") != "" {
		return h.ErrorFallback(c, err)
	}

	return ToFiberHandler(page)(c)
}
