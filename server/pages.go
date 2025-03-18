package server

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

type Endpoint any

type PreprocessedEndpoint interface {
	Endpoint
	Preprocess(c *fiber.Ctx) error
}

type RunnableEndpoint interface {
	Endpoint
	Run(d *Server) error
}

type PostprocessedEndpoint interface {
	Endpoint
	Postprocess(c *fiber.Ctx) error
}

type HandledEndpoint interface {
	Endpoint
	HandleError(d *Server, c *fiber.Ctx, err error) error
}

type HtmlEndpoint interface {
	Endpoint
	RenderHtml(c *fiber.Ctx) error
}

type JsonEndpoint interface {
	Endpoint
	RenderJson(c *fiber.Ctx) error
}

type TextEndpoint interface {
	Endpoint
	RenderText(c *fiber.Ctx) error
}

type RedirectError struct {
	error
	Path string
	Back bool
}

func (err *RedirectError) Unwrap() error {
	return err.error
}

type ErrorHandler struct {
	err error
}

func (handler *ErrorHandler) Preprocess(c *fiber.Ctx) error {
	c.Locals("Error", handler.err.Error())

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
	var err *RedirectError
	if errors.As(handler.err, &err) {
		if err.Back {
			return c.RedirectBack(err.Path)
		}
		return c.Redirect(err.Path)
	}

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

func (d *Server) RenderEndpoint(p Endpoint) fiber.Handler {
	preprocessed, is_preprocessed := p.(PreprocessedEndpoint)
	runnable, is_runnable := p.(RunnableEndpoint)
	postprocessed, is_postprocessed := p.(PostprocessedEndpoint)
	handle, is_handled := p.(HandledEndpoint)

	html, html_ok := p.(HtmlEndpoint)
	json, json_ok := p.(JsonEndpoint)
	text, text_ok := p.(TextEndpoint)

	return func(c *fiber.Ctx) error {
		if is_preprocessed {
			if err := preprocessed.Preprocess(c); err != nil {
				if is_handled {
					return handle.HandleError(d, c, err)
				} else {
					return d.RenderEndpoint(&ErrorHandler{err})(c)
				}
			}
		}

		if is_runnable {
			if err := runnable.Run(d); err != nil {
				if is_handled {
					return handle.HandleError(d, c, err)
				} else {
					return d.RenderEndpoint(&ErrorHandler{err})(c)
				}
			}
		}

		if is_postprocessed {
			if err := postprocessed.Postprocess(c); err != nil {
				if is_handled {
					return handle.HandleError(d, c, err)
				} else {
					return d.RenderEndpoint(&ErrorHandler{err})(c)
				}
			}
		}

		if html_ok && c.Accepts("text/html") != "" {
			return html.RenderHtml(c)
		}

		if json_ok && c.Accepts("application/json") != "" {
			return json.RenderJson(c)
		}

		if text_ok && c.Accepts("text/plain") != "" {
			return text.RenderText(c)
		}

		if html_ok {
			return html.RenderHtml(c)
		} else {
			return c.SendStatus(fiber.StatusNoContent)
		}
	}
}
