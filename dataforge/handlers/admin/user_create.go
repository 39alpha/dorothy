package admin

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/mail"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type UserCreateForm struct {
	handlers.App
}

func (form *UserCreateForm) Pre(c *fiber.Ctx) error {
	return RequireAdmin(c)
}

func (form *UserCreateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("admin/user-create", handlers.Bind(c), "layouts/main")
}

type UserCreate struct {
	handlers.App

	title   string
	baseUrl string

	create models.CreateUser
}

func (page *UserCreate) Pre(c *fiber.Ctx) error {
	if err := RequireAdmin(c); err != nil {
		return err
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
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (page *UserCreate) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverForm(&UserCreateForm{
		App: page.App,
	}, c, err)
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
