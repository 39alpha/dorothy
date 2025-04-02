package handlers

import (
	"context"
	"maps"
	"reflect"
	"slices"
	"time"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/dataforge/auth"
	"github.com/39alpha/dorothy/dataforge/db"
	"github.com/39alpha/dorothy/dataforge/mail"
	"github.com/39alpha/dorothy/dataforge/models"
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

type Handler interface {
	Run(c *fiber.Ctx) error
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

func MakeHandler(handler Handler) Handler {
	t := reflect.TypeOf(handler)
	var ptr reflect.Value
	if t.Kind() == reflect.Pointer {
		ptr = reflect.New(t.Elem())
	} else {
		ptr = reflect.New(t)
	}
	return ptr.Interface().(Handler)
}

func PerRequest(handler Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return ToFiberHandler(MakeHandler(handler))(c)
	}
}

func ToFiberHandler(page Handler) fiber.Handler {
	recover, is_recover := page.(RecoverHandler)

	html, html_ok := page.(HtmlHandler)
	json, json_ok := page.(JsonHandler)
	text, text_ok := page.(TextHandler)

	return func(c *fiber.Ctx) error {
		if err := page.Run(c); err != nil {
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

func Get[T any](c *fiber.Ctx, name string) T {
	entity, _ := c.Locals(name).(T)
	return entity
}

func GetAuthUser(c *fiber.Ctx) *models.User {
	return Get[*models.User](c, "AuthUser")
}

func GetDB(c *fiber.Ctx) *db.DB {
	return Get[*db.DB](c, "Database")
}

func GetDorothy(c *fiber.Ctx) *core.Dorothy {
	return Get[*core.Dorothy](c, "Dorothy")
}

func GetIpfs(c *fiber.Ctx) *core.Ipfs {
	dorothy := GetDorothy(c)
	if dorothy == nil {
		return nil
	}
	return &dorothy.Ipfs
}

func GetContext(c *fiber.Ctx, timeout ...time.Duration) (context.Context, context.CancelFunc) {
	var ctx context.Context = context.Background()

	dorothy := GetDorothy(c)
	if dorothy != nil {
		ctx = dorothy
	}

	if len(timeout) > 0 {
		return context.WithTimeout(ctx, timeout[0])
	}
	return context.WithCancel(ctx)
}

func GetAuth(c *fiber.Ctx) *auth.Auth {
	return Get[*auth.Auth](c, "Auth")
}

func GetMailer(c *fiber.Ctx) *mail.Mailer {
	return Get[*mail.Mailer](c, "Mailer")
}

func GetError(c *fiber.Ctx) error {
	return Get[error](c, "Error")
}

func Bind(c *fiber.Ctx, local ...fiber.Map) fiber.Map {
	bind := fiber.Map{
		"AuthUser": GetAuthUser(c),
		"Error":    GetError(c),
	}

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

func RequireLogin(c *fiber.Ctx) error {
	if GetAuthUser(c) == nil {
		return fiber.ErrUnauthorized
	}
	return nil
}
