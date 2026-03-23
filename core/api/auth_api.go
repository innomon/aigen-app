package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-cms/core/services"
	"github.com/innomon/aigen-cms/core/descriptors"
)

type AuthApi struct {
	authService       services.IAuthService
	permissionService services.IPermissionService
}

func NewAuthApi(authService services.IAuthService, permissionService services.IPermissionService) *AuthApi {
	return &AuthApi{
		authService:       authService,
		permissionService: permissionService,
	}
}

func (a *AuthApi) Register(r chi.Router) {
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", a.DoRegister)
		r.Post("/login", a.DoLogin)
		r.Post("/login/channel", a.DoLoginByChannel)
		r.With(a.JWTMiddleware).Get("/me", a.GetMe)
	})

	// Add routes expected by frontend
	r.With(a.JWTMiddleware).Get("/api/me", a.GetMe)
	r.Get("/api/logout", a.DoLogout)
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *AuthApi) DoRegister(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := a.authService.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(user)
}

func (a *AuthApi) DoLogin(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := a.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
	})

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (a *AuthApi) DoLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusOK)
}

func (a *AuthApi) GetMe(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value("userId").(int64)
	if userId == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := a.authService.Me(r.Context(), userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(user)
}

func (a *AuthApi) JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := ""

		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			cookie, err := r.Cookie("token")
			if err == nil {
				tokenString = cookie.Value
			}
		}

		var userId int64
		var roles []string
		var err error

		if tokenString == "" {
			userId = 0
			roles = []string{descriptors.RoleGuest}
		} else {
			userId, roles, err = a.authService.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
		}

		ctx := context.WithValue(r.Context(), "userId", userId)
		ctx = context.WithValue(ctx, "roles", roles)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type channelAuthRequest struct {
	ChannelType descriptors.ChannelType `json:"channelType"`
	Identifier  string                  `json:"identifier"`
	Token       string                  `json:"token"`
}

func (a *AuthApi) DoLoginByChannel(w http.ResponseWriter, r *http.Request) {
	var req channelAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ip := r.RemoteAddr
	ua := r.UserAgent()

	token, err := a.authService.LoginByChannel(r.Context(), req.ChannelType, req.Identifier, req.Token, ip, ua)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
	})

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (a *AuthApi) RBACMiddleware(action string, explicitResource ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userId, _ := r.Context().Value("userId").(int64)
			roles, _ := r.Context().Value("roles").([]string)

			resourceName := ""
			if len(explicitResource) > 0 {
				resourceName = explicitResource[0]
			} else {
				resourceName = chi.URLParam(r, "name")
			}

			if resourceName == "" {
				// If not entity-based route, maybe we can't check here
				next.ServeHTTP(w, r)
				return
			}

			hasAccess, err := a.permissionService.HasAccess(r.Context(), userId, roles, resourceName, action)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if !hasAccess {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
