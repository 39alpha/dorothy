package handlers

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type RegisterForm struct {
	App
	Err error

	authUser *models.User
}

func (form *RegisterForm) Pre(c *fiber.Ctx) error {
	form.authUser, _ = c.Locals("AuthUser").(*models.User)
	form.Err, _ = c.Locals("Error").(error)

	return nil
}

func (form *RegisterForm) RenderHtml(c *fiber.Ctx) error {
	if form.authUser != nil {
		return c.Redirect("/")
	}

	return c.Render("register", Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Error":    form.Err,
	}), "layouts/main")
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
