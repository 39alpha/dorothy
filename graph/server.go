package graph

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/core/model"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/go-chi/oauth"
	ipfs "github.com/ipfs/go-ipfs-api"
)

type Server struct {
	*chi.Mux
	config *core.Config
}

type OAuthSettings struct {
	secret string
	ttl    time.Duration
}

func loadOAuthSettings() (*OAuthSettings, error) {
	ttl_string := os.Getenv("DOROTHY_OAUTH_TTL")
	if ttl_string == "" {
		ttl_string = "300"
	}
	ttl, err := strconv.Atoi(ttl_string)
	if err != nil {
		return nil, fmt.Errorf("DOROTHY_OAUTH_TTL is not a valid integer")
	}

	secret := os.Getenv("DOROTHY_OAUTH_SECRET")
	if secret == "" {
		secret, err = generateSecret()
		if err != nil {
			return nil, fmt.Errorf("DOROTHY_SERVER_SECRET environment variable empty and failed to generate secret")
		}
	}

	return &OAuthSettings{
		secret: secret,
		ttl:    time.Second * time.Duration(ttl),
	}, nil
}

func generateSecret() (string, error) {
	key := make([]byte, 32)

	if _, err := rand.Read(key); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

func NewServer(config *core.Config) (*Server, error) {
	oauth_settings, err := loadOAuthSettings()
	if err != nil {
		return nil, err
	}

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

	s := oauth.NewBearerServer(
		// DGM: These details should be system settings (ish)
		oauth_settings.secret,
		oauth_settings.ttl,
		&UserVerifier{db: session},
		nil,
	)

	c := Config{Resolvers: &Resolver{config: config, db: session}}
	router.Route("/", func(r chi.Router) {
		r.Use(oauth.Authorize(oauth_settings.secret, nil))
		r.Handle("/query", handler.NewDefaultServer(NewExecutableSchema(c)))
	})
	router.Post("/token", s.UserCredentials)
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
			result := session.Find(&user)
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
