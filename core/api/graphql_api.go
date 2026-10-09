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

	result, err := a.graphqlService.Query(r.Context(), req.Query, req.Variables)
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

	var response interface{}
	if m, ok := result.(map[string]interface{}); ok {
		if _, hasData := m["data"]; hasData {
			response = result
		} else {
			response = map[string]interface{}{"data": result}
		}
	} else {
		response = map[string]interface{}{"data": result}
	}

	json.NewEncoder(w).Encode(response)
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
  <div id="root">
    <style>
      body {
        background-color: #172a3a;
        font-family: Open Sans, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif;
        height: 100vh;
        margin: 0;
        overflow: hidden;
      }
      #root {
        height: 100%;
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
      img {
        width: 72px;
        height: 72px;
      }
      .title {
        font-weight: 500;
        color: #fff;
      }
    </style>
    <img src="//cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png" alt="GraphQL Logo">
    <div class="loading">Loading <span class="title">GraphQL Playground</span>...</div>
  </div>
  <script>
    window.addEventListener('load', function (event) {
      if (typeof GraphQLPlayground !== 'undefined') {
        GraphQLPlayground.init(document.getElementById('root'), {
          endpoint: '/api/graphql',
          settings: {
            'request.credentials': 'include',
            'schema.polling.enable': false
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
