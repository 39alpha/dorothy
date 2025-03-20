package server

import (
	"errors"

	"github.com/39alpha/dorothy/core"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type MergeConflict struct {
	error
	conflicts []core.Conflict
}

type ErrorHandler struct {
	err error
}

func GormToFiber(err error) error {
	if err == nil {
		return err
	}

	mapping := map[error]error{
		gorm.ErrRecordNotFound: fiber.ErrNotFound,
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

func (handler *ErrorHandler) Preprocess(d *Server, c *fiber.Ctx) error {
	c.Locals("Error", handler.err)

	var e *fiber.Error
	if errors.As(handler.err, &e) {
		c.Status(e.Code)
	} else {
		c.Status(fiber.StatusInternalServerError)
	}

	return nil
}

func (handler *ErrorHandler) HandleError(d *Server, c *fiber.Ctx, err error) error {
	c.Status(fiber.StatusInternalServerError)
	if c.Accepts("application/json") != "" {
		return c.JSON(fiber.Map{
			"error": err.Error(),
		})
	} else if c.Accepts("text/plain") != "" {
		return c.SendString(err.Error())
	}

	return c.SendStatus(fiber.StatusInternalServerError)
}

func (handler *ErrorHandler) RenderHtml(c *fiber.Ctx) error {
	return c.SendString(handler.err.Error())
}

func (handler *ErrorHandler) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"error": handler.err.Error(),
	})
}

func (handler *ErrorHandler) RenderText(c *fiber.Ctx) error {
	return c.SendString(handler.err.Error())
}

func (d *Server) ErrorFallback(c *fiber.Ctx, err error) error {
	return d.RenderEndpoint(&ErrorHandler{err})(c)
}

func (d *Server) HandleFormError(form Endpoint, c *fiber.Ctx, err error) error {
	handler := &ErrorHandler{err}
	_ = handler.Preprocess(d, c)

	if c.Accepts("text/html") != "" {
		return d.RenderEndpoint(form)(c)
	} else if c.Accepts("application/json") != "" || c.Accepts("text/plain") != "" {
		return d.RenderEndpoint(&ErrorHandler{err})(c)
	}

	return d.RenderEndpoint(form)(c)
}
