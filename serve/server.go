package serve

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"fmt"
	"maps"
	"net/http"
	"os"

	"github.com/39alpha/dorothy/core"
	"github.com/go-chi/jwtauth/v5"
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

func generateSecret() (string, error) {
	key := make([]byte, 32)

	if _, err := rand.Read(key); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

func makeJWTAuth() (*jwtauth.JWTAuth, error) {
	secret := os.Getenv("DOROTHY_OAUTH_SECRET")
	if secret == "" {
		var err error
		secret, err = generateSecret()
		if err != nil {
			return nil, fmt.Errorf("DOROTHY_SERVER_SECRET environment variable empty and failed to generate secret")
		}
	}

	return jwtauth.New("HS256", []byte(secret), nil), nil
}

func NewServer(config *core.Config) (*Server, error) {
	jwtAuth, err := makeJWTAuth()
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

	app.Use(Verifier(jwtAuth))
	app.Use(Authenticator(jwtAuth, session))

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
