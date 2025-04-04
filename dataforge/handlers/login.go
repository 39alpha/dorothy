package handlers

import (
	"fmt"
	"time"

	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type LoginForm struct{}

func (form *LoginForm) Run(c *fiber.Ctx) error {
	GetLogger(c).Trace().Handler("LoginForm").Send()
	return nil
}

func (form *LoginForm) RenderHtml(c *fiber.Ctx) error {
	logger := GetLogger(c).Trace().Renderer("LoginForm", "RenderHtml").Send()

	logger.Trace().Msg("Require Unauthenticated")
	authUser := GetAuthUser(c)
	if authUser != nil {
		logger.Trace().Uint("user_id", authUser.ID).Msg("User Authenticated")
		return c.Redirect("/")
	}

	bindings := Bind(c)

	if c.Query("Redirect") != "" {
		bindings["Redirect"] = c.Query("Redirect")
	}

	return c.Render("login", bindings, "layouts/main")
}

type Login struct {
	Redirect string
}

func (page *Login) Run(c *fiber.Ctx) error {
	logger := GetLogger(c).Trace().Handler("LoginForm").Send()
	db := GetDB(c)

	var fields struct {
		Redirect string
	}
	_ = c.BodyParser(&fields)
	page.Redirect = fields.Redirect

	logger.Trace().Msg("Parse Request Body")
	userLogin := models.UserLogin{}
	if err := c.BodyParser(&userLogin); err != nil {
		logger.Debug().Err(err).Msg("Bad Request")
		return fiber.ErrBadRequest
	}

	logger.Trace().Msg("Validate Credentials")
	err := db.ValidateCredentials(userLogin.Email, userLogin.Password)
	if err != nil {
		logger.Debug().Err(err).Msg("Invalid Credentials")
		return fmt.Errorf("%w: invalid login credentials", fiber.ErrUnauthorized)
	}

	logger.Trace().Msg("Find User")
	user := &models.User{Email: userLogin.Email}
	err = db.Select("id", "email", "name", "orcid").Where(user).First(user).Error
	if err != nil {
		logger.Error(err).Msg("Find User Failed")
		return fmt.Errorf("%w: an unexpected error occurred", fiber.ErrInternalServerError)
	}

	logger.Trace().Msg("Make Auth Token")
	token, err := GetAuth(c).MakeToken(user)
	if err != nil {
		logger.Error(err).Msg("Make Auth Token Failed")
		return fmt.Errorf("%w: an unexpected error occurred", fiber.ErrInternalServerError)
	}

	logger.Trace().Msg("Add Cookie")
	c.Cookie(&fiber.Cookie{
		Name:    "jwt",
		Value:   token,
		Expires: time.Now().Add(72 * time.Hour),
	})

	return nil
}

func (page *Login) Recover(c *fiber.Ctx, err error) error {
	GetLogger(c).Trace().Recover("LoginForm").Send()
	return RecoverForm(&LoginForm{}, c, err)
}

func (page *Login) RenderHtml(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("LoginForm", "RenderHtml").Send()

	if page.Redirect == "" {
		return c.Redirect("/")
	} else {
		return c.Redirect(page.Redirect)
	}
}

func (*Login) RenderJson(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("LoginForm", "RenderJson").Send()
	return c.JSON(fiber.Map{"message": "success"})
}

func (*Login) RenderText(c *fiber.Ctx) error {
	GetLogger(c).Trace().Renderer("LoginForm", "RenderText").Send()
	return c.SendString("success")
}
