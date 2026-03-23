package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed all:ui
var uiAssets embed.FS

type StaticApi struct{}

func NewStaticApi() *StaticApi {
	return &StaticApi{}
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
	
	// If a request doesn't match an API or a specific file, it might be a direct link to an admin page
	// So we might need a fallback for /admin
}
