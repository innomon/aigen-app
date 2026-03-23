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

type CommentApi struct {
	commentService services.ICommentService
	authApi        *AuthApi
}

func NewCommentApi(commentService services.ICommentService, authApi *AuthApi) *CommentApi {
	return &CommentApi{
		commentService: commentService,
		authApi:        authApi,
	}
}

func (a *CommentApi) Register(r chi.Router) {
	r.Route("/api/comments", func(r chi.Router) {
		r.Use(a.authApi.JWTMiddleware)
		
		r.With(a.authApi.RBACMiddleware("read", "comments")).Get("/{entityName}/{recordId}", a.List)
		r.With(a.authApi.RBACMiddleware("write", "comments")).Post("/", a.Save)
		r.With(a.authApi.RBACMiddleware("delete", "comments")).Delete("/{id}", a.Delete)
	})
}

func (a *CommentApi) List(w http.ResponseWriter, r *http.Request) {
	entityName := chi.URLParam(r, "entityName")
	recordIdStr := chi.URLParam(r, "recordId")
	recordId, _ := strconv.ParseInt(recordIdStr, 10, 64)

	query := r.URL.Query()
	strArgs := make(datamodels.StrArgs)
	for k, v := range query {
		strArgs[k] = v
	}
	parseResult := datamodels.ParseQuery(strArgs)

	comments, err := a.commentService.List(r.Context(), entityName, recordId, parseResult.Pagination)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(comments)
}

func (a *CommentApi) Save(w http.ResponseWriter, r *http.Request) {
	var comment descriptors.Comment
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// In a real app, get userId from auth
	comment.CreatedBy = "admin"

	savedComment, err := a.commentService.Save(r.Context(), &comment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(savedComment)
}

func (a *CommentApi) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// In a real app, get userId from auth
	userId := "admin"

	if err := a.commentService.Delete(r.Context(), userId, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
