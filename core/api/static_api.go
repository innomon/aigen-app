package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

//go:embed all:ui
var uiAssets embed.FS

type StaticApi struct {
	wwwRoot     string
	filesPrefix string
}

func NewStaticApi(wwwRoot, filesPrefix string) *StaticApi {
	return &StaticApi{wwwRoot: wwwRoot, filesPrefix: filesPrefix}
}

func (a *StaticApi) Register(r chi.Router) {
	// Root of embedded files is "ui"
	sub, _ := fs.Sub(uiAssets, "ui")
	fileServer := http.FileServer(http.FS(sub))

	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
	})

	// Serve all files from root
	r.Handle("/admin/*", http.StripPrefix("/admin", fileServer))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	// Serve from WWWRoot if it exists
	if a.wwwRoot != "" && a.filesPrefix != "" {
		pattern := a.filesPrefix
		if !strings.HasSuffix(pattern, "/") {
			pattern += "/"
		}
		pattern += "*"
		// Serve the whole wwwRoot directory. 
		// This will handle the files prefix and any other static files in wwwRoot.
		r.Handle(pattern, http.FileServer(http.Dir(a.wwwRoot)))
	}
}
