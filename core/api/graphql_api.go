package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-cms/core/services"
)

type GraphQLApi struct {
	graphqlService services.IGraphQLService
	authApi        *AuthApi
}

func NewGraphQLApi(graphqlService services.IGraphQLService, authApi *AuthApi) *GraphQLApi {
	return &GraphQLApi{
		graphqlService: graphqlService,
		authApi:        authApi,
	}
}

func (a *GraphQLApi) Register(r chi.Router) {
	r.Route("/api/graphql", func(r chi.Router) {
		r.Use(a.authApi.JWTMiddleware)
		r.Use(a.authApi.RBACMiddleware("read", "graphql"))
		r.Post("/", a.Query)
	})
}

type graphqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func (a *GraphQLApi) Query(w http.ResponseWriter, r *http.Request) {
	var req graphqlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := a.graphqlService.Query(r.Context(), req.Query, req.Variables)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}
