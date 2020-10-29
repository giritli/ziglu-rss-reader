package api

import (
	"encoding/json"
	"github.com/go-chi/chi"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"net/http"
	"reader/internal/api/response"
	"reader/internal/middleware"
	"reader/internal/storage"
	"time"
)

type API struct {
	s storage.Storage
}

const OffsetTimeFormat = "2006-01-02T15:04:05"

func timeOffsetFromRequest(r *http.Request) time.Time {
	offset := r.URL.Query().Get("offset")
	t, err := time.Parse(OffsetTimeFormat, offset)
	if err != nil {
		t = time.Now()
	}

	return t
}

func NewAPI(s storage.Storage) http.Handler {
	r := chi.NewRouter()
	a := &API{
		s: s,
	}

	r.Use(chiMiddleware.SetHeader("Content-Type", "application/json"))
	r.Get("/feeds", a.Feeds)
	r.Get("/latest", a.Latest)

	r.Group(func(r chi.Router) {
		r.Use(middleware.UUID)

		r.Get("/latest/{uuid}", a.LatestFromFeed)
		r.Get("/article/{uuid}", a.Article)
	})

	return r
}

func (a *API) Feeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := a.s.Feeds()
	if err != nil {
		response.WithMessage(w, http.StatusInternalServerError, "could not retrieve feeds")
		return
	}

	if err := json.NewEncoder(w).Encode(feeds); err != nil {
		response.WithMessage(w, http.StatusInternalServerError, "could not generate response")
	}
}

func (a *API) Latest(w http.ResponseWriter, r *http.Request) {
	articles, err := a.s.Latest(timeOffsetFromRequest(r))
	if err != nil {
		response.WithMessage(w, http.StatusInternalServerError, "could not retrieve latest articles from feed")
		return
	}

	if err := json.NewEncoder(w).Encode(articles); err != nil {
		response.WithMessage(w, http.StatusInternalServerError, "could not generate response")
	}
}

func (a *API) LatestFromFeed(w http.ResponseWriter, r *http.Request) {
	u, err := middleware.UUIDFromContext(r.Context())
	if err != nil {
		response.WithMessage(w, http.StatusBadRequest, "UUID not found")
		return
	}

	articles, err := a.s.LatestFromFeed(u, timeOffsetFromRequest(r))
	if err != nil {
		response.WithMessage(w, http.StatusInternalServerError, "could not retrieve latest articles from feed")
		return
	}

	if err := json.NewEncoder(w).Encode(articles); err != nil {
		response.WithMessage(w, http.StatusInternalServerError, "could not generate response")
	}
}

func (a *API) Article(w http.ResponseWriter, r *http.Request) {
	u, err := middleware.UUIDFromContext(r.Context())
	if err != nil {
		response.WithMessage(w, http.StatusBadRequest, "UUID not found")
		return
	}

	article, err := a.s.Article(u)
	if err != nil {
		response.WithMessage(w, http.StatusInternalServerError, "could not retrieve article")
		return
	}

	if err := json.NewEncoder(w).Encode(article); err != nil {
		response.WithMessage(w, http.StatusInternalServerError, "could not generate response")
	}
}