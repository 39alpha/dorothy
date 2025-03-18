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

func Verifier(auth *Auth) fiber.Handler {
	return func(c *fiber.Ctx) error {
		req, err := adaptor.ConvertRequest(c, false)
		if err != nil {
			return err
		}

		token, err := jwtauth.VerifyRequest(auth.JWTAuth, req, jwtauth.TokenFromCookie)
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

func Authenticator(auth *Auth, db *DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, claims, _ := fromContext(c)
		if token != nil && jwt.Validate(token, auth.ValidateOptions()...) == nil {
			if email, ok := claims["email"]; ok {
				var user *models.User
				err := db.Preload("Role").
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

type RegistrationForm struct{}

func (r *RegistrationForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("views/register", Bind(c, fiber.Map{
		"AuthUser": c.Locals("AuthUser"),
	}), "views/layouts/main")
}

type Registration struct {
	newUser models.NewUser
}

func (r *Registration) Preprocess(c *fiber.Ctx) error {
	var new_user models.NewUser
	if err := c.BodyParser(&new_user); err != nil {
		return fiber.ErrBadRequest
	}
	return nil
}

func (r *Registration) Run(d *Server) error {
	return d.db.CreateUser(&r.newUser)
	// return c.Status(fiber.StatusInternalServerError).RedirectBack("register")
}

func (r *Registration) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/login")
}

func (r *Registration) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

type LoginForm struct{}

func (form *LoginForm) RenderHtml(c *fiber.Ctx) error {
	if c.Locals("AuthUser") != nil {
		return c.Redirect("/")
	}

	bindings := Bind(c, fiber.Map{
		"AuthUser": c.Locals("AuthUser"),
		"Error":    c.Locals("Error"),
	})

	if c.Query("Redirect") != "" {
		bindings["Redirect"] = c.Query("Redirect")
	}

	return c.Render("views/login", bindings, "views/layouts/main")
}

type Login struct {
	fields struct {
		Redirect string
	}
	userLogin models.UserLogin
	token     string
}

func (page *Login) Preprocess(c *fiber.Ctx) error {
	c.BodyParser(&page.fields)

	if err := c.BodyParser(&page.userLogin); err != nil {
		return &RedirectError{
			error: fiber.ErrBadRequest,
			Path:  "/login",
		}
	}

	return nil
}

func (page *Login) Run(d *Server) error {
	if err := d.db.ValidateCredentials(page.userLogin.Email, page.userLogin.Password); err != nil {
		return &RedirectError{
			error: fmt.Errorf("%w: invalid login credentials", fiber.ErrUnauthorized),
			Path:  "/login",
		}
	}

	user := &models.User{Email: page.userLogin.Email}
	err := d.db.Select("id", "email", "name", "orcid").Where(user).First(user).Error
	if err != nil {
		return &RedirectError{
			error: fmt.Errorf("%w: an unexpected error occurred", fiber.ErrInternalServerError),
			Path:  "/login",
		}
	}

	page.token, err = d.auth.MakeToken(user)
	if err != nil {
		return &RedirectError{
			error: fmt.Errorf("%w: an unexpected error occurred", fiber.ErrInternalServerError),
			Path:  "/login",
		}
	}

	return nil
}

func (page *Login) Postprocess(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:    "jwt",
		Value:   page.token,
		Expires: time.Now().Add(72 * time.Hour),
	})

	return nil
}

func (page *Login) HandleError(d *Server, c *fiber.Ctx, err error) error {
	handler := &ErrorHandler{err}
	handler.Preprocess(c)

	if c.Accepts("text/html") != "" {
		return d.RenderEndpoint(&LoginForm{})(c)
	} else if c.Accepts("application/json") != "" || c.Accepts("text/plain") != "" {
		return d.RenderEndpoint(&ErrorHandler{err})(c)
	}

	return d.RenderEndpoint(&LoginForm{})(c)
}

func (page *Login) RenderHtml(c *fiber.Ctx) error {
	if page.fields.Redirect == "" {
		return c.Redirect("/")
	} else {
		return c.Redirect(page.fields.Redirect)
	}
}

func (page *Login) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (page *Login) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}

type Logout struct{}

func (page *Logout) Postprocess(c *fiber.Ctx) error {
	c.ClearCookie("jwt")
	return nil
}

func (page *Logout) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/")
}

func (page *Logout) RenderJson(c *fiber.Ctx) error {
	return c.SendString("success")
}

func (page *Logout) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
