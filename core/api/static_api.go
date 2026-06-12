package api

import (
	"archive/zip"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/innomon/aigen-app/core/app_extensions"
	"github.com/go-chi/chi/v5"
)


//go:embed all:ui
var uiAssets embed.FS

type StaticApi struct {
	wwwRoot       string
	customUIPath  string
	filesPrefix   string
	extensionService *app_extensions.AppExtensionService
}

func NewStaticApi(wwwRoot, customUIPath, filesPrefix string, extensionService *app_extensions.AppExtensionService) *StaticApi {
	return &StaticApi{wwwRoot: wwwRoot, customUIPath: customUIPath, filesPrefix: filesPrefix, extensionService: extensionService}
}

func (a *StaticApi) Register(r chi.Router) {
	sub, _ := fs.Sub(uiAssets, "ui")

	// Set up OverlayFS with custom directory at higher priority
	var customFS fs.FS
	if a.customUIPath != "" {
		if _, err := os.Stat(a.customUIPath); err == nil {
			customFS = os.DirFS(a.customUIPath)
		}
	}

	overlayFS := NewOverlayFS(customFS, sub)
	fileServer := http.FileServer(http.FS(overlayFS))

	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
	})

	r.Handle("/admin/*", http.StripPrefix("/admin", fileServer))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	// Serve from app extension JARs
	if a.extensionService != nil {
		r.Get("/_extensions/{id}/*", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			info, ok := a.extensionService.Get(id)
			if !ok || info.Status != app_extensions.StatusActive {
				http.NotFound(w, r)
				return
			}

			// Open the JAR and serve from wwwroot/
			jar, err := zip.OpenReader(info.Path)
			if err != nil {
				http.Error(w, "failed to open extension", http.StatusInternalServerError)
				return
			}
			defer jar.Close()

			sub, err := fs.Sub(jar, "wwwroot")
			if err != nil {
				http.NotFound(w, r)
				return
			}

			// Trim the prefix /_extensions/{id}/
			prefix := "/_extensions/" + id + "/"
			path := strings.TrimPrefix(r.URL.Path, prefix)
			
			file, err := sub.Open(path)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer file.Close()
			
			// We need to handle content type correctly, so using http.ServeContent might be better
			// for a simple implementation, http.FileServer on a sub-fs is easier but it needs the prefix handling.
			
			http.StripPrefix(prefix, http.FileServer(http.FS(sub))).ServeHTTP(w, r)
		})
	}

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
