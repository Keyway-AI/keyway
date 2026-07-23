package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed webdist
var webFS embed.FS

// spaHandler serves the embedded web dashboard, falling back to index.html for
// client-side routes (SPA history mode). The production bundle is written into
// webdist/ at container-build time; a placeholder ships otherwise.
func spaHandler() http.Handler {
	sub, err := fs.Sub(webFS, "webdist")
	if err != nil {
		return http.NotFoundHandler()
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API routes are handled elsewhere; never fall through to the SPA.
		if strings.HasPrefix(r.URL.Path, "/v1/") {
			http.NotFound(w, r)
			return
		}
		// If the requested asset exists, serve it; otherwise serve index.html.
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(sub, p); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
