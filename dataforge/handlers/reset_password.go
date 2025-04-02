package handlers

import (
	"fmt"
	"net/url"

	"github.com/39alpha/dorothy/dataforge/mail"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type ResetPasswordForm struct {
	App

	isInvitation bool
	user         *models.User
}

func (form *ResetPasswordForm) Run(c *fiber.Ctx) error {
	authUser := GetAuthUser(c)
	if authUser != nil {
		if Redirectable(c) {
			return c.Redirect("/profile/change-password")
		} else {
			return fiber.ErrForbidden
		}
	}

	password := c.Query("id")
	token := c.Query("token")

	form.isInvitation = c.QueryBool("invitation")
	form.user, _ = GetDB(c).ValidateResetCredentials(token, password, false)

	return nil
}

func (form *ResetPasswordForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("reset-password", Bind(c, fiber.Map{
		"User":         form.user,
		"IsInvitation": form.isInvitation,
	}), "layouts/main")
}

type ResetPassword struct {
	App

	isInvitation bool
	user         *models.User
}

func (page *ResetPassword) Run(c *fiber.Ctx) error {
	db := GetDB(c)

	authUser := GetAuthUser(c)
	if authUser != nil {
		if Redirectable(c) {
			return c.Redirect("/profile")
		} else if !authUser.HasAdminRole() {
			return fiber.ErrForbidden
		}
	}

	page.isInvitation = c.QueryBool("invitation")

	password := c.Query("id")
	token := c.Query("token")

	state := c.Locals("State").(fiber.Map)
	title := state["Title"].(string)
	baseUrl := state["BaseUrl"].(string)

	if password == "" && token == "" {
		reset := models.RequestPasswordReset{}
		if err := c.BodyParser(&reset); err != nil {
			return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
		}

		token, err := db.CreatePasswordReset(reset.Email)
		if err != nil {
			return GormToFiber(err)
		}

		mailer := GetMailer(c)
		if mailer == nil {
			return fmt.Errorf("%w: cannot send invitations right now", fiber.ErrInternalServerError)
		}

		resetUrl := fmt.Sprintf("%s/reset-password?%s", baseUrl, token)

		message := mail.Message{
			To:           reset.Email,
			Subject:      fmt.Sprintf("Reset %s Password", title),
			HtmlTemplate: "emails/reset-password.html",
			TextTemplate: "emails/reset-password.txt",
			Data: map[string]any{
				"Title":    title,
				"ResetUrl": resetUrl,
			},
		}

		if err = mailer.Send(message); err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	} else {
		var err error
		page.user, err = db.ValidateResetCredentials(token, password, true)
		if err != nil {
			return fiber.ErrInternalServerError
		}

		change := models.ChangePassword{}
		if err := c.BodyParser(&change); err != nil {
			return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
		}
		if err := db.UserChangePassword(change); err != nil {
			return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
		}
	}

	return nil
}

func (page *ResetPassword) RenderHtml(c *fiber.Ctx) error {
	if page.user != nil {
		if page.isInvitation {
			return c.Redirect(fmt.Sprintf("/login?Redirect=%s", url.QueryEscape("/profile")))
		} else {
			return c.Redirect("/login")
		}
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
