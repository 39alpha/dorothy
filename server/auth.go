package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/39alpha/dorothy/server/models"
	"github.com/go-chi/jwtauth/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type Auth struct {
	*jwtauth.JWTAuth
}

func NewAuth() (*Auth, error) {
	secret := os.Getenv("DOROTHY_OAUTH_SECRET")
	if secret == "" {
		var err error
		secret, err = generateSecret()
		if err != nil {
			return nil, fmt.Errorf("DOROTHY_SERVER_SECRET environment variable empty and failed to generate secret")
		}
	}

	return &Auth{jwtauth.New("HS256", []byte(secret), nil)}, nil
}

func generateSecret() (string, error) {
	key := make([]byte, 32)

	if _, err := rand.Read(key); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

func (auth *Auth) MakeToken(user *models.User) (string, error) {
	claims := map[string]any{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"orcid": user.Orcid,
	}
	jwtauth.SetIssuedNow(claims)
	jwtauth.SetExpiryIn(claims, 72*time.Hour)

	_, token, err := auth.Encode(claims)

	return token, err
}

func (d *Server) Verifier() fiber.Handler {
	return func(c *fiber.Ctx) error {
		req, err := adaptor.ConvertRequest(c, false)
		if err != nil {
			return err
		}

		token, _ := jwtauth.VerifyRequest(d.auth.JWTAuth, req, jwtauth.TokenFromCookie)
		c.Locals("Token", token)
		return c.Next()
	}
}

func fromContext(c *fiber.Ctx) (jwt.Token, map[string]any, error) {
	token, _ := c.Locals("Token").(jwt.Token)

	var err error
	var claims map[string]any

	if token != nil {
		claims, err = token.AsMap(context.Background())
		if err != nil {
			return token, nil, err
		}
	} else {
		claims = map[string]any{}
	}

	err, _ = c.Locals("Error").(error)

	return token, claims, err
}

func (d *Server) Authenticator() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, claims, _ := fromContext(c)
		if token != nil && jwt.Validate(token, d.auth.ValidateOptions()...) == nil {
			if email, ok := claims["email"]; ok {
				var user *models.User
				err := d.db.Preload("Role").
					Preload("TeamPrivileges.Team").
					Preload("DatasetPrivileges.Dataset.Team").
					First(&user, "email = ?", email).Error

				if err != nil {
					return c.Status(fiber.StatusInternalServerError).
						SendString("internal server error")
				}

				user.PasswordHash = nil
				c.Locals("AuthUser", user)
			}
		}

		if c.Locals("AuthUser") == nil {
			c.ClearCookie("jwt")
		}

		return c.Next()
	}
}
