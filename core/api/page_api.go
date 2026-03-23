package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/core/services"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

type PageApi struct {
	pageService services.IPageService
	authService services.IAuthService
	authApi     *AuthApi
}

func NewPageApi(pageService services.IPageService, authService services.IAuthService, authApi *AuthApi) *PageApi {
	return &PageApi{
		pageService: pageService,
		authService: authService,
		authApi:     authApi,
	}
}

func (a *PageApi) Register(r chi.Router) {
	// Catch all for pages
	r.With(a.authApi.JWTMiddleware).Get("/*", a.Render)
}

func (a *PageApi) Render(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	query := r.URL.Query()
	strArgs := make(datamodels.StrArgs)
	for k, v := range query {
		strArgs[k] = v
	}

	html, err := a.pageService.Render(r.Context(), path, strArgs)
	if err != nil {
		if path == "/" || path == "" {
			// If we are at the root, check if the current user has a dashboard page configured
			userId, _ := r.Context().Value("userId").(int64)
			var dashboardPageId string
			
			if userId == 0 {
				// Guest user
				guestRole, err := a.authService.GetRoleByName(r.Context(), descriptors.RoleGuest)
				if err == nil && guestRole != nil && guestRole.DashboardPageId != "" {
					dashboardPageId = guestRole.DashboardPageId
				}
			} else {
				// Authenticated user
				user, err := a.authService.Me(r.Context(), userId)
				if err == nil && user != nil {
					// Use the first role's dashboard for simplicity, or the default role
					if user.DefaultRoleId != nil {
						for _, role := range user.RolesDetails {
							if role.Id == *user.DefaultRoleId && role.DashboardPageId != "" {
								dashboardPageId = role.DashboardPageId
								break
							}
						}
					}
					// If no default role dashboard found, try any
					if dashboardPageId == "" && len(user.RolesDetails) > 0 {
						for _, role := range user.RolesDetails {
							if role.DashboardPageId != "" {
								dashboardPageId = role.DashboardPageId
								break
							}
						}
					}
				}
			}

			if dashboardPageId != "" {
				// Fetch the dashboard page and render it instead of redirecting
				html, err := a.pageService.Render(r.Context(), fmt.Sprintf("/api/pages/%s", dashboardPageId), strArgs)
				if err == nil {
					w.Header().Set("Content-Type", "text/html")
					w.Write([]byte(html))
					return
				}
			}

			// Fallback: Redirect to admin interface if no dashboard found
			http.Redirect(w, r, "/admin/list.html", http.StatusFound)
			return
		}
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
