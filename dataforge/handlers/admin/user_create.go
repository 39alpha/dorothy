package admin

import (
	"errors"
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/mail"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type UserCreateForm struct {
	handlers.ErrorHandler

	authUser *models.User
}

func (form *UserCreateForm) Pre(c *fiber.Ctx) error {
	form.authUser, _ = c.Locals("AuthUser").(*models.User)
	if form.authUser == nil {
		form.Err = fiber.ErrUnauthorized
		return form.Err
	} else if !form.authUser.HasAdminRole() {
		form.Err = fiber.ErrForbidden
		return form.Err
	}

	return nil
}

func (form *UserCreateForm) Recover(c *fiber.Ctx, err error) error {
	var e *fiber.Error

	if errors.As(err, &e) && e == fiber.ErrUnauthorized && handlers.Redirectable(c) {
		return c.Status(e.Code).Redirect("/login?Redirect=" + c.Path())
	}

	return form.ErrorFallback(c, err)
}

func (form *UserCreateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("admin/user-create", handlers.Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"Error":    form.Err,
	}), "layouts/main")
}

type UserCreate struct {
	handlers.ErrorHandler

	title   string
	baseUrl string

	authUser *models.User
	create   models.CreateUser
}

func (page *UserCreate) Pre(c *fiber.Ctx) error {
	page.authUser, _ = c.Locals("AuthUser").(*models.User)
	if page.authUser == nil {
		return fiber.ErrUnauthorized
	} else if !page.authUser.HasAdminRole() {
		return fiber.ErrForbidden
	}

	if err := c.BodyParser(&page.create); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	state := c.Locals("State").(fiber.Map)
	page.title = state["Title"].(string)
	page.baseUrl = state["BaseUrl"].(string)

	return nil
}

func (page *UserCreate) Run() error {
	token, err := page.DB().InviteUser(page.create)
	if err != nil {
		fmt.Println(err)
		return handlers.GormToFiber(err)
	}

	resetUrl := fmt.Sprintf("%s/reset-password?invitation=true&%s", page.baseUrl, token)

	err = page.Mailer().Send(mail.Message{
		To:           page.create.Email,
		Subject:      fmt.Sprintf("Invitation to %s", page.title),
		HtmlTemplate: "emails/invitation.html",
		TextTemplate: "emails/invitation.txt",
		Data: map[string]any{
			"Title":    page.title,
			"ResetUrl": resetUrl,
		},
	})
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (page *UserCreate) Recover(c *fiber.Ctx, err error) error {
	return page.HandleFormError(&UserCreateForm{}, c, err)
}

func (page *UserCreate) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/admin/users")
}

func (page *UserCreate) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (*UserCreate) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
