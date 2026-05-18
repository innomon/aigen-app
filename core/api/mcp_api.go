package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-app/core/services"
	"github.com/innomon/aigen-app/infrastructure/filestore"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type MCPApi struct {
	mcpService        services.IMCPService
	authApi           *AuthApi
	tempAccessService services.ITempAccessService
	filestore         filestore.IFileStore
}

func NewMCPApi(mcpService services.IMCPService, authApi *AuthApi, tempAccessService services.ITempAccessService, filestore filestore.IFileStore) *MCPApi {
	return &MCPApi{
		mcpService:        mcpService,
		authApi:           authApi,
		tempAccessService: tempAccessService,
		filestore:         filestore,
	}
}

func (a *MCPApi) Register(r chi.Router) {
	r.Route("/api/mcp", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(a.Authenticate)

			handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
				return a.mcpService.GetServer()
			}, nil)

			r.Handle("/", handler)
			r.Handle("/sse", handler)
		})

		// Management endpoints for temporary access
		r.Route("/temp-access/{path}", func(r chi.Router) {
			r.Use(a.authApi.JWTMiddleware)
			r.Post("/upload", a.HandleUpload)
			r.Delete("/{filename}", a.HandleDelete)
			r.Post("/cleanup", a.HandleCleanup)
		})
	})
}

func (a *MCPApi) HandleCleanup(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")
	rolesInterface := r.Context().Value("roles")
	if rolesInterface == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	roles := rolesInterface.([]string)

	// Check RBAC
	if mcpSvc, ok := a.tempAccessService.(*services.TempAccessService); ok {
		config := mcpSvc.GetConfig(path)
		if config == nil {
			http.Error(w, "Invalid temporary access path", http.StatusBadRequest)
			return
		}

		hasRole := false
		for _, r := range roles {
			if r == config.Role {
				hasRole = true
				break
			}
		}
		if !hasRole {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	} else {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	count, err := a.tempAccessService.CleanupExpired(r.Context(), path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cleanup failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Cleanup successful: %d files deleted", count)
}

func (a *MCPApi) HandleUpload(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")
	rolesInterface := r.Context().Value("roles")
	if rolesInterface == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	roles := rolesInterface.([]string)

	// Check RBAC
	if mcpSvc, ok := a.tempAccessService.(*services.TempAccessService); ok {
		config := mcpSvc.GetConfig(path)
		if config == nil {
			http.Error(w, "Invalid temporary access path", http.StatusBadRequest)
			return
		}

		hasRole := false
		for _, r := range roles {
			if r == config.Role {
				hasRole = true
				break
			}
		}
		if !hasRole {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	} else {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Handle upload
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file from request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename := header.Filename
	fullPath := filepath.Join(path, filename)

	err = a.filestore.Upload(r.Context(), fullPath, file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to upload file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "File uploaded successfully: %s", fullPath)
}

func (a *MCPApi) HandleDelete(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")
	filename := chi.URLParam(r, "filename")
	rolesInterface := r.Context().Value("roles")
	if rolesInterface == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	roles := rolesInterface.([]string)

	// Check RBAC
	if mcpSvc, ok := a.tempAccessService.(*services.TempAccessService); ok {
		config := mcpSvc.GetConfig(path)
		if config == nil {
			http.Error(w, "Invalid temporary access path", http.StatusBadRequest)
			return
		}

		hasRole := false
		for _, r := range roles {
			if r == config.Role {
				hasRole = true
				break
			}
		}
		if !hasRole {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	} else {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	fullPath := filepath.Join(path, filename)
	err := a.filestore.Delete(r.Context(), fullPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File deleted successfully: %s", fullPath)
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
