package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/core/services"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

type ChannelApi struct {
	channelService services.IChannelService
	authApi        *AuthApi
}

func NewChannelApi(channelService services.IChannelService, authApi *AuthApi) *ChannelApi {
	return &ChannelApi{
		channelService: channelService,
		authApi:        authApi,
	}
}

func (a *ChannelApi) Register(r chi.Router) {
	r.Route("/api/channels", func(r chi.Router) {
		// Public webhook for external ADKs (whatsADK, mailADK, etc.)
		r.Post("/webhook/{type}", a.HandleWebhook)

		// Protected routes for users
		r.Group(func(r chi.Router) {
			r.Use(a.authApi.JWTMiddleware)
			r.Post("/", a.RegisterChannel)
			r.Post("/verify", a.VerifyChannel)
			r.Get("/", a.GetMyChannels)
			r.Get("/logs", a.GetMyAuthLogs)
		})
	})
}

func (a *ChannelApi) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	channelType := descriptors.ChannelType(chi.URLParam(r, "type"))
	
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// For MVP: Simple identifier extraction. 
	// In production, you would verify a shared secret or API key here.
	identifier, _ := payload["from"].(string)

	err := a.channelService.HandleInbound(r.Context(), channelType, identifier, payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type registerChannelRequest struct {
	ChannelType descriptors.ChannelType `json:"channelType"`
	Identifier  string                  `json:"identifier"`
	Metadata    map[string]interface{}  `json:"metadata"`
}

func (a *ChannelApi) RegisterChannel(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value("userId").(int64)
	if userId == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req registerChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	channel, err := a.channelService.RegisterChannel(r.Context(), userId, req.ChannelType, req.Identifier, req.Metadata)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(channel)
}

type verifyChannelRequest struct {
	ChannelType descriptors.ChannelType `json:"channelType"`
	Token       string                  `json:"token"`
}

func (a *ChannelApi) VerifyChannel(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value("userId").(int64)
	if userId == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req verifyChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	success, err := a.channelService.VerifyChannel(r.Context(), userId, req.ChannelType, req.Token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": success})
}

func (a *ChannelApi) GetMyChannels(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value("userId").(int64)
	if userId == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	channels, err := a.channelService.GetChannelsByUserId(r.Context(), userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(channels)
}

func (a *ChannelApi) GetMyAuthLogs(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value("userId").(int64)
	if userId == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	pagination := datamodels.Pagination{}
	if limitStr != "" {
		pagination.Limit = &limitStr
	}
	if offsetStr != "" {
		pagination.Offset = &offsetStr
	}

	logs, total, err := a.channelService.GetAuthLogs(r.Context(), userId, pagination)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  logs,
		"total": total,
	})
}
