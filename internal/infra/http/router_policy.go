package httpx

import (
	"net/http"
	"slices"
	"strings"

	"github.com/example/go-service-template-rest/internal/problem"
	"github.com/go-chi/chi/v5"
)

// applyHTTPPolicy installs the fallback answers for requests the mounted API did
// not match: what an unrouted path returns, and what an unmatched method returns
// beside its Allow header.
func applyHTTPPolicy(root chi.Router) {
	root.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, r, notFoundProblem())
	})

	root.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		allowMethods := allowedMethodsForPath(root, r.URL.Path)
		if len(allowMethods) == 0 {
			writeProblem(w, r, notFoundProblem())
			return
		}
		allowMethods = ensureMethodAllowed(allowMethods, http.MethodOptions)

		if r.Method == http.MethodOptions {
			setAllowHeader(w, allowMethods)

			if isCORSPreflightRequest(r) {
				writeProblem(w, r, problemResponse{code: problem.CodeMethodNotAllowed, detail: "cors preflight is not enabled"})
				return
			}

			w.WriteHeader(http.StatusNoContent)
			return
		}

		setAllowHeader(w, allowMethods)
		writeProblem(w, r, problemResponse{code: problem.CodeMethodNotAllowed, detail: "method is not allowed for this resource"})
	})
}

// boundedHTTPMethods is the set probed to build an Allow header, and bounds how
// many router matches one 405 costs.
//
// CONNECT and TRACE are deliberately absent: chi cannot route CONNECT from an
// ordinary handler and oapi-codegen never generates TRACE, so probing costs a
// router match per 405 to always answer no — and advertising TRACE in an Allow
// header is a finding in most scanners.
var boundedHTTPMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodDelete,
	http.MethodOptions,
	http.MethodPatch,
	http.MethodPost,
	http.MethodPut,
}

func allowedMethodsForPath(root chi.Router, path string) []string {
	if path == "" {
		path = "/"
	}

	allowMethods := make([]string, 0, len(boundedHTTPMethods))
	for _, method := range boundedHTTPMethods {
		routeContext := chi.NewRouteContext()
		if root.Match(routeContext, method, path) {
			allowMethods = append(allowMethods, method)
		}
	}
	return allowMethods
}

func setAllowHeader(w http.ResponseWriter, methods []string) {
	w.Header().Del("Allow")
	if len(methods) > 0 {
		w.Header().Set("Allow", strings.Join(methods, ", "))
	}
}

func ensureMethodAllowed(methods []string, method string) []string {
	if slices.Contains(methods, method) {
		return methods
	}
	return append(methods, method)
}

func isCORSPreflightRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	return r.Header.Get("Origin") != "" && r.Header.Get("Access-Control-Request-Method") != ""
}
