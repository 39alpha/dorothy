package handlers

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type RegisterForm struct {
	App
}

func (form *RegisterForm) RenderHtml(c *fiber.Ctx) error {
	if GetAuthUser(c) != nil {
		return c.Redirect("/")
	}

	return c.Render("register", Bind(c), "layouts/main")
}

type Register struct {
	App

	newUser models.NewUser
}

func (page *Register) Pre(c *fiber.Ctx) error {
	if err := c.BodyParser(&page.newUser); err != nil {
		return fiber.ErrBadRequest
	}
	return nil
}

func (page *Register) Run() error {
	if err := page.DB().NewUser(&page.newUser); err != nil {
		return fmt.Errorf("%w: User already exists", fiber.ErrBadRequest)
	}
	return nil
}

func (page *Register) Recover(c *fiber.Ctx, err error) error {
	return RecoverForm(&RegisterForm{
		App: page.App,
	}, c, err)
}

func (*Register) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/login")
}

func (*Register) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}
