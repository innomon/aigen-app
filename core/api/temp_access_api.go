package api

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/core/services"
	"github.com/innomon/aigen-app/infrastructure/filestore"
)

type TempAccessApi struct {
	configs     []descriptors.TemporaryAccessConfig
	service     services.ITempAccessService
	filestore   filestore.IFileStore
}

func NewTempAccessApi(configs []descriptors.TemporaryAccessConfig, service services.ITempAccessService, filestore filestore.IFileStore) *TempAccessApi {
	return &TempAccessApi{
		configs:   configs,
		service:   service,
		filestore: filestore,
	}
}

func (a *TempAccessApi) Register(r chi.Router) {
	for _, c := range a.configs {
		path := c.Path
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		r.Get(path+"/{filename}", a.handleGet(c.Path))
	}
}

func (a *TempAccessApi) handleGet(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := chi.URLParam(r, "filename")
		if filename == "" {
			http.Error(w, "Filename required", http.StatusBadRequest)
			return
		}

		expired, err := a.service.IsExpired(r.Context(), path, filename)
		if err != nil {
			if strings.Contains(err.Error(), "file not found") {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if expired {
			http.Error(w, "Link has expired", http.StatusGone)
			return
		}

		fullPath := filepath.Join(path, filename)

		// Get metadata for content type
		meta, err := a.filestore.GetMetadata(r.Context(), fullPath)
		if err == nil && meta != nil {
			w.Header().Set("Content-Type", meta.ContentType)
		}

		err = a.filestore.Download(r.Context(), fullPath, w)
		if err != nil {
			http.Error(w, "Failed to download file", http.StatusInternalServerError)
		}
	}
}
