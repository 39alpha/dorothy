package serve

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/core/model"
	"github.com/go-chi/jwtauth/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/lestrrat-go/jwx/v2/jwt"
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

func Login(jwtAuth *jwtauth.JWTAuth, db *core.DatabaseSession) fiber.Handler {
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

		claims := map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"orcid": user.Orcid,
		}
		jwtauth.SetIssuedNow(claims)
		jwtauth.SetExpiryIn(claims, 72*time.Hour)

		_, tokenString, err := jwtAuth.Encode(claims)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).RedirectBack("login")
		}

		c.Cookie(&fiber.Cookie{
			Name:    "jwt",
			Value:   tokenString,
			Expires: time.Now().Add(1 * time.Minute),
		})

		return c.RedirectToRoute("", nil)
	}
}

func Logout(c *fiber.Ctx) error {
	c.ClearCookie("jwt")
	return c.RedirectBack("")
}

func Verifier(ja *jwtauth.JWTAuth) fiber.Handler {
	return Verify(ja, jwtauth.TokenFromCookie)
}

func Verify(ja *jwtauth.JWTAuth, findTokenFns ...func(r *http.Request) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		req, err := adaptor.ConvertRequest(c, false)
		if err != nil {
			return err
		}

		token, err := jwtauth.VerifyRequest(ja, req, findTokenFns...)
		c.Locals("Token", token)
		c.Locals("Error", err)
		return c.Next()
	}
}

func FromContext(c *fiber.Ctx) (jwt.Token, map[string]interface{}, error) {
	token, _ := c.Locals("Token").(jwt.Token)

	var err error
	var claims map[string]interface{}

	if token != nil {
		claims, err = token.AsMap(context.Background())
		if err != nil {
			return token, nil, err
		}
	} else {
		claims = map[string]interface{}{}
	}

	err, _ = c.Locals("Error").(error)

	return token, claims, err
}

func Authenticator(ja *jwtauth.JWTAuth, db *core.DatabaseSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, claims, _ := FromContext(c)
		if token != nil && jwt.Validate(token, ja.ValidateOptions()...) == nil {
			if email, ok := claims["email"]; ok {
				var user model.User
				err := db.Select("id", "email", "name", "orcid").First(&user, "email = ?", email).Error
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).SendString("internal server error")
				}
				c.Locals("AuthUser", user)
			}
		}

		if c.Locals("AuthUser") == nil {
			c.ClearCookie("jwt")
		}

		return c.Next()
	}
}
