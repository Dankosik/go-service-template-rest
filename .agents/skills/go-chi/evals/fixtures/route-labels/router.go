package routelabels

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Recorder interface {
	Record(route string)
}

func Router(labels Recorder) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			route := chi.RouteContext(req.Context()).RoutePattern()
			if route == "" {
				route = req.URL.Path
			}
			labels.Record(route)
			next.ServeHTTP(w, req)
		})
	})
	r.Get("/users/{userID}", func(http.ResponseWriter, *http.Request) {})
	return r
}
