package graph

import (
	"context"
	"net/http"
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
