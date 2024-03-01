package server

import (
	"context"
	"embed"
	"maps"
	"net/http"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/server/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"

	ipfs "github.com/ipfs/go-ipfs-api"
)

//go:embed views
var viewsfs embed.FS

//go:embed static
var staticfs embed.FS

var global fiber.Map

func merge(other fiber.Map) fiber.Map {
	result := fiber.Map{}
	maps.Copy(result, global)
	maps.Copy(result, other)
	return result
}

type Server struct {
	*fiber.App
	config *core.Config
}

func NewServer(config *core.Config) (*Server, error) {
	jwtAuth, err := auth.NewAuth()
	if err != nil {
		return nil, err
	}

	global = fiber.Map{
		"Title": "Dorothy",
	}

	session, err := core.NewDatabaseSession(config)
	if err != nil {
		return nil, err
	}
	session.Initialize()

	engine := html.NewFileSystem(http.FS(viewsfs), ".html")

	app := fiber.New(fiber.Config{
		Prefork:       true,
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

	app.Use(auth.Verifier(jwtAuth))
	app.Use(auth.Authenticator(jwtAuth, session))

	app.Use("/static", filesystem.New(filesystem.Config{
		Root:       http.FS(staticfs),
		PathPrefix: "/static",
		Browse:     true,
	}))

	app.Get("/", Index)

	app.Get("/register", RegistrationForm)
	app.Post("/register", Registration(session))

	app.Get("/login", LoginForm)
	app.Post("/login", Login(jwtAuth, session))

	app.Get("/logout", Logout)

	dorothy := &Server{app, config}

	return dorothy, dorothy.initialize()
}

func (d *Server) initialize() error {
	client, err := core.NewIpfs(d.config)
	if err != nil {
		return err
	}

	return client.FilesMkdir(context.TODO(), core.FS_ROOT, func(r *ipfs.RequestBuilder) error {
		r.Option("parents", true)
		return nil
	})
}

func (d *Server) Listen(addr string) error {
	return d.App.Listen(addr)
}
