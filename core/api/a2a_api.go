package api

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/core/services"
)

type A2AApi struct {
	a2aService services.IA2AService
	authService services.IAuthService
	config      descriptors.ChannelsConfig
}

func NewA2AApi(a2aService services.IA2AService, authService services.IAuthService, config descriptors.ChannelsConfig) *A2AApi {
	return &A2AApi{
		a2aService:  a2aService,
		authService: authService,
		config:      config,
	}
}

func (a *A2AApi) Register(r chi.Router) {
	handler := a2asrv.NewJSONRPCHandler(a.a2aService.GetHandler())
	
	r.Handle("/api/a2a", a.HandleA2A(handler))
	r.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(a.a2aService.GetAgentCard()))
}

func (a *A2AApi) HandleA2A(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// A2A Ed25519 JWT Auth
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			
			// Verify token against trusted channel keys
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				
				claims := token.Claims.(jwt.MapClaims)
				iss := claims["iss"].(string)
				
				// Find public key for issuer (channel)
				for _, tk := range a.config.TrustedKeys {
					if tk.Id == iss {
						pubKeyBytes, err := base64.RawURLEncoding.DecodeString(tk.PublicKey)
						if err != nil {
							return nil, err
						}
						return ed25519.PublicKey(pubKeyBytes), nil
					}
				}
				
				return nil, fmt.Errorf("untrusted A2A issuer: %s", iss)
			})

			if err == nil && token.Valid {
				// Add A2A claims to context
				claims := token.Claims.(jwt.MapClaims)
				ctx := context.WithValue(r.Context(), "a2a_claims", claims)
				r = r.WithContext(ctx)
			}
		}

		next.ServeHTTP(w, r)
	}
}
