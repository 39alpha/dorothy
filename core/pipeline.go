package core

import (
	"context"
	"net/http"
)

type Middleware func(http.Handler) http.Handler

type ContextAdder func(ctx context.Context) context.Context

func WithContext(a ContextAdder) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := a(r.Context())
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func WithConfig(config *Config) Middleware {
	return WithContext(func(ctx context.Context) context.Context {
		return context.WithValue(ctx, "config", config)
	})
}

func GetConfig(ctx context.Context) *Config {
	config, _ := ctx.Value("config").(*Config)
	return config
}

func WithDbSession(session *DatabaseSession) Middleware {
	return WithContext(func(ctx context.Context) context.Context {
		return context.WithValue(ctx, "db_session", session)
	})
}

func GetDbSession(ctx context.Context) *DatabaseSession {
	return ctx.Value("db_session").(*DatabaseSession)
}
