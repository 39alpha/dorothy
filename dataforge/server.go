package dataforge

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/dataforge/auth"
	"github.com/39alpha/dorothy/dataforge/db"
	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/log"
	"github.com/39alpha/dorothy/dataforge/mail"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
)

//go:embed views
var embeddedViews embed.FS

type Server struct {
	*fiber.App
	dorothy *core.Dorothy
	config  *core.ServerConfig
	auth    *auth.Auth
	db      *db.DB
	viewsfs http.FileSystem
	logger  log.Logger
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

func checkConfig(config *core.ServerConfig) error {
	if config == nil {
		return fmt.Errorf("no server configuration provided")
	} else if config.Database == nil {
		return fmt.Errorf("no server.database configuration provided")
	} else if config.Mail == nil {
		return fmt.Errorf("no server.mail configuration provided")
	}

	return nil
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

	config := dorothy.Config.Server
	if err := checkConfig(config); err != nil {
		return nil, err
	}

	logger, _ := log.NewLogger(config.Log)

	logger.Trace().Msg("Connecting to IPFS")
	if err := dorothy.ConnectIpfs(); err != nil {
		return nil, err
	}

	logger.Trace().Msg("Generating Auth")
	jwtAuth, err := auth.New()
	if err != nil {
		return nil, err
	}

	logger.Trace().Msg("Opening Database")
	session, err := db.Open(config.Database)
	if err != nil {
		return nil, err
	}

	logger.Trace().Msg("Initializing Database")
	if err = session.Initialize(); err != nil {
		return nil, err
	}

	var viewsfs http.FileSystem
	if config.Views == "" {
		logger.Info().Msg("Using embeddedViews views")
		fsys, err := fs.Sub(embeddedViews, "views")
		if err != nil {
			panic(err)
		}
		viewsfs = http.FS(fsys)
	} else {
		logger.Info().Msg("Using live views")
		viewsfs = http.Dir(config.Views)
	}

	logger.Trace().Msg("Creating Engine")
	engine := html.NewFileSystem(viewsfs, ".html")
	engine.AddFunc("TimeFmt", func(t time.Time) string {
		return t.Format("2006-01-02 15:04:05")
	})
	engine.AddFunc("GatewayUrl", func(hash string) string {
		return dorothy.Config.Ipfs.GatewayUrl(hash)
	})

	logger.Trace().Msg("Creating App")
	app := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: false,
		StrictRouting: false,
		ServerHeader:  "Dorothy",
		AppName:       "Dorothy",
		Views:         engine,
		ErrorHandler:  handlers.ErrorHandler,
	})

	server := &Server{app, dorothy, config, jwtAuth, session, viewsfs, logger}
	server.setup()

	return server, nil
}

func (d *Server) Listen(host string, port int) error {
	url := fmt.Sprintf("%s:%d", host, port)
	if d.config.BaseUrl == "" {
		if host == "" {
			d.config.BaseUrl = fmt.Sprintf("http://127.0.0.1:%d", port)
		} else {
			d.config.BaseUrl = url
		}
	}
	return d.App.Listen(url)
}

func (d *Server) ListenOnPort(port int) error {
	return d.Listen("", port)
}

func (d *Server) setup() {
	d.logger.Trace().Msg("Setting up App")

	d.Use(func(c *fiber.Ctx) error {
		key := make([]byte, 16)
		_, _ = rand.Read(key)
		id := base64.RawURLEncoding.EncodeToString(key)
		c.Locals("RequestID", id)

		logger := d.logger.RequestContext(c)
		c.Locals("Logger", &logger)

		d.logger.
			Trace().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("request_id", id).
			Msg("Initiated Request")

		return c.Next()
	})

	d.Use(recover.New())

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
		d.logger.Trace().
			Str("request_id", handlers.RequestID(c)).
			Msg("Setting Locals")

		if d.db == nil || d.dorothy == nil || d.auth == nil {
			return fmt.Errorf("%w: cannot load %s %s right now", fiber.ErrInternalServerError, c.Method(), c.Path())
		}

		c.Locals("Database", d.db)
		c.Locals("Dorothy", d.dorothy)
		c.Locals("Auth", d.auth)
		c.Locals("Mailer", mail.NewMailer(*d.config.Mail, d.viewsfs))

		return c.Next()
	})

	d.Use(func(c *fiber.Ctx) error {
		d.logger.Trace().
			Str("request_id", handlers.RequestID(c)).
			Msg("Setting Local State")

		state := fiber.Map{
			"BaseUrl":           d.config.BaseUrl,
			"Title":             "Dorothy",
			"SubTitle":          "Welcome to the dataforge",
			"AllowRegistration": d.config.AllowRegistration,
			"BrandColor":        d.config.BrandColor,
			"FooterText":        d.config.FooterText,
			"Path":              c.Path(),
		}
		if d.config.Title != "" {
			state["Title"] = d.config.Title
		}
		if d.config.SubTitle != "" {
			state["SubTitle"] = d.config.SubTitle
		}
		c.Locals("State", state)

		return c.Next()
	})

	d.Use(d.auth.Verifier())
	d.Use(d.auth.Authenticator(d.db))

	for _, route := range Routes() {
		d.logger.Trace().
			Str("endpoint", route.endpoint).
			Str("method", string(route.method)).
			Msg("Adding Route")

		if !d.config.AllowRegistration && route.endpoint == "/register" {
			continue
		}

		handler := handlers.PerRequest(route.handler)

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
