package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/innomon/aigen-cms/core/services"
)

type MCPApi struct {
	mcpService services.IMCPService
}

func NewMCPApi(mcpService services.IMCPService) *MCPApi {
	return &MCPApi{
		mcpService: mcpService,
	}
}

func (a *MCPApi) Register(r chi.Router) {
	// MCP SSE handler
	// NewStreamableHTTPHandler creates a handler for the MCP SSE transport.
	// It requires an authentication middleware to identify the user/role.
	
	r.Route("/api/mcp", func(r chi.Router) {
		r.Use(a.Authenticate)
		
		handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
			return a.mcpService.GetServer()
		}, nil)
		
		r.Handle("/", handler)
		r.Handle("/sse", handler)
	})
}

func (a *MCPApi) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("apiKey")
		}

		if apiKey == "" {
			http.Error(w, "Unauthorized: API Key missing", http.StatusUnauthorized)
			return
		}

		// Verify API Key via MCP Service
		if mcpSvc, ok := a.mcpService.(*services.MCPService); ok {
			userId, roles, err := mcpSvc.Authenticate(r.Context(), apiKey)
			if err != nil {
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Add to context
			ctx := context.WithValue(r.Context(), "userId", userId)
			ctx = context.WithValue(ctx, "roles", roles)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}
