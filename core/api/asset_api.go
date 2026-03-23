package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-cms/core/services"
)

type AssetApi struct {
	assetService *services.AssetService
}

func NewAssetApi(assetService *services.AssetService) *AssetApi {
	return &AssetApi{assetService: assetService}
}

func (a *AssetApi) Register(r chi.Router) {
	r.Route("/api/assets", func(r chi.Router) {
		r.Get("/chunk-status", a.GetChunkStatus)
		r.Post("/upload-chunk", a.UploadChunk)
		r.Post("/commit-chunks", a.CommitChunks)
	})
}

func (a *AssetApi) GetChunkStatus(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("fileName")
	fileSizeStr := r.URL.Query().Get("fileSize")
	fileSize, _ := strconv.ParseInt(fileSizeStr, 10, 64)

	// In a real app, get userId from context/auth
	userId := "admin"

	status, err := a.assetService.ChunkStatus(r.Context(), userId, fileName, fileSize)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(status)
}

func (a *AssetApi) UploadChunk(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	chunkNumberStr := r.URL.Query().Get("chunkNumber")
	chunkNumber, _ := strconv.Atoi(chunkNumberStr)

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	err = a.assetService.UploadChunk(r.Context(), path, chunkNumber, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *AssetApi) CommitChunks(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	fileName := r.URL.Query().Get("fileName")

	savedAsset, err := a.assetService.CommitChunks(r.Context(), path, fileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(savedAsset)
}
