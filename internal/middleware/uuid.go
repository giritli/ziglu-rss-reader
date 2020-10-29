package middleware

import (
	"context"
	"errors"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"net/http"
)

type contextKey string

const uuidKey contextKey = "uuid"

func UUID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, err := uuid.Parse(chi.URLParam(r, "uuid")); err == nil {
			r = r.WithContext(context.WithValue(r.Context(), uuidKey, u))
		}

		next.ServeHTTP(w, r)
	})
}

func UUIDFromContext(ctx context.Context) (uuid.UUID, error) {
	u, ok := ctx.Value(uuidKey).(uuid.UUID)
	if !ok {
		return uuid.UUID{}, errors.New("no UUID found")
	}

	return u, nil
}