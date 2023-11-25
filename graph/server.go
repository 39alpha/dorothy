package graph

import (
	"context"
	"net/http"

	"github.com/39alpha/dorothy/core"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	ipfs "github.com/ipfs/go-ipfs-api"
)

type Server struct {
	*chi.Mux
	config *core.Config
}

func NewServer(config *core.Config) (*Server, error) {
	session, err := core.NewDatabaseSession(config)
	if err != nil {
		return nil, err
	}
	session.Close()

	router := chi.NewRouter()

	router.Use(middleware.Recoverer)
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT"},
	}).Handler)

	router.Use(WithDbSession(config))

	schema := NewExecutableSchema(Config{Resolvers: &Resolver{config}})
	router.Handle("/query", handler.NewDefaultServer(schema))

	dorothy := &Server{router, config}
	err = dorothy.initialize()

	return dorothy, err
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
	return http.ListenAndServe(addr, d)
}
