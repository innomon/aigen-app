package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-app/core/services"
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
	mountGraphQL := func(base string) {
		r.Get(base, a.Playground)
		r.Get(base+"/", a.Playground)
		r.With(a.authApi.JWTMiddleware, a.authApi.RBACMiddleware("read", "graphql")).Post(base, a.Query)
		r.With(a.authApi.JWTMiddleware, a.authApi.RBACMiddleware("read", "graphql")).Post(base+"/", a.Query)
	}

	mountGraphQL("/graphql")
	mountGraphQL("/api/graphql")
	mountGraphQL("/api/schemas/graphql")
}

type graphqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func (a *GraphQLApi) Playground(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("query") != "" {
		a.authApi.JWTMiddleware(a.authApi.RBACMiddleware("read", "graphql")(http.HandlerFunc(a.Query))).ServeHTTP(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(graphqlPlaygroundHTML))
}

func (a *GraphQLApi) Query(w http.ResponseWriter, r *http.Request) {
	var req graphqlRequest
	if r.Method == http.MethodGet {
		req.Query = r.URL.Query().Get("query")
		if vars := r.URL.Query().Get("variables"); vars != "" {
			_ = json.Unmarshal([]byte(vars), &req.Variables)
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"errors": []map[string]interface{}{
					{"message": err.Error()},
				},
			})
			return
		}
	}

	result, err := a.graphqlService.Execute(r.Context(), req.Query, req.Variables)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]interface{}{
				{"message": err.Error()},
			},
		})
		return
	}

	if result == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"data": nil})
		return
	}

	json.NewEncoder(w).Encode(result)
}

const graphqlPlaygroundHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
  <title>GraphQL Playground</title>
  <link rel="stylesheet" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
  <link rel="shortcut icon" href="/favicon.ico" />
  <script src="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
  <div class="graphql-topbar">
    <div class="graphql-nav-left">
      <button type="button" onclick="goBack()" class="back-btn" title="Back">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" style="margin-right: 6px;"><line x1="19" y1="12" x2="5" y2="12"></line><polyline points="12 19 5 12 12 5"></polyline></svg>
        Back
      </button>
      <span class="nav-divider">/</span>
      <a href="/admin/dashboard.html" class="nav-link">Dashboard</a>
      <span class="nav-divider">/</span>
      <span class="nav-current">GraphQL Playground</span>
    </div>
    <div class="graphql-nav-right">
      <a href="/admin/list.html?type=query" class="nav-sublink">Saved Queries</a>
      <a href="/admin/list.html?type=entity" class="nav-sublink">Entities</a>
      <a href="/admin/index.html" class="nav-sublink">Admin</a>
    </div>
  </div>
  <div id="root">
    <style>
      body {
        background-color: #0f1c24;
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
        height: 100vh;
        margin: 0;
        overflow: hidden;
      }
      .graphql-topbar {
        height: 42px;
        background: #0f1c24;
        border-bottom: 1px solid rgba(255, 255, 255, 0.08);
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 0 16px;
        box-sizing: border-box;
        user-select: none;
        position: relative;
        z-index: 100;
        font-size: 13px;
      }
      .graphql-nav-left, .graphql-nav-right {
        display: flex;
        align-items: center;
        gap: 10px;
      }
      .back-btn {
        display: inline-flex;
        align-items: center;
        color: #e1e8ed;
        background: rgba(255, 255, 255, 0.08);
        border: 1px solid rgba(255, 255, 255, 0.15);
        border-radius: 4px;
        padding: 4px 10px;
        font-size: 13px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.15s ease;
      }
      .back-btn:hover {
        background: rgba(255, 255, 255, 0.16);
        color: #ffffff;
        border-color: rgba(255, 255, 255, 0.3);
      }
      .nav-divider {
        color: rgba(255, 255, 255, 0.3);
        font-size: 13px;
      }
      .nav-link {
        color: rgba(255, 255, 255, 0.7);
        text-decoration: none;
        font-size: 13px;
        transition: color 0.15s ease;
      }
      .nav-link:hover {
        color: #ffffff;
        text-decoration: underline;
      }
      .nav-current {
        color: #17b890;
        font-size: 13px;
        font-weight: 600;
      }
      .nav-sublink {
        color: rgba(255, 255, 255, 0.6);
        text-decoration: none;
        font-size: 12px;
        padding: 4px 8px;
        border-radius: 3px;
        transition: all 0.15s ease;
      }
      .nav-sublink:hover {
        color: #ffffff;
        background: rgba(255, 255, 255, 0.08);
      }
      #root {
        height: calc(100vh - 42px);
        width: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .loading {
        font-size: 28px;
        font-weight: 200;
        color: rgba(255, 255, 255, .7);
        margin-left: 20px;
        text-align: center;
      }
      img.playground-logo {
        width: 72px;
        height: 72px;
      }
      .title {
        font-weight: 500;
        color: #fff;
      }
    </style>
    <img class="playground-logo" src="//cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png" alt="GraphQL Logo">
    <div class="loading">Loading <span class="title">GraphQL Playground</span>...</div>
  </div>
  <script>
    function goBack() {
      if (window.history.length > 1 && document.referrer && document.referrer.indexOf(window.location.host) !== -1) {
        window.history.back();
      } else {
        window.location.href = '/admin/dashboard.html';
      }
    }

    window.addEventListener('load', function (event) {
      if (typeof GraphQLPlayground !== 'undefined') {
        GraphQLPlayground.init(document.getElementById('root'), {
          endpoint: '/api/graphql',
          settings: {
            'request.credentials': 'include',
            'schema.polling.enable': false,
            'tracing.hideTracingResponse': false,
            'tracing.tracingSupported': true
          },
          tabs: [
            {
              endpoint: '/api/graphql',
              name: 'Sample Query',
              query: "# Welcome to GraphQL Playground!\n# Queries are authenticated with your admin session cookie or Authorization header.\n\nquery {\n  CompanyList {\n    company_name\n    abbr\n  }\n}\n"
            }
          ]
        });
      } else {
        setTimeout(function() {
          var el = document.querySelector('.loading');
          if (el && !window.GraphQLPlayground) {
            el.innerHTML = 'Could not load GraphQL Playground from CDN.<br><small style="font-size:14px;color:#aaa">You can still execute queries directly via POST /api/graphql or use an external GraphQL client pointing to /graphql.</small>';
          }
        }, 3000);
      }
    });
  </script>
</body>
</html>`
