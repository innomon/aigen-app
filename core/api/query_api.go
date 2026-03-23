package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-cms/core/services"
)

type QueryApi struct {
	graphqlService services.IGraphQLService
	authApi        *AuthApi
}

func NewQueryApi(graphqlService services.IGraphQLService, authApi *AuthApi) *QueryApi {
	return &QueryApi{
		graphqlService: graphqlService,
		authApi:        authApi,
	}
}

func (a *QueryApi) Register(r chi.Router) {
	r.Route("/api/queries/{name}", func(r chi.Router) {
		r.Use(a.authApi.JWTMiddleware)
		r.Use(a.authApi.RBACMiddleware("read", "queries"))
		r.Get("/", a.Execute)
		r.Post("/", a.ExecuteWithBody)
	})
}

func (a *QueryApi) Execute(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	
	// Convert URL query params to variables
	variables := make(map[string]interface{})
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			variables[k] = v[0]
		}
	}

	result, err := a.graphqlService.ExecuteStoredQuery(r.Context(), name, variables)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (a *QueryApi) ExecuteWithBody(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	
	var variables map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&variables); err != nil {
		// Fallback if body is empty or not valid JSON
		variables = make(map[string]interface{})
	}

	result, err := a.graphqlService.ExecuteStoredQuery(r.Context(), name, variables)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}
