package server

import (
	"embed"
	"fmt"
	"net/http"

	"github.com/39alpha/dorothy/core"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
)

//go:embed views
var viewsfs embed.FS

//go:embed static
var staticfs embed.FS

type Server struct {
	*fiber.App
	*core.Dorothy
}

func NewServer() (*Server, error) {
	dorothy, err := core.NewDorothy()
	if err != nil {
		return nil, err
	}
	return NewServerFromDorothy(dorothy)
}

func NewServerFromConfigFile(filename string, noinherit bool) (*Server, error) {
	dorothy, err := core.NewDorothyFromConfigFile(filename, noinherit)
	if err != nil {
		return nil, err
	}
	return NewServerFromDorothy(dorothy)
}

func NewServerFromDorothy(dorothy *core.Dorothy) (*Server, error) {
	jwtAuth, err := NewAuth()
	if err != nil {
		return nil, err
	}

	session, err := NewDatabaseSession(dorothy.Config.Database)
	if err != nil {
		return nil, err
	}
	session.Initialize()

	engine := html.NewFileSystem(http.FS(viewsfs), ".html")

	app := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: false,
		StrictRouting: false,
		ServerHeader:  "Dorothy",
		AppName:       "Dorothy",
		Views:         engine,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	app.Use("/static", filesystem.New(filesystem.Config{
		Root:       http.FS(staticfs),
		PathPrefix: "/static",
		Browse:     true,
	}))
	app.Use(favicon.New(favicon.Config{
		FileSystem: http.FS(staticfs),
		File:       "/static/favicon.ico",
	}))

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("State", fiber.Map{
			"Title": "Dorothy",
		})
		return c.Next()
	})
	app.Use(Verifier(jwtAuth))
	app.Use(Authenticator(jwtAuth, session))
	app.Use(GetOrganizations(session))

	app.Get("/", Index)

	app.Get("/register", RegistrationForm)
	app.Post("/register", Registration(session))
	app.Get("/login", LoginForm)
	app.Post("/login", Login(jwtAuth, session))
	app.Get("/logout", Logout)

	app.Get("/organization/create", CreateOrganizationForm)
	app.Post("/organization/create", CreateOrganization(session))

	organization := app.Group("/:organization", GetOrganization(session))
	organization.Get("/", Organization)
	organization.Get("/dataset/create", CreateDatasetForm)
	organization.Post("/dataset/create", CreateDataset(session))

	dataset := organization.Group("/:dataset", GetDataset(session))
	dataset.Get("/", Dataset)

	return &Server{app, dorothy}, nil
}

func (d *Server) Listen(host string, port int) error {
	return d.App.Listen(fmt.Sprintf("%s:%d", host, port))
}

func (d *Server) ListenOnPort(port int) error {
	return d.Listen("", port)
}
