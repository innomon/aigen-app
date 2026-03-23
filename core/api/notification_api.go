package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-cms/core/services"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

type NotificationApi struct {
	notificationService services.INotificationService
	authApi             *AuthApi
}

func NewNotificationApi(notificationService services.INotificationService, authApi *AuthApi) *NotificationApi {
	return &NotificationApi{
		notificationService: notificationService,
		authApi:             authApi,
	}
}

func (a *NotificationApi) Register(r chi.Router) {
	r.Route("/api/notifications", func(r chi.Router) {
		r.Use(a.authApi.JWTMiddleware)
		r.Get("/", a.List)
		r.Put("/{id}/read", a.MarkAsRead)
		r.Put("/read-all", a.MarkAllAsRead)
	})
}

func (a *NotificationApi) List(w http.ResponseWriter, r *http.Request) {
	userId := fmt.Sprintf("%v", r.Context().Value("userId"))
	
	query := r.URL.Query()
	strArgs := make(datamodels.StrArgs)
	for k, v := range query {
		strArgs[k] = v
	}
	parseResult := datamodels.ParseQuery(strArgs)

	notifications, err := a.notificationService.List(r.Context(), userId, parseResult.Pagination)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(notifications)
}

func (a *NotificationApi) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userId := fmt.Sprintf("%v", r.Context().Value("userId"))
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	if err := a.notificationService.MarkAsRead(r.Context(), userId, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *NotificationApi) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	userId := fmt.Sprintf("%v", r.Context().Value("userId"))

	if err := a.notificationService.MarkAllAsRead(r.Context(), userId); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
