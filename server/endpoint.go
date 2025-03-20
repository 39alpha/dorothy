package server

import (
	"github.com/gofiber/fiber/v2"
)

type Endpoint any

type PreprocessedEndpoint interface {
	Endpoint
	Preprocess(d *Server, c *fiber.Ctx) error
}

type RunnableEndpoint interface {
	Endpoint
	Run(d *Server) error
}

type PostprocessedEndpoint interface {
	Endpoint
	Postprocess(d *Server, c *fiber.Ctx) error
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

type EndpointCtor func() Endpoint

func PerRequest[K Endpoint]() EndpointCtor {
	return func() Endpoint {
		return new(K)
	}
}

func (d *Server) RenderEndpointCtor(ctor EndpointCtor) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return d.RenderEndpoint(ctor())(c)
	}
}

func (d *Server) RenderEndpoint(page Endpoint) fiber.Handler {
	preprocessed, is_preprocessed := page.(PreprocessedEndpoint)
	runnable, is_runnable := page.(RunnableEndpoint)
	postprocessed, is_postprocessed := page.(PostprocessedEndpoint)
	handle, is_handled := page.(HandledEndpoint)

	html, html_ok := page.(HtmlEndpoint)
	json, json_ok := page.(JsonEndpoint)
	text, text_ok := page.(TextEndpoint)

	return func(c *fiber.Ctx) error {
		if is_preprocessed {
			if err := preprocessed.Preprocess(d, c); err != nil {
				if is_handled {
					return handle.HandleError(d, c, err)
				} else {
					return d.ErrorFallback(c, err)
				}
			}
		}

		if is_runnable {
			if err := runnable.Run(d); err != nil {
				if is_handled {
					return handle.HandleError(d, c, err)
				} else {
					return d.ErrorFallback(c, err)
				}
			}
		}

		if is_postprocessed {
			if err := postprocessed.Postprocess(d, c); err != nil {
				if is_handled {
					return handle.HandleError(d, c, err)
				} else {
					return d.ErrorFallback(c, err)
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

		return c.SendStatus(fiber.StatusNoContent)
	}
}
