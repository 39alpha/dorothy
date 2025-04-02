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

func (form *UserCreateForm) Run(c *fiber.Ctx) error {
	return RequireAdmin(c)
}

func (form *UserCreateForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("admin/user-create", handlers.Bind(c), "layouts/main")
}

type UserCreate struct {
	handlers.App
}

func (page *UserCreate) Run(c *fiber.Ctx) error {
	if err := RequireAdmin(c); err != nil {
		return err
	}

	var create models.CreateUser
	if err := c.BodyParser(&create); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	state := c.Locals("State").(fiber.Map)
	title := state["Title"].(string)
	baseUrl := state["BaseUrl"].(string)

	token, err := page.DB().InviteUser(create)
	if err != nil {
		fmt.Println(err)
		return handlers.GormToFiber(err)
	}

	mailer := handlers.GetMailer(c)
	if mailer == nil {
		return fmt.Errorf("%w: cannot send invitations right now", fiber.ErrInternalServerError)
	}

	resetUrl := fmt.Sprintf("%s/reset-password?invitation=true&%s", baseUrl, token)

	message := mail.Message{
		To:           create.Email,
		Subject:      fmt.Sprintf("Invitation to %s", title),
		HtmlTemplate: "emails/invitation.html",
		TextTemplate: "emails/invitation.txt",
		Data: map[string]any{
			"Title":    title,
			"ResetUrl": resetUrl,
		},
	}

	if err = mailer.Send(message); err != nil {
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
