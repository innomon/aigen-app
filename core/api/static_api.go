package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed all:ui
var uiAssets embed.FS

type StaticApi struct {
	wwwRoot string
}

func NewStaticApi(wwwRoot string) *StaticApi {
	return &StaticApi{wwwRoot: wwwRoot}
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
	if a.wwwRoot != "" {
		// Serve the whole wwwRoot directory. 
		// This will handle /files/* (stored in wwwRoot/files) and any other static files.
		r.Handle("/files/*", http.FileServer(http.Dir(a.wwwRoot)))
	}
}
