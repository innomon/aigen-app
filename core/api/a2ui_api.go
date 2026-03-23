package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-cms/core/services"
)

type A2UIApi struct {
	a2uiService services.IA2UIService
	authApi     *AuthApi
}

func NewA2UIApi(a2uiService services.IA2UIService, authApi *AuthApi) *A2UIApi {
	return &A2UIApi{
		a2uiService: a2uiService,
		authApi:     authApi,
	}
}

func (a *A2UIApi) Register(r chi.Router) {
	r.Route("/api/a2ui", func(r chi.Router) {
		r.Use(a.authApi.JWTMiddleware)
		r.Use(a.authApi.RBACMiddleware("read", "a2ui"))
		r.Get("/stream", a.Stream)
		r.Post("/action", a.Action)
	})
}

func (a *A2UIApi) Stream(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Flush the headers
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	ch, subID := a.a2uiService.Subscribe()
	defer a.a2uiService.Unsubscribe(subID)

	// Keep the connection open
	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			return
		case components := <-ch:
			data, err := json.Marshal(components)
			if err != nil {
				fmt.Fprintf(w, "error: %v\n\n", err)
			} else {
				fmt.Fprintf(w, "data: %s\n\n", data)
			}
			flusher.Flush()
		}
	}
}

type UserAction struct {
	ComponentID string                 `json:"componentId"`
	ActionType  string                 `json:"actionType"`
	Data        map[string]interface{} `json:"data"`
}

func (a *A2UIApi) Action(w http.ResponseWriter, r *http.Request) {
	var action UserAction
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Simple logic for the prototype: 
	// If it's a "counter" action, increment the counter in the service
	if action.ActionType == "increment" {
		ctx := r.Context()
		c, ok := a.a2uiService.GetComponent(action.ComponentID)
		if ok && c.Type == "Button" {
			// Find the text component to update
			// For this prototype, let's assume there's a "counter-text"
			textComp, ok := a.a2uiService.GetComponent("counter-text")
			if ok {
				count := textComp.Attributes["count"].(float64)
				count++
				textComp.Attributes["count"] = count
				textComp.Attributes["content"] = fmt.Sprintf("Count: %v", count)
				a.a2uiService.UpdateComponent(ctx, textComp)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}
