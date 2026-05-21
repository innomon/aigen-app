package api

import (
	"archive/zip"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/innomon/aigen-app/core/plugins"
	"github.com/go-chi/chi/v5"
)


//go:embed all:ui
var uiAssets embed.FS

type StaticApi struct {
	wwwRoot       string
	customUIPath  string
	filesPrefix   string
	pluginService *plugins.PluginService
}

func NewStaticApi(wwwRoot, customUIPath, filesPrefix string, pluginService *plugins.PluginService) *StaticApi {
	return &StaticApi{wwwRoot: wwwRoot, customUIPath: customUIPath, filesPrefix: filesPrefix, pluginService: pluginService}
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

	// Serve from plugin JARs
	if a.pluginService != nil {
		r.Get("/_plugins/{id}/*", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			info, ok := a.pluginService.Get(id)
			if !ok || info.Status != plugins.StatusActive {
				http.NotFound(w, r)
				return
			}

			// Open the JAR and serve from wwwroot/
			jar, err := zip.OpenReader(info.Path)
			if err != nil {
				http.Error(w, "failed to open plugin", http.StatusInternalServerError)
				return
			}
			defer jar.Close()

			sub, err := fs.Sub(jar, "wwwroot")
			if err != nil {
				http.NotFound(w, r)
				return
			}

			// Trim the prefix /_plugins/{id}/
			prefix := "/_plugins/" + id + "/"
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
