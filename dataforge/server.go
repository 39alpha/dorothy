package dataforge

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/dataforge/db"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
)

//go:embed views
var embeddedViews embed.FS

type Server struct {
	*fiber.App
	*core.Dorothy
	auth    *Auth
	db      *db.DB
	viewsfs http.FileSystem
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

	session, err := db.Open(dorothy.Config.Database)
	if err != nil {
		return nil, err
	}

	if err = session.Initialize(); err != nil {
		return nil, err
	}

	var viewsfs http.FileSystem
	if dorothy.Config.Server == nil || dorothy.Config.Server.Views == "" {
		fmt.Println("INFO: Using embedded views")
		fsys, err := fs.Sub(embeddedViews, "views")
		if err != nil {
			panic(err)
		}
		viewsfs = http.FS(fsys)
	} else {
		fmt.Println("INFO: Using live views")
		viewsfs = http.Dir(dorothy.Config.Server.Views)
	}

	engine := html.NewFileSystem(viewsfs, ".html")
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

	server := &Server{app, dorothy, jwtAuth, session, viewsfs}
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
		Root:       d.viewsfs,
		PathPrefix: "/static",
		Browse:     true,
	}))
	d.Use(favicon.New(favicon.Config{
		FileSystem: d.viewsfs,
		File:       "/static/favicon.ico",
	}))

	d.Use(func(c *fiber.Ctx) error {
		c.Locals("State", fiber.Map{
			"Title": "Dorothy",
			"Path":  c.Path(),
		})
		return c.Next()
	})
	d.Use(d.Verifier())
	d.Use(d.Authenticator())

	d.Get("/", PerRequest[Index]())

	d.Get("/register", PerRequest[RegistrationForm]())
	d.Post("/register", PerRequest[Registration]())
	d.Get("/login", PerRequest[LoginForm]())
	d.Post("/login", PerRequest[Login]())
	d.Get("/logout", PerRequest[Logout]())

	d.Get("/team/create", PerRequest[CreateTeamForm]())
	d.Post("/team/create", PerRequest[CreateTeam]())
	d.Post("/team/availability", PerRequest[TeamAvailability]())

	team := d.Group("/:team")
	team.Get("/", PerRequest[GetTeam]())
	team.Get("/settings", PerRequest[UpdateTeamForm]())
	team.Post("/settings", PerRequest[UpdateTeam]())
	team.Get("/dataset/create", PerRequest[CreateDatasetForm]())
	team.Post("/dataset/create", PerRequest[CreateDataset]())
	team.Post("/dataset/availability", PerRequest[DatasetAvailability]())

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
