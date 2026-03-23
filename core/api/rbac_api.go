package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-cms/core/services"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

type RBACApi struct {
	entityService services.IEntityService
	authApi       *AuthApi
}

func NewRBACApi(entityService services.IEntityService, authApi *AuthApi) *RBACApi {
	return &RBACApi{
		entityService: entityService,
		authApi:       authApi,
	}
}

func (a *RBACApi) Register(r chi.Router) {
	r.Route("/api/rbac", func(r chi.Router) {
		r.Use(a.authApi.JWTMiddleware)
		r.Use(a.AdminOnlyMiddleware)

		// Roles
		r.Get("/roles", a.ListRoles)
		r.Post("/roles", a.SaveRole)
		r.Delete("/roles/{id}", a.DeleteRole)

		// Permissions
		r.Get("/permissions", a.ListPermissions)
		r.Post("/permissions", a.SavePermission)
		r.Delete("/permissions/{id}", a.DeletePermission)

		// User Roles
		r.Get("/user-roles", a.ListUserRoles)
		r.Post("/user-roles", a.SaveUserRole)
		r.Delete("/user-roles/{id}", a.DeleteUserRole)
		
		// User Permissions (Row Level)
		r.Get("/user-permissions", a.ListUserPermissions)
		r.Post("/user-permissions", a.SaveUserPermission)
		r.Delete("/user-permissions/{id}", a.DeleteUserPermission)
	})
}

func (a *RBACApi) AdminOnlyMiddleware(next http.Handler) http.Handler {
	// For RBAC management, we typically want System Managers or SA
	return a.authApi.RBACMiddleware("sa")(next) 
}

func (a *RBACApi) ListRoles(w http.ResponseWriter, r *http.Request) {
	items, total, err := a.entityService.List(r.Context(), "Role", datamodels.Pagination{}, nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"items": items, "total": total})
}

func (a *RBACApi) SaveRole(w http.ResponseWriter, r *http.Request) {
	var record datamodels.Record
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	var saved datamodels.Record
	var err error
	if id, ok := record["id"]; ok && id != nil {
		saved, err = a.entityService.Update(r.Context(), "Role", record)
	} else {
		saved, err = a.entityService.Insert(r.Context(), "Role", record)
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(saved)
}

func (a *RBACApi) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.entityService.Delete(r.Context(), "Role", id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *RBACApi) ListPermissions(w http.ResponseWriter, r *http.Request) {
	items, total, err := a.entityService.List(r.Context(), "DocPerm", datamodels.Pagination{}, nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"items": items, "total": total})
}

func (a *RBACApi) SavePermission(w http.ResponseWriter, r *http.Request) {
	var record datamodels.Record
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	var saved datamodels.Record
	var err error
	if id, ok := record["id"]; ok && id != nil {
		saved, err = a.entityService.Update(r.Context(), "DocPerm", record)
	} else {
		saved, err = a.entityService.Insert(r.Context(), "DocPerm", record)
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(saved)
}

func (a *RBACApi) DeletePermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.entityService.Delete(r.Context(), "DocPerm", id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *RBACApi) ListUserRoles(w http.ResponseWriter, r *http.Request) {
	items, total, err := a.entityService.List(r.Context(), "UserRole", datamodels.Pagination{}, nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"items": items, "total": total})
}

func (a *RBACApi) SaveUserRole(w http.ResponseWriter, r *http.Request) {
	var record datamodels.Record
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	var saved datamodels.Record
	var err error
	if id, ok := record["id"]; ok && id != nil {
		saved, err = a.entityService.Update(r.Context(), "UserRole", record)
	} else {
		saved, err = a.entityService.Insert(r.Context(), "UserRole", record)
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(saved)
}

func (a *RBACApi) DeleteUserRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.entityService.Delete(r.Context(), "UserRole", id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *RBACApi) ListUserPermissions(w http.ResponseWriter, r *http.Request) {
	items, total, err := a.entityService.List(r.Context(), "UserPermission", datamodels.Pagination{}, nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"items": items, "total": total})
}

func (a *RBACApi) SaveUserPermission(w http.ResponseWriter, r *http.Request) {
	var record datamodels.Record
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	var saved datamodels.Record
	var err error
	if id, ok := record["id"]; ok && id != nil {
		saved, err = a.entityService.Update(r.Context(), "UserPermission", record)
	} else {
		saved, err = a.entityService.Insert(r.Context(), "UserPermission", record)
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(saved)
}

func (a *RBACApi) DeleteUserPermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.entityService.Delete(r.Context(), "UserPermission", id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
