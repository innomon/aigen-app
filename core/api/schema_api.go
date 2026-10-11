package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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
		r.Get("/{schemaId}", a.GetBySchemaId)

		r.Group(func(r chi.Router) {
			r.Use(a.AdminOnlyMiddleware)
			r.Post("/", a.Save)
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

func (a *SchemaApi) Save(w http.ResponseWriter, r *http.Request) {
	var schema descriptors.Schema
	if err := json.NewDecoder(r.Body).Decode(&schema); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	asPublished := r.URL.Query().Get("asPublished") == "true"
	
	savedSchema, err := a.schemaService.Save(r.Context(), &schema, asPublished)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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
