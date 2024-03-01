package server

import (
	"fmt"
	"time"

	"github.com/39alpha/dorothy/server/auth"
	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/core/model"
	"github.com/gofiber/fiber/v2"
)

func Index(c *fiber.Ctx) error {
	return c.Render("views/index", merge(fiber.Map{
		"IsLoggedIn": c.Locals("AuthUser") != nil,
	}), "views/layouts/main")
}

func RegistrationForm(c *fiber.Ctx) error {
	return c.Render("views/register", merge(fiber.Map{
		"IsLoggedIn": c.Locals("AuthUser") != nil,
	}), "views/layouts/main")
}

func Registration(db *core.DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var new_user model.NewUser
		if err := c.BodyParser(&new_user); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
		}

		if err := db.CreateUser(&new_user, "user"); err != nil {
			return c.Status(fiber.StatusInternalServerError).RedirectBack("register")
		}

		return c.RedirectToRoute("", nil)
	}
}

func LoginForm(c *fiber.Ctx) error {
	return c.Render("views/login", merge(fiber.Map{
		"IsLoggedIn": c.Locals("AuthUser") != nil,
	}), "views/layouts/main")
}

func Login(jwtAuth *auth.Auth, db *core.DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var login model.UserLogin
		if err := c.BodyParser(&login); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("%v", err))
		}

		if err := db.ValidateCredentials(login.Email, login.Password); err != nil {
			return c.Status(fiber.StatusUnauthorized).RedirectBack("login")
		}

		user := &model.User{Email: login.Email}
		err := db.Select("id", "email", "name", "orcid").Where(user).First(user).Error
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).RedirectBack("login")
		}
		
		token, err := jwtAuth.MakeToken(user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).RedirectBack("login")
		}

		c.Cookie(&fiber.Cookie{
			Name:    "jwt",
			Value:   token,
			Expires: time.Now().Add(1 * time.Minute),
		})

		return c.RedirectToRoute("", nil)
	}
}

func Logout(c *fiber.Ctx) error {
	c.ClearCookie("jwt")
	return c.RedirectBack("")
}
