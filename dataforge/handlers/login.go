package handlers

import (
	"fmt"
	"time"

	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type LoginForm struct {
	App

	authUser *models.User
}

func (form *LoginForm) Run(c *fiber.Ctx) error {
	return nil
}

func (form *LoginForm) RenderHtml(c *fiber.Ctx) error {
	authUser := GetAuthUser(c)

	if authUser != nil {
		return c.Redirect("/")
	}

	bindings := Bind(c)
	if c.Query("Redirect") != "" {
		bindings["Redirect"] = c.Query("Redirect")
	}

	return c.Render("login", bindings, "layouts/main")
}

type Login struct {
	App

	Redirect string
}

func (page *Login) Run(c *fiber.Ctx) error {
	auth := GetAuth(c)
	if auth == nil {
		return fmt.Errorf("%w: login isn't working right now", fiber.ErrInternalServerError)
	}

	var fields struct {
		Redirect string
	}
	_ = c.BodyParser(&fields)
	page.Redirect = fields.Redirect

	userLogin := models.UserLogin{}
	if err := c.BodyParser(&userLogin); err != nil {
		return fiber.ErrBadRequest
	}

	err := page.DB().ValidateCredentials(userLogin.Email, userLogin.Password)
	if err != nil {
		return fmt.Errorf("%w: invalid login credentials", fiber.ErrUnauthorized)
	}

	user := &models.User{Email: userLogin.Email}
	err = page.DB().Select("id", "email", "name", "orcid").Where(user).First(user).Error
	if err != nil {
		return fmt.Errorf("%w: an unexpected error occurred", fiber.ErrInternalServerError)
	}

	token, err := auth.MakeToken(user)
	if err != nil {
		return fmt.Errorf("%w: an unexpected error occurred", fiber.ErrInternalServerError)
	}

	c.Cookie(&fiber.Cookie{
		Name:    "jwt",
		Value:   token,
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
	if page.Redirect == "" {
		return c.Redirect("/")
	} else {
		return c.Redirect(page.Redirect)
	}
}

func (*Login) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (*Login) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
