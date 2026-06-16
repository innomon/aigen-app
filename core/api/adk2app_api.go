package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/core/services"
	"google.golang.org/adk/server/adkrest"
)

type ADK2AppApi struct {
	chatService       *services.ChatService
	authService       services.IAuthService
	permissionService services.IPermissionService
	whatsappService   services.IWhatsAppService
	extensionService  services.IAppExtensionService
	adkHandler        http.Handler
}

func NewADK2AppApi(
	chatService *services.ChatService,
	authService services.IAuthService,
	permissionService services.IPermissionService,
	whatsappService services.IWhatsAppService,
	extensionService services.IAppExtensionService,
) (*ADK2AppApi, error) {
	loader := services.NewAppAgentLoader(chatService, extensionService)

	server, err := adkrest.NewServer(adkrest.ServerConfig{
		SessionService:  chatService.SessionService,
		AgentLoader:     loader,
		SSEWriteTimeout: 30 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &ADK2AppApi{
		chatService:       chatService,
		authService:       authService,
		permissionService: permissionService,
		whatsappService:   whatsappService,
		extensionService:  extensionService,
		adkHandler:        server,
	}, nil
}

func (a *ADK2AppApi) Register(r chi.Router) {
	r.Route("/api/adk2app", func(r chi.Router) {
		r.Use(a.AuthenticateAndAuthorize)
		r.Mount("/", http.StripPrefix("/api/adk2app", a.adkHandler))
	})
}

func (a *ADK2AppApi) AuthenticateAndAuthorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract Bearer token
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized: missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// 2. Verify RS256 JWT from Gateway
		claims, err := a.whatsappService.VerifyADKJWT(tokenString)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// 3. Extract appName and userID from request
		appName := ""
		userID := ""

		if strings.Contains(r.URL.Path, "/apps/") {
			appName, userID = parseSessionPath(r.URL.Path)
		} else if r.URL.Path == "/api/adk2app/run" || r.URL.Path == "/api/adk2app/run_sse" {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			var runReq struct {
				AppName string `json:"appName"`
				UserID  string `json:"userId"`
			}
			if err := json.Unmarshal(bodyBytes, &runReq); err == nil {
				appName = runReq.AppName
				userID = runReq.UserID
			}
		}

		if appName == "" || userID == "" {
			http.Error(w, "Bad Request: missing appName or userId", http.StatusBadRequest)
			return
		}

		// 4. Prevent impersonation (token user_id must match request's userID)
		if claims.UserID != userID {
			http.Error(w, "Forbidden: user ID mismatch", http.StatusForbidden)
			return
		}

		// 5. Resolve user and roles in aigen-app
		token, err := a.authService.LoginByChannel(r.Context(), descriptors.ChannelWhatsApp, claims.UserID, "", r.RemoteAddr, r.UserAgent())
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
		userId, roles, err := a.authService.ValidateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// 6. Enforce RBAC for appName
		hasAccess, err := a.permissionService.HasAccess(r.Context(), userId, roles, appName, "read")
		if err != nil {
			http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !hasAccess {
			http.Error(w, "Forbidden: access denied for agent "+appName, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func parseSessionPath(path string) (string, string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 6 && parts[0] == "api" && parts[1] == "adk2app" && parts[2] == "apps" && parts[4] == "users" {
		return parts[3], parts[5]
	}
	return "", ""
}
