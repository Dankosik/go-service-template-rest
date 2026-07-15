package useafterroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", func(http.ResponseWriter, *http.Request) {})
	r.Use(middleware.RequestID)
	return r
}
