package graph

import (
	"context"
	"net/http"

	"github.com/39alpha/dorothy/core"
)

type Middleware func(http.Handler) http.Handler

type ContextAdder func(ctx context.Context) (context.Context, error)

func WithContext(a ContextAdder) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, err := a(r.Context())
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func WithDbSession(config *core.Config) Middleware {
	return WithContext(func(ctx context.Context) (context.Context, error) {
		session, err := core.NewDatabaseSession(config)
		if err != nil {
			return ctx, err
		}
		return context.WithValue(ctx, "db_session", session), nil
	})
}

func GetDbSession(ctx context.Context) *core.DatabaseSession {
	return ctx.Value("db_session").(*core.DatabaseSession)
}
