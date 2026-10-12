package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/core/services"
)

type SchemaApi struct {
	schemaService services.ISchemaService
	authApi       *AuthApi
}

func NewSchemaApi(schemaService services.ISchemaService, authApi *AuthApi) *SchemaApi {
	return &SchemaApi{
		schemaService: schemaService,
		authApi:       authApi,
	}
}

func (a *SchemaApi) Register(r chi.Router) {
	r.Route("/api/schemas", func(r chi.Router) {
		r.Use(a.authApi.JWTMiddleware)

		r.Get("/graphql", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/graphql", http.StatusFound)
		})
		r.Get("/graphql/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/graphql", http.StatusFound)
		})

		r.Get("/", a.GetAll)
		r.Get("/name/{name}", a.GetByName)
		r.Get("/history/{schemaId}", a.GetHistory)
		r.Get("/{schemaId}", a.GetBySchemaId)

		r.Group(func(r chi.Router) {
			r.Use(a.AdminOnlyMiddleware)
			r.Post("/", a.Save)
			r.Post("/publish", a.Publish)
			r.Post("/unpublish", a.Unpublish)
			r.Delete("/{schemaId}", a.Delete)
		})
	})
}

func (a *SchemaApi) AdminOnlyMiddleware(next http.Handler) http.Handler {
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

func parseSchemaType(typeName string) descriptors.SchemaType {
	switch strings.ToLower(typeName) {
	case "entity":
		return descriptors.EntitySchema
	case "menu":
		return descriptors.MenuSchema
	case "query":
		return descriptors.QuerySchema
	case "page":
		return descriptors.PageSchema
	default:
		return descriptors.SchemaType(typeName)
	}
}

func (a *SchemaApi) GetAll(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	typeName := query.Get("type")
	statusName := query.Get("status")

	var schemaType *descriptors.SchemaType
	if typeName != "" {
		st := parseSchemaType(typeName)
		schemaType = &st
	}

	var status *descriptors.PublicationStatus
	if statusName != "" {
		ps := descriptors.PublicationStatus(statusName)
		status = &ps
	}

	names := query["names"]

	schemas, err := a.schemaService.All(r.Context(), schemaType, names, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schemas)
}

func (a *SchemaApi) GetByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	query := r.URL.Query()
	typeName := query.Get("type")
	statusName := query.Get("status")

	var schemaType descriptors.SchemaType = descriptors.EntitySchema
	if typeName != "" {
		schemaType = parseSchemaType(typeName)
	}

	var status *descriptors.PublicationStatus
	if statusName != "" {
		ps := descriptors.PublicationStatus(statusName)
		status = &ps
	}

	schema, err := a.schemaService.ByNameOrDefault(r.Context(), name, schemaType, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if schema == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schema)
}

func (a *SchemaApi) GetBySchemaId(w http.ResponseWriter, r *http.Request) {
	schemaId := chi.URLParam(r, "schemaId")
	
	// If it's a numeric ID, try fetching by primary key
	if id, err := strconv.ParseInt(schemaId, 10, 64); err == nil {
		schema, err := a.schemaService.ById(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if schema == nil {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(schema)
		return
	}

	schema, err := a.schemaService.BySchemaId(r.Context(), schemaId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if schema == nil {
		http.NotFound(w, r)
		return
	}

	json.NewEncoder(w).Encode(schema)
}

func (a *SchemaApi) GetHistory(w http.ResponseWriter, r *http.Request) {
	schemaId := chi.URLParam(r, "schemaId")

	schema, err := a.schemaService.BySchemaId(r.Context(), schemaId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if schema == nil {
		if id, err := strconv.ParseInt(schemaId, 10, 64); err == nil {
			schema, _ = a.schemaService.ById(r.Context(), id)
		}
	}
	if schema == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]*descriptors.Schema{schema})
}

func (a *SchemaApi) Save(w http.ResponseWriter, r *http.Request) {
	var schema descriptors.Schema
	if err := json.NewDecoder(r.Body).Decode(&schema); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	asPublished := r.URL.Query().Get("asPublished") == "true" || r.URL.Query().Get("publish") == "true"
	
	savedSchema, err := a.schemaService.Save(r.Context(), &schema, asPublished)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(savedSchema)
}

type SchemaActionRequest struct {
	Id       interface{} `json:"id"`
	SchemaId string      `json:"schemaId"`
}

func (a *SchemaApi) resolveSchema(ctx context.Context, schemaId string, id interface{}) (*descriptors.Schema, error) {
	if schemaId != "" {
		schema, err := a.schemaService.BySchemaId(ctx, schemaId)
		if err == nil && schema != nil {
			return schema, nil
		}
	}
	if id != nil {
		switch v := id.(type) {
		case float64:
			return a.schemaService.ById(ctx, int64(v))
		case int64:
			return a.schemaService.ById(ctx, v)
		case int:
			return a.schemaService.ById(ctx, int64(v))
		case string:
			if v != "" {
				if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
					return a.schemaService.ById(ctx, intVal)
				}
				return a.schemaService.BySchemaId(ctx, v)
			}
		}
	}
	return nil, nil
}

func (a *SchemaApi) Publish(w http.ResponseWriter, r *http.Request) {
	var req SchemaActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	schema, err := a.resolveSchema(r.Context(), req.SchemaId, req.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if schema == nil {
		http.NotFound(w, r)
		return
	}

	savedSchema, err := a.schemaService.Save(r.Context(), schema, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(savedSchema)
}

func (a *SchemaApi) Unpublish(w http.ResponseWriter, r *http.Request) {
	var req SchemaActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	schema, err := a.resolveSchema(r.Context(), req.SchemaId, req.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if schema == nil {
		http.NotFound(w, r)
		return
	}

	if schema.Id == 0 {
		schema.Id = time.Now().UnixNano()
	}

	savedSchema, err := a.schemaService.Save(r.Context(), schema, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(savedSchema)
}

func (a *SchemaApi) Delete(w http.ResponseWriter, r *http.Request) {
	schemaId := chi.URLParam(r, "schemaId")
	if id, err := strconv.ParseInt(schemaId, 10, 64); err == nil {
		if schema, err := a.schemaService.ById(r.Context(), id); err == nil && schema != nil {
			schemaId = schema.SchemaId
		}
	}
	if err := a.schemaService.Delete(r.Context(), schemaId); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

