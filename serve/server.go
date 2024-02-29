package serve

import (
	"context"

	"github.com/39alpha/dorothy/core"
	"github.com/gofiber/fiber/v2"

	ipfs "github.com/ipfs/go-ipfs-api"
)

type Server struct {
	*fiber.App
	config *core.Config
}

func NewServer(config *core.Config) (*Server, error) {
	session, err := core.NewDatabaseSession(config)
	if err != nil {
		return nil, err
	}
	session.Initialize()

	app := fiber.New(fiber.Config{
		Prefork: true,
		CaseSensitive: false,
		StrictRouting: false,
		ServerHeader: "Dorothy",
		AppName: "Dorothy",
	})

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
