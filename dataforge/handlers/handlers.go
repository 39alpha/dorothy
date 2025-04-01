package handlers

import (
	"maps"
	"reflect"
	"slices"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/dataforge/auth"
	"github.com/39alpha/dorothy/dataforge/db"
	"github.com/39alpha/dorothy/dataforge/mail"
	"github.com/gofiber/fiber/v2"
)

var disallowedNames []string

func init() {
	// DGM: This must be sorted lexicographically
	disallowedNames = []string{
		"availability",
		"available",
		"change-password",
		"change-passwords",
		"create",
		"creates",
		"dataset",
		"datasets",
		"login",
		"logout",
		"privilege",
		"privileges",
		"profile",
		"profiles",
		"register",
		"search",
		"searches",
		"settings",
		"team",
		"teams",
		"user",
		"users",
	}
}

func IsDisallowedName(name string) bool {
	_, ok := slices.BinarySearch(disallowedNames, name)
	return ok
}

type App interface {
	Dorothy() *core.Dorothy
	Auth() *auth.Auth
	DB() *db.DB
	Mailer() *mail.Mailer
}

type Handler interface {
	App
}

type PreHandler interface {
	Handler
	Pre(c *fiber.Ctx) error
}

type RunHandler interface {
	Handler
	Run() error
}

type PostHandler interface {
	Handler
	Post(c *fiber.Ctx) error
}

type RecoverHandler interface {
	Handler
	Recover(c *fiber.Ctx, err error) error
}

type HtmlHandler interface {
	Handler
	RenderHtml(c *fiber.Ctx) error
}

type JsonHandler interface {
	Handler
	RenderJson(c *fiber.Ctx) error
}

type TextHandler interface {
	Handler
	RenderText(c *fiber.Ctx) error
}

func MakeHandler(handler Handler, app App) Handler {
	t := reflect.TypeOf(handler)
	var ptr reflect.Value
	if t.Kind() == reflect.Pointer {
		ptr = reflect.New(t.Elem())
	} else {
		ptr = reflect.New(t)
	}
	field := ptr.Elem().FieldByName("App")
	field.Set(reflect.ValueOf(app))
	return ptr.Interface().(Handler)
}

func PerRequest(handler Handler, app App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return ToFiberHandler(MakeHandler(handler, app))(c)
	}
}

func ToFiberHandler(page Handler) fiber.Handler {
	pre, is_pre := page.(PreHandler)
	run, is_run := page.(RunHandler)
	post, is_post := page.(PostHandler)
	recover, is_recover := page.(RecoverHandler)

	html, html_ok := page.(HtmlHandler)
	json, json_ok := page.(JsonHandler)
	text, text_ok := page.(TextHandler)

	return func(c *fiber.Ctx) error {
		var err error

		if is_pre {
			err = pre.Pre(c)
		}

		if err == nil && is_run {
			err = run.Run()
		}

		if err == nil && is_post {
			err = post.Post(c)
		}

		if err != nil {
			if is_recover {
				return recover.Recover(c, err)
			}
			return err
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

func Bind(c *fiber.Ctx, local ...fiber.Map) fiber.Map {
	bind := fiber.Map{}

	state, ok := c.Locals("State").(fiber.Map)
	if !ok || state != nil {
		maps.Copy(bind, state)
	}
	for _, l := range local {
		maps.Copy(bind, l)
	}

	return bind
}

func AcceptsHtml(c *fiber.Ctx) bool {
	return c.Accepts("text/html") != ""
}

func AcceptsJson(c *fiber.Ctx) bool {
	return c.Accepts("application/json") != ""
}

func AcceptsText(c *fiber.Ctx) bool {
	return c.Accepts("text/plain") != ""
}

func Redirectable(c *fiber.Ctx) bool {
	return AcceptsHtml(c) || (!AcceptsJson(c) && !AcceptsText(c))
}
