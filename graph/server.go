package graph

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/core/model"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/go-chi/jwtauth/v5"
	ipfs "github.com/ipfs/go-ipfs-api"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type Server struct {
	*chi.Mux
	config *core.Config
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

func generateSecret() (string, error) {
	key := make([]byte, 32)

	if _, err := rand.Read(key); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

func Authenticator(tokenAuth *jwtauth.JWTAuth, db *core.DatabaseSession) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			token, claims, err := jwtauth.FromContext(ctx)

			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			if token != nil && jwt.Validate(token, tokenAuth.ValidateOptions()...) == nil {
				if email, ok := claims["email"]; ok {
					var user model.User
					result := db.
						Select("id", "email", "name", "orcid").
						Where("email = ?", email).
						First(&user)
					if result.Error != nil {
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					}
					ctx = context.WithValue(ctx, "auth_user", user)
				}
			}

			// Token is authenticated, pass it through
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(hfn)
	}
}

func NewServer(config *core.Config) (*Server, error) {
	tokenAuth, err := makeJWTAuth()
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
		AllowedMethods:   []string{"POST"},
	}).Handler)

	c := Config{Resolvers: &Resolver{config: config, db: session}}
	router.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(Authenticator(tokenAuth, session))

		r.Handle("/query", handler.NewDefaultServer(NewExecutableSchema(c)))
	})
	router.Group(func(r chi.Router) {
		// DGM: The rate limit here should probably be a system setting
		r.Use(httprate.LimitByIP(5, 1*time.Minute))
		r.Use(middleware.AllowContentType("application/json"))

		r.Post("/login", core.Login(tokenAuth, config, session))
		r.Post("/register", core.Registration(config, session))
	})

	dorothy := &Server{router, config}

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
	return http.ListenAndServe(addr, d)
}
