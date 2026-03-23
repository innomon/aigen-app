package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/core/services"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

type AuditApi struct {
	auditService services.IAuditService
	authApi      *AuthApi
}

func NewAuditApi(auditService services.IAuditService, authApi *AuthApi) *AuditApi {
	return &AuditApi{
		auditService: auditService,
		authApi:      authApi,
	}
}

func (a *AuditApi) Register(r chi.Router) {
	r.Route("/api/auditlogs", func(r chi.Router) {
		r.Use(a.authApi.JWTMiddleware)
		r.Use(a.AdminOnlyMiddleware)
		r.Get("/", a.List)
		r.Get("/{id}", a.Get)
	})
}

func (a *AuditApi) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	strArgs := make(datamodels.StrArgs)
	for k, v := range query {
		strArgs[k] = v
	}
	parseResult := datamodels.ParseQuery(strArgs)

	logs, err := a.auditService.List(r.Context(), parseResult.Pagination)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(logs)
}

func (a *AuditApi) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	log, err := a.auditService.ById(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if log == nil {
		http.NotFound(w, r)
		return
	}
	json.NewEncoder(w).Encode(log)
}

func (a *AuditApi) AdminOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roles, _ := r.Context().Value("roles").([]string)
		isAdmin := false
		for _, role := range roles {
			if role == descriptors.RoleSa || role == descriptors.RoleAdmin {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
