package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/core/services"
)

type EngagementApi struct {
	engagementService services.IEngagementService
	authApi           *AuthApi
}

func NewEngagementApi(engagementService services.IEngagementService, authApi *AuthApi) *EngagementApi {
	return &EngagementApi{
		engagementService: engagementService,
		authApi:           authApi,
	}
}

func (a *EngagementApi) Register(r chi.Router) {
	r.Route("/api/engagements", func(r chi.Router) {
		r.Use(a.authApi.JWTMiddleware)
		r.Use(a.authApi.RBACMiddleware("write", "engagements"))
		r.Post("/track", a.Track)
	})
}

func (a *EngagementApi) Track(w http.ResponseWriter, r *http.Request) {
	var status descriptors.EngagementStatus
	if err := json.NewDecoder(r.Body).Decode(&status); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// In a real app, get userId from auth or cookie
	if status.UserId == "" {
		status.UserId = "anonymous"
	}

	if err := a.engagementService.Track(r.Context(), &status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
