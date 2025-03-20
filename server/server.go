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
		err := dorothy.ReloadIpfs()
		if err != nil {
			return nil, err
		}
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

	if err = session.Initialize(); err != nil {
		return nil, err
	}

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
		Root:       http.FS(viewsfs),
		PathPrefix: "/views/static",
		Browse:     true,
	}))
	d.Use(favicon.New(favicon.Config{
		FileSystem: http.FS(viewsfs),
		File:       "/views/static/favicon.ico",
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

	d.Get("/", PerRequest[Index]())

	d.Get("/register", PerRequest[RegistrationForm]())
	d.Post("/register", PerRequest[Registration]())
	d.Get("/login", PerRequest[LoginForm]())
	d.Post("/login", PerRequest[Login]())
	d.Get("/logout", PerRequest[Logout]())

	d.Get("/team/create", PerRequest[CreateTeamForm]())
	d.Post("/team/create", PerRequest[CreateTeam]())

	team := d.Group("/:team")
	team.Get("/", PerRequest[GetTeam]())
	team.Get("/settings", PerRequest[UpdateTeamForm]())
	team.Post("/settings", PerRequest[UpdateTeam]())
	team.Get("/dataset/create", PerRequest[CreateDatasetForm]())
	team.Post("/dataset/create", PerRequest[CreateDataset]())

	dataset := team.Group("/:dataset")
	dataset.Get("/", PerRequest[GetDataset]())
	dataset.Post("/", PerRequest[ReceiveDataset]())
	dataset.Delete("/", PerRequest[DeleteDataset]())
	dataset.Get("/settings", PerRequest[UpdateDatasetForm]())
	dataset.Post("/settings", PerRequest[UpdateDataset]())
}

func (d *Server) Get(path string, ctors ...EndpointCtor) Router {
	handlers := []fiber.Handler{}
	for _, ctor := range ctors {
		handlers = append(handlers, d.RenderEndpointCtor(ctor))
	}

	return Router{
		Router: d.App.Get(path, handlers...),
		Server: d,
	}
}

func (d *Server) Post(path string, ctors ...EndpointCtor) Router {
	handlers := []fiber.Handler{}
	for _, ctor := range ctors {
		handlers = append(handlers, d.RenderEndpointCtor(ctor))
	}

	return Router{
		Router: d.App.Post(path, handlers...),
		Server: d,
	}
}

type Router struct {
	fiber.Router
	Server *Server
}

func (d *Server) Group(path string, ctors ...EndpointCtor) Router {
	handlers := []fiber.Handler{}
	for _, ctor := range ctors {
		handlers = append(handlers, d.RenderEndpointCtor(ctor))
	}

	return Router{
		Router: d.App.Group(path, handlers...),
		Server: d,
	}
}

func (r *Router) Get(path string, ctors ...EndpointCtor) Router {
	handlers := []fiber.Handler{}
	for _, ctor := range ctors {
		handlers = append(handlers, r.Server.RenderEndpointCtor(ctor))
	}

	return Router{
		Router: r.Router.Get(path, handlers...),
		Server: r.Server,
	}
}

func (r *Router) Post(path string, ctors ...EndpointCtor) Router {
	handlers := []fiber.Handler{}
	for _, ctor := range ctors {
		handlers = append(handlers, r.Server.RenderEndpointCtor(ctor))
	}

	return Router{
		Router: r.Router.Post(path, handlers...),
		Server: r.Server,
	}
}

func (r *Router) Delete(path string, ctors ...EndpointCtor) Router {
	handlers := []fiber.Handler{}
	for _, ctor := range ctors {
		handlers = append(handlers, r.Server.RenderEndpointCtor(ctor))
	}

	return Router{
		Router: r.Router.Delete(path, handlers...),
		Server: r.Server,
	}
}

func (r *Router) Group(path string, ctors ...EndpointCtor) Router {
	handlers := []fiber.Handler{}
	for _, ctor := range ctors {
		handlers = append(handlers, r.Server.RenderEndpointCtor(ctor))
	}

	return Router{
		Router: r.Router.Group(path, handlers...),
		Server: r.Server,
	}
}
