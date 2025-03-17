package server

import (
	"embed"
	"fmt"
	"net/http"
	"time"

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
	auth *Auth
	db   *DB
}

func NewServer(global bool) (*Server, error) {
	dorothy, err := core.NewDorothy()
	if err != nil {
		return nil, err
	}

	return NewServerFromDorothy(dorothy, global)
}

func NewServerFromConfigFile(filename string, noinherit, global bool) (*Server, error) {
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

	return NewServerFromDorothy(dorothy, global)
}

func NewServerFromDorothy(dorothy *core.Dorothy, global bool) (*Server, error) {
	if global {
		if dorothy.Config.Ipfs == nil {
			dorothy.Config.Ipfs = &core.IpfsConfig{
				Global: true,
			}
		} else {
			dorothy.Config.Ipfs.Global = true
		}
		dorothy.ReloadIpfs()
	}
	if err := dorothy.ConnectIpfs(); err != nil {
		return nil, err
	}

	jwtAuth, err := NewAuth()
	if err != nil {
		return nil, err
	}

	session, err := OpenDB(dorothy.Config.Database)
	if err != nil {
		return nil, err
	}
	session.Initialize()

	engine := html.NewFileSystem(http.FS(viewsfs), ".html")
	engine.AddFunc("TimeFmt", func(t time.Time) string {
		return t.Format("2006-01-02 15:04:05")
	})
	engine.AddFunc("GatewayUrl", func(hash string) string {
		return dorothy.Config.Ipfs.GatewayUrl(hash)
	})

	app := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: false,
		StrictRouting: false,
		ServerHeader:  "Dorothy",
		AppName:       "Dorothy",
		Views:         engine,
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
			"Path":  c.Path(),
		})
		return c.Next()
	})
	d.Use(Verifier(d.auth))
	d.Use(Authenticator(d.auth, d.db))
	d.Use(d.LoadTeams)

	d.Get("/", d.Index)

	d.Get("/register", RegistrationForm)
	d.Post("/register", d.Registration)
	d.Get("/login", LoginForm)
	d.Post("/login", d.Login)
	d.Get("/logout", Logout)

	d.Get("/team/create", CreateTeamForm)
	d.Post("/team/create", d.CreateTeam)

	team := d.Group("/:team", d.LoadTeam)
	team.Get("/", GetTeam)
	team.Get("/settings", TeamSettingsForm)
	team.Post("/settings", d.UpdateTeamSettings)
	team.Get("/dataset/create", CreateDatasetForm)
	team.Post("/dataset/create", d.CreateDataset)

	dataset := team.Group("/:dataset", d.LoadDataset)
	dataset.Get("/", GetDataset(d.Ipfs.Identity))
	dataset.Post("/", d.RecieveDataset)
	dataset.Delete("/", d.DeleteDataset)
	dataset.Get("/settings", DatasetSettingsForm)
	dataset.Post("/settings", d.UpdateDatasetSettings)
}
