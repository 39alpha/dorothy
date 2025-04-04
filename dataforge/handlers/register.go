package handlers

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type RegisterForm struct{}

func (form *RegisterForm) Run(c *fiber.Ctx) error {
	GetLogger(c).Trace().Handler("RegisterForm").Send()
	return nil
}

func (form *RegisterForm) RenderHtml(c *fiber.Ctx) error {
	logger := GetLogger(c).Trace().Renderer("RegisterForm", "RenderHtml").Send()

	logger.Trace().Msg("Require Unauthenticated")
	authUser := GetAuthUser(c)
	if authUser != nil {
		logger.Debug().Uint("user_id", authUser.ID).Msg("User Authenticated")
		return c.Redirect("/")
	}

	return c.Render("register", Bind(c), "layouts/main")
}

type Register struct{}

func (page *Register) Run(c *fiber.Ctx) error {
	logger := GetLogger(c).Trace().Handler("RegisterForm").Send()

	logger.Trace().Msg("Parse Request Body")
	var newUser models.NewUser
	if err := c.BodyParser(&newUser); err != nil {
		logger.Debug().Err(err).Msg("Bad Request")
		return fiber.ErrBadRequest
	}

	logger.Trace().Msg("New User")
	if err := GetDB(c).NewUser(&newUser); err != nil {
		logger.Debug().Err(err).Str("user_email", newUser.Email).Msg("New User Failed")
		return fmt.Errorf("%w: User already exists", fiber.ErrBadRequest)
	}

	return nil
}

func (page *Register) Recover(c *fiber.Ctx, err error) error {
	GetLogger(c).Trace().Recover("RegisterForm").Send()
	return RecoverForm(&RegisterForm{}, c, err)
}

func (*Register) RenderHtml(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("RegisterForm", "RenderHtml").Send()
	return c.Redirect("/login")
}

func (*Register) RenderJson(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("RegisterForm", "RenderJson").Send()
	return c.JSON(fiber.Map{"message": "success"})
}

func (*Register) RenderText(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("RegisterForm", "RenderText").Send()
	return c.SendString("success")
}
