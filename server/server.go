package server

import (
	"context"
	"embed"
	"fmt"
	"net/http"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/server/model"
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
	auth    *Auth
	session *DatabaseSession
}

func NewServer() (*Server, error) {
	dorothy, err := core.NewDorothy()
	if err != nil {
		return nil, err
	}

	return NewServerFromDorothy(dorothy)
}

func NewServerFromConfigFile(filename string, noinherit bool) (*Server, error) {
	dorothy, err := core.NewDorothy()
	if err != nil {
		return nil, err
	}

	if noinherit {
		if err := dorothy.ResetConfig(); err != nil {
			return nil, err
		}
	}

	if err = dorothy.LoadConfigFile(filename); err != nil {
		return nil, err
	}

	return NewServerFromDorothy(dorothy)
}

func NewServerFromDorothy(dorothy *core.Dorothy) (*Server, error) {
	if err := dorothy.ConnectIpfs(); err != nil {
		return nil, err
	}

	jwtAuth, err := NewAuth()
	if err != nil {
		return nil, err
	}

	session, err := NewDatabaseSession(dorothy.Config.Database)
	if err != nil {
		return nil, err
	}
	session.Initialize()

	app := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: false,
		StrictRouting: false,
		ServerHeader:  "Dorothy",
		AppName:       "Dorothy",
		Views:         html.NewFileSystem(http.FS(viewsfs), ".html"),
	})

	server := &Server{app, dorothy, jwtAuth, session}
	server.setup()

	return server, nil
}

func (d *Server) Listen(host string, port int) error {
	return d.App.Listen(fmt.Sprintf("%s:%d", host, port))
}

func (d *Server) ListenOnPort(port int) error {
	return d.Listen("", port)
}

func (d *Server) setup() {
	d.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	d.Use("/static", filesystem.New(filesystem.Config{
		Root:       http.FS(staticfs),
		PathPrefix: "/static",
		Browse:     true,
	}))
	d.Use(favicon.New(favicon.Config{
		FileSystem: http.FS(staticfs),
		File:       "/static/favicon.ico",
	}))

	d.Use(func(c *fiber.Ctx) error {
		c.Locals("State", fiber.Map{
			"Title": "Dorothy",
		})
		return c.Next()
	})
	d.Use(Verifier(d.auth))
	d.Use(Authenticator(d.auth, d.session))
	d.Use(GetOrganizations(d.session))

	d.Get("/", Index)

	d.Get("/register", RegistrationForm)
	d.Post("/register", Registration(d.session))
	d.Get("/login", LoginForm)
	d.Post("/login", Login(d.auth, d.session))
	d.Get("/logout", Logout)

	d.Get("/organization/create", CreateOrganizationForm)
	d.Post("/organization/create", CreateOrganization(d.session))

	organization := d.Group("/:organization", GetOrganization(d.session))
	organization.Get("/", Organization)
	organization.Get("/dataset/create", CreateDatasetForm)
	organization.Post("/dataset/create", d.CreateDatasetHandler())

	dataset := organization.Group("/:dataset", d.GetDataset())
	dataset.Get("/", d.Dataset())
	dataset.Post("/", d.RecieveDataset())
}

func (d *Server) CreateDataset(dataset model.NewDataset, authUser *model.User) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manifest, err := d.Ipfs.CreateEmptyManifest(ctx)
	if err != nil {
		return nil
	}

	return d.session.CreateDataset(dataset, manifest, authUser)
}
