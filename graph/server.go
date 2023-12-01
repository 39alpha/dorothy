package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/core/model"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
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
	session.Initialize()

	router := chi.NewRouter()

	router.Use(middleware.Recoverer)
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT"},
	}).Handler)

	c := Config{Resolvers: &Resolver{config: config, db: session}}
	router.Handle("/", playground.Handler("Dorothy", "/query"))
	router.Handle("/query", handler.NewDefaultServer(NewExecutableSchema(c)))
	router.Route("/register", func(r chi.Router) {
		// DGM: The rate limit here should probably be a system setting
		r.Use(httprate.LimitByIP(5, 1*time.Minute))
		r.Use(middleware.AllowContentType("application/json"))
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			decoder := json.NewDecoder(r.Body)
			var new_user model.NewUser
			if err := decoder.Decode(&new_user); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("illformed request"))
				return
			}

			_, err := core.CreateUser(r.Context(), config, session, &new_user)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("user already exists"))
				return
			}

			var user model.User
			result := session.Select("email", "name", "orcid").Find(&user)
			if result.Error != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("an unexpected error occured"))
				return
			}

			var buf bytes.Buffer
			encoder := json.NewEncoder(&buf)
			if err := encoder.Encode(&user); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("an unexpected error occured"))
				return
			}

			w.Write(buf.Bytes())
		})
	})

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
