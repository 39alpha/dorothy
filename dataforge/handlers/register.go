package handlers

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type RegisterForm struct {
	App
}

func (form *RegisterForm) Run(c *fiber.Ctx) error {
	return nil
}

func (form *RegisterForm) RenderHtml(c *fiber.Ctx) error {
	if GetAuthUser(c) != nil {
		return c.Redirect("/")
	}

	return c.Render("register", Bind(c), "layouts/main")
}

type Register struct {
	App
}

func (page *Register) Run(c *fiber.Ctx) error {
	var newUser models.NewUser
	if err := c.BodyParser(&newUser); err != nil {
		return fiber.ErrBadRequest
	}

	if err := GetDB(c).NewUser(&newUser); err != nil {
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
