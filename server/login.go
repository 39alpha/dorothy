package server

import (
	"fmt"
	"time"

	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
)

type LoginForm struct {
	authUser *models.User
	err      error
}

func (form *LoginForm) Preprocess(d *Server, c *fiber.Ctx) error {
	form.authUser, _ = c.Locals("AuthUser").(*models.User)
	form.err, _ = c.Locals("Error").(error)

	return nil
}

func (form *LoginForm) RenderHtml(c *fiber.Ctx) error {
	if form.authUser != nil {
		return c.Redirect("/")
	}

	bindings := Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Error":    form.err,
	})

	if c.Query("Redirect") != "" {
		bindings["Redirect"] = c.Query("Redirect")
	}

	return c.Render("login", bindings, "layouts/main")
}

type Login struct {
	fields struct {
		Redirect string
	}
	userLogin models.UserLogin
	token     string
}

func (page *Login) Preprocess(d *Server, c *fiber.Ctx) error {
	_ = c.BodyParser(&page.fields)

	if err := c.BodyParser(&page.userLogin); err != nil {
		return fiber.ErrBadRequest
	}

	return nil
}

func (page *Login) Run(d *Server) error {
	if err := d.db.ValidateCredentials(page.userLogin.Email, page.userLogin.Password); err != nil {
		return fmt.Errorf("%w: invalid login credentials", fiber.ErrUnauthorized)
	}

	user := &models.User{Email: page.userLogin.Email}
	err := d.db.Select("id", "email", "name", "orcid").Where(user).First(user).Error
	if err != nil {
		return fmt.Errorf("%w: an unexpected error occurred", fiber.ErrInternalServerError)
	}

	page.token, err = d.auth.MakeToken(user)
	if err != nil {
		return fmt.Errorf("%w: an unexpected error occurred", fiber.ErrInternalServerError)
	}

	return nil
}

func (page *Login) Postprocess(d *Server, c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:    "jwt",
		Value:   page.token,
		Expires: time.Now().Add(72 * time.Hour),
	})

	return nil
}

func (page *Login) HandleError(d *Server, c *fiber.Ctx, err error) error {
	return d.HandleFormError(&LoginForm{}, c, err)
}

func (page *Login) RenderHtml(c *fiber.Ctx) error {
	if page.fields.Redirect == "" {
		return c.Redirect("/")
	} else {
		return c.Redirect(page.fields.Redirect)
	}
}

func (page *Login) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (page *Login) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
