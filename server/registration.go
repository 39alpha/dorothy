package server

import (
	"fmt"

	"github.com/39alpha/dorothy/server/models"
	"github.com/gofiber/fiber/v2"
)

type RegistrationForm struct {
	authUser *models.User
	err      error
}

func (form *RegistrationForm) Preprocess(d *Server, c *fiber.Ctx) error {
	form.authUser, _ = c.Locals("AuthUser").(*models.User)
	form.err, _ = c.Locals("Error").(error)

	return nil
}

func (form *RegistrationForm) RenderHtml(c *fiber.Ctx) error {
	if form.authUser != nil {
		return c.Redirect("/")
	}

	return c.Render("views/register", Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Error":    form.err,
	}), "views/layouts/main")
}

type Registration struct {
	newUser models.NewUser
}

func (r *Registration) Preprocess(d *Server, c *fiber.Ctx) error {
	if err := c.BodyParser(&r.newUser); err != nil {
		return fiber.ErrBadRequest
	}
	return nil
}

func (r *Registration) Run(d *Server) error {
	if err := d.db.CreateUser(&r.newUser); err != nil {
		return fmt.Errorf("%w: User already exists", fiber.ErrBadRequest)
	}
	return nil
}

func (r *Registration) HandleError(d *Server, c *fiber.Ctx, err error) error {
	return d.HandleFormError(&RegistrationForm{}, c, err)
}

func (r *Registration) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/login")
}

func (r *Registration) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}
