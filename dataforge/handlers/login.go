package handlers

import (
	"fmt"
	"time"

	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type LoginForm struct {
	App
	Err error

	authUser *models.User
}

func (form *LoginForm) Pre(c *fiber.Ctx) error {
	form.authUser, _ = c.Locals("AuthUser").(*models.User)
	form.Err, _ = c.Locals("Error").(error)

	return nil
}

func (form *LoginForm) RenderHtml(c *fiber.Ctx) error {
	if form.authUser != nil {
		return c.Redirect("/")
	}

	bindings := Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Error":    form.Err,
	})

	if c.Query("Redirect") != "" {
		bindings["Redirect"] = c.Query("Redirect")
	}

	return c.Render("login", bindings, "layouts/main")
}

type Login struct {
	App

	fields struct {
		Redirect string
	}
	userLogin models.UserLogin
	token     string
}

func (page *Login) Pre(c *fiber.Ctx) error {
	_ = c.BodyParser(&page.fields)

	if err := c.BodyParser(&page.userLogin); err != nil {
		return fiber.ErrBadRequest
	}

	return nil
}

func (page *Login) Run() error {
	err := page.DB().ValidateCredentials(page.userLogin.Email, page.userLogin.Password)
	if err != nil {
		return fmt.Errorf("%w: invalid login credentials", fiber.ErrUnauthorized)
	}

	user := &models.User{Email: page.userLogin.Email}
	err = page.DB().Select("id", "email", "name", "orcid").Where(user).First(user).Error
	if err != nil {
		return fmt.Errorf("%w: an unexpected error occurred", fiber.ErrInternalServerError)
	}

	page.token, err = page.Auth().MakeToken(user)
	if err != nil {
		return fmt.Errorf("%w: an unexpected error occurred", fiber.ErrInternalServerError)
	}

	return nil
}

func (page *Login) Post(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:    "jwt",
		Value:   page.token,
		Expires: time.Now().Add(72 * time.Hour),
	})

	return nil
}

func (page *Login) Recover(c *fiber.Ctx, err error) error {
	return RecoverForm(&LoginForm{
		App: page.App,
	}, c, err)
}

func (page *Login) RenderHtml(c *fiber.Ctx) error {
	if page.fields.Redirect == "" {
		return c.Redirect("/")
	} else {
		return c.Redirect(page.fields.Redirect)
	}
}

func (*Login) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (*Login) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
