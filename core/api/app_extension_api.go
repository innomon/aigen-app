package api

import (
	"encoding/json"
	"net/http"

	"github.com/innomon/aigen-app/core/app_extensions"
	"github.com/go-chi/chi/v5"
)

type AppExtensionApi struct {
	Service *app_extensions.AppExtensionService
	Auth    *AuthApi
}

func NewAppExtensionApi(service *app_extensions.AppExtensionService, auth *AuthApi) *AppExtensionApi {
	return &AppExtensionApi{Service: service, Auth: auth}
}

func (a *AppExtensionApi) Register(r chi.Router) {
	r.Route("/api/app-extensions", func(r chi.Router) {
		r.Use(a.Auth.JWTMiddleware)
		r.Use(a.Auth.RequireAdmin)
		r.Get("/", a.listExtensions)
		r.Post("/{id}/mount", a.mountExtension)
		r.Post("/{id}/authorize", a.authorizePermission)
		r.Post("/{id}/secrets", a.setSecret)
	})
}

func (a *AppExtensionApi) listExtensions(w http.ResponseWriter, r *http.Request) {
	list := a.Service.All()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (a *AppExtensionApi) mountExtension(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.Service.MountExtension(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("App Extension mounted successfully"))
}

func (a *AppExtensionApi) authorizePermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req app_extensions.PermissionRequirement
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	adminID := "admin" // In a real app, get from context
	if err := a.Service.AuthorizePermission(r.Context(), id, req, adminID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Permission authorized"))
}

func (a *AppExtensionApi) setSecret(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	a.Service.SetSecret(id, req.Key, req.Value)
	w.Write([]byte("Secret saved successfully"))
}
