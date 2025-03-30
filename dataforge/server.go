package dataforge

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/dataforge/auth"
	"github.com/39alpha/dorothy/dataforge/db"
	"github.com/39alpha/dorothy/dataforge/handlers"
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
	dorothy *core.Dorothy
	auth    *auth.Auth
	db      *db.DB
	viewsfs http.FileSystem
}

func (s *Server) Dorothy() *core.Dorothy {
	return s.dorothy
}

func (s *Server) Auth() *auth.Auth {
	return s.auth
}

func (s *Server) DB() *db.DB {
	return s.db
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

	jwtAuth, err := auth.New()
	if err != nil {
		return nil, err
	}

	serverConfig := dorothy.Config.Server
	if serverConfig == nil {
		return nil, fmt.Errorf("no server configuration provided")
	}

	if serverConfig.Database == nil {
		return nil, fmt.Errorf("no server.database configuration provided")
	}

	session, err := db.Open(serverConfig.Database)
	if err != nil {
		return nil, err
	}

	if err = session.Initialize(); err != nil {
		return nil, err
	}

	var viewsfs http.FileSystem
	if serverConfig.Views == "" {
		fmt.Println("INFO: Using embedded views")
		fsys, err := fs.Sub(embeddedViews, "views")
		if err != nil {
			panic(err)
		}
		viewsfs = http.FS(fsys)
	} else {
		fmt.Println("INFO: Using live views")
		viewsfs = http.Dir(serverConfig.Views)
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
	d.Use(d.auth.Verifier())
	d.Use(d.auth.Authenticator(d.db))

	for _, route := range Routes() {
		handler := handlers.PerRequest(route.handler, d)

		switch route.method {
		case GET:
			d.Get(route.endpoint, handler)
		case POST:
			d.Post(route.endpoint, handler)
		case DELETE:
			d.Delete(route.endpoint, handler)
		}
	}
}
