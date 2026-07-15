package livecontexthead

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func WriteAllow(root *chi.Mux, w http.ResponseWriter, r *http.Request) {
	methods := make([]string, 0, 2)
	if root.Match(chi.RouteContext(r.Context()), http.MethodGet, r.URL.Path) {
		methods = append(methods, http.MethodGet, http.MethodHead)
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
}
