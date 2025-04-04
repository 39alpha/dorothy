package handlers

import (
	"fmt"
	"net/url"

	"github.com/39alpha/dorothy/dataforge/mail"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type ResetPasswordForm struct {
	isInvitation bool
	user         *models.User
}

func (form *ResetPasswordForm) Run(c *fiber.Ctx) error {
	logger := GetLogger(c).Trace().Handler("ResetPasswordForm").Send()

	logger.Trace().Msg("Require Unauthenticated")
	authUser := GetAuthUser(c)
	if authUser != nil {
		logger.Debug().Msg("Authenticated")
		if Redirectable(c) {
			logger.Debug().Redirect("/profile").Send()
			return c.Redirect("/profile")
		} else {
			logger.Debug().Msg("Forbidden")
			return fiber.ErrForbidden
		}
	}

	password := c.Query("id")
	token := c.Query("token")

	form.isInvitation = c.QueryBool("invitation")

	logger.Trace().Msg("Validate Reset Credentials")
	var err error
	form.user, err = GetDB(c).ValidateResetCredentials(token, password, false)
	if err != nil {
		logger.Debug().Err(err).Msg("Validate Reset Credentials Failed")
	}

	return nil
}

func (form *ResetPasswordForm) RenderHtml(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("ResetPasswordForm", "RenderHtml").Send()
	return c.Render("reset-password", Bind(c, fiber.Map{
		"User":         form.user,
		"IsInvitation": form.isInvitation,
	}), "layouts/main")
}

type ResetPassword struct {
	isInvitation bool
	user         *models.User
}

func (page *ResetPassword) Run(c *fiber.Ctx) error {
	logger := GetLogger(c).Trace().Handler("ResetPassword").Send()
	db := GetDB(c)

	logger.Trace().Msg("Require Unauthenticated")
	authUser := GetAuthUser(c)
	if authUser != nil {
		logger.Debug().Msg("Authenticated")
		if Redirectable(c) {
			logger.Debug().Redirect("/profile").Send()
			return c.Redirect("/profile")
		} else if !authUser.HasAdminRole() {
			logger.Debug().Msg("Forbidden")
			return fiber.ErrForbidden
		}
	}

	page.isInvitation = c.QueryBool("invitation")

	password := c.Query("id")
	token := c.Query("token")

	state := c.Locals("State").(fiber.Map)
	title := state["Title"].(string)
	baseUrl := state["BaseUrl"].(string)

	if password == "" || token == "" {
		logger.Trace().Msg("Reset Initialization")

		logger.Trace().Msg("Parse Request Body")
		reset := models.RequestPasswordReset{}
		if err := c.BodyParser(&reset); err != nil {
			logger.Debug().Err(err).Msg("Bad Request")
			return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
		}

		logger.Trace().Msg("Create Password Reset")
		token, err := db.CreatePasswordReset(reset.Email)
		if err != nil {
			logger.Error(err).Msg("Create Password Reset Failed")
			return GormToFiber(err)
		}

		logger.Trace().Msg("Get Mailer")
		mailer := GetMailer(c)
		if mailer == nil {
			logger.Error(fmt.Errorf("no mailer")).Msg("Invalid Initialization")
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

		logger.Trace().Msg("Send Email")
		if err = mailer.Send(message); err != nil {
			logger.Error(err).Msg("Failed to Send Mail")
			return fmt.Errorf("failed to send email: %w", err)
		}
	} else {
		logger.Trace().Msg("Reset Password")

		logger.Trace().Msg("Validate Reset Credentials")
		var err error
		page.user, err = db.ValidateResetCredentials(token, password, true)
		if err != nil {
			logger.Error(err).Msg("Invalid Credentials")
			return fiber.ErrInternalServerError
		}

		logger.Trace().Msg("Parse Request Body")
		change := models.ChangePassword{}
		if err := c.BodyParser(&change); err != nil {
			logger.Error(err).Msg("Invalid Credentials")
			return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
		}

		logger.Trace().Msg("Change User Password")
		if err := db.UserChangePassword(change); err != nil {
			logger.Error(err).Msg("Change User Password Failed")
			return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
		}
	}

	return nil
}

func (page *ResetPassword) RenderHtml(c *fiber.Ctx) error {
	logger := GetLogger(c).Trace().Renderer("ResetPassword", "RenderHtml").Send()

	if page.user != nil {
		logger.Trace().Msg("No User")
		if page.isInvitation {
			path := fmt.Sprintf("/login?Redirect=%s", url.QueryEscape("/profile"))
			logger.Debug().Redirect(path).Send()
			return c.Redirect(path)
		} else {
			logger.Debug().Redirect("/login").Send()
			return c.Redirect("/login")
		}
	} else {
		logger.Trace().Msg("Rendering")
		return c.Render("reset-password-sent", Bind(c), "layouts/main")
	}
}

func (page *ResetPassword) RenderJson(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("ResetPassword", "RenderJson").Send()
	return c.JSON(fiber.Map{"message": "success"})
}

func (page *ResetPassword) RenderText(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("ResetPassword", "RenderText").Send()
	return c.SendString("success")
}
