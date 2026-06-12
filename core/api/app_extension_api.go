package api

import (
	"encoding/json"
	"net/http"

	"github.com/innomon/aigen-app/core/plugins"
	"github.com/go-chi/chi/v5"
)

type PluginApi struct {
	Service *plugins.PluginService
	Auth    *AuthApi
}

func NewPluginApi(service *plugins.PluginService, auth *AuthApi) *PluginApi {
	return &PluginApi{Service: service, Auth: auth}
}

func (a *PluginApi) Register(r chi.Router) {
	r.Route("/api/plugins", func(r chi.Router) {
		r.Use(a.Auth.JWTMiddleware)
		r.Use(a.Auth.RequireAdmin)
		r.Get("/", a.listPlugins)
		r.Post("/{id}/mount", a.mountPlugin)
		r.Post("/{id}/authorize", a.authorizePermission)
		r.Post("/{id}/secrets", a.setSecret)
	})
}

func (a *PluginApi) listPlugins(w http.ResponseWriter, r *http.Request) {
	list := a.Service.All()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (a *PluginApi) mountPlugin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.Service.MountPlugin(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Plugin mounted successfully"))
}

func (a *PluginApi) authorizePermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req plugins.PermissionRequirement
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

func (a *PluginApi) setSecret(w http.ResponseWriter, r *http.Request) {
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
