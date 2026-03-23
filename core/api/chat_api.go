package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-cms/core/services"
)

type ChatApi struct {
	chatService *services.ChatService
	authApi     *AuthApi
}

func NewChatApi(chatService *services.ChatService, authApi *AuthApi) *ChatApi {
	return &ChatApi{
		chatService: chatService,
		authApi:     authApi,
	}
}

func (a *ChatApi) Register(r chi.Router) {
	r.Route("/api/chat", func(r chi.Router) {
		r.Use(a.authApi.JWTMiddleware)
		r.Use(a.authApi.RBACMiddleware("read", "chat"))
		r.Post("/message", a.Message)
	})
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

func (a *ChatApi) Message(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	
	resp, err := a.chatService.ProcessMessage(ctx, req.Message, nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ChatResponse{Error: fmt.Sprintf("agent error: %v", err)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ChatResponse{Response: resp})
}
