package handlers

import (
	"fmt"
	"net/url"

	"github.com/39alpha/dorothy/dataforge/mail"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type ResetPasswordForm struct {
	ErrorHandler

	token    string
	password string
	user     *models.User
}

func (form *ResetPasswordForm) Pre(c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser != nil {
		if Redirectable(c) {
			return c.Redirect("/profile/change-password")
		} else {
			return fiber.ErrForbidden
		}
	}

	form.password = c.Query("id")
	form.token = c.Query("token")

	return nil
}

func (form *ResetPasswordForm) Run() error {
	form.user, _ = form.DB().ValidateResetCredentials(form.token, form.password, false)
	return nil
}

func (form *ResetPasswordForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("reset-password", Bind(c, fiber.Map{
		"User": form.user,
	}), "layouts/main")
}

type ResetPassword struct {
	ErrorHandler

	user *models.User

	title   string
	baseUrl string
	reset   models.RequestPasswordReset
	change  models.ChangePassword
}

func (page *ResetPassword) Pre(c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser != nil {
		if Redirectable(c) {
			return c.Redirect("/profile/change-password")
		} else {
			return fiber.ErrForbidden
		}
	}

	password := c.Query("id")
	token := c.Query("token")
	state := c.Locals("State").(fiber.Map)
	page.title = state["Title"].(string)
	page.baseUrl = state["BaseUrl"].(string)

	if password == "" && token == "" {
		if err := c.BodyParser(&page.reset); err != nil {
			return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
		}
	} else {
		var err error
		page.user, err = page.DB().ValidateResetCredentials(token, password, true)
		if err != nil {
			return fiber.ErrInternalServerError
		}
		if err := c.BodyParser(&page.change); err != nil {
			return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
		}
	}

	return nil
}

func (page *ResetPassword) Run() error {
	if page.user != nil {
		if err := page.DB().UserChangePassword(page.change); err != nil {
			return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
		}
	} else {
		token, id, err := page.DB().CreatePasswordReset(page.reset.Email)
		if err != nil {
			return err
		}

		resetUrl := fmt.Sprintf(
			"%s/reset-password?id=%s&token=%s",
			page.baseUrl,
			url.QueryEscape(id),
			url.QueryEscape(token),
		)

		err = page.Mailer().Send(mail.Message{
			To:           page.reset.Email,
			Subject:      "Reset Dorothy Password",
			HtmlTemplate: "emails/reset-password.html",
			TextTemplate: "emails/reset-password.txt",
			Data: map[string]any{
				"Title":    page.title,
				"ResetUrl": resetUrl,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	}
	return nil
}

func (page *ResetPassword) RenderHtml(c *fiber.Ctx) error {
	if page.user != nil {
		return c.Redirect("/login")
	} else {
		return c.Render("reset-password-sent", Bind(c), "layouts/main")
	}
}

func (page *ResetPassword) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (page *ResetPassword) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
