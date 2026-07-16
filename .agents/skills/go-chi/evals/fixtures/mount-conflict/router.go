package mountconflict

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Router() http.Handler {
	r := chi.NewRouter()
	r.Handle("/files/*", http.FileServer(http.Dir(".")))
	r.Mount("/files", chi.NewRouter())
	return r
}
