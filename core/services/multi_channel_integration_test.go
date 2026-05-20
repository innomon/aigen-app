package services_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/innomon/aigen-app/core/api"
	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/core/services"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/stretchr/testify/assert"
)

func TestA2AIntegration(t *testing.T) {
	ctx := context.Background()
	dao, _ := relationdbdao.CreateDao("memory://")
	dao.EnsureTable(ctx)
	
	schemaSvc := services.NewSchemaService(dao)
	evolutionSvc := services.NewEvolutionService(dao, schemaSvc)
	permSvc := services.NewPermissionService(dao, schemaSvc)
	entitySvc := services.NewEntityService(schemaSvc, evolutionSvc, dao, permSvc)
	intSvc := services.NewInteractionService(dao)
	commSvc := services.NewCommerceService(entitySvc)
	chatSvc, err := services.NewChatService("", entitySvc, schemaSvc, evolutionSvc, services.NewA2UIService(), intSvc, commSvc)
	assert.NoError(t, err)
	
	a2aSvc := services.NewA2AService(chatSvc, "localhost")
	
	// Setup Ed25519 Keys
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pubKeyStr := base64.RawURLEncoding.EncodeToString(pubKey)
	
	config := descriptors.ChannelsConfig{
		TrustedKeys: []descriptors.TrustedKey{
			{Id: "trusted-agent", PublicKey: pubKeyStr},
		},
	}
	
	authSvc := services.NewAuthService(dao, "secret", nil, nil)
	a2aApi := api.NewA2AApi(a2aSvc, authSvc, config)
	
	r := chi.NewRouter()
	a2aApi.Register(r)

	t.Run("Valid A2A JWT Access", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
			"iss": "trusted-agent",
			"iat": time.Now().Unix(),
		})
		tokenStr, _ := token.SignedString(privKey)
		
		// Mock A2A JSON-RPC Request
		body := `{"jsonrpc":"2.0","method":"execute","params":{"message":{"role":"user","parts":[{"text":"hello"}]}},"id":1}`
		req := httptest.NewRequest("POST", "/api/a2a", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "jsonrpc")
	})
}

func TestMCPIntegration(t *testing.T) {
	ctx := context.Background()
	dao, _ := relationdbdao.CreateDao("memory://")
	dao.EnsureTable(ctx)
	
	schemaSvc := services.NewSchemaService(dao)
	evolutionSvc := services.NewEvolutionService(dao, schemaSvc)
	permSvc := services.NewPermissionService(dao, schemaSvc)
	entitySvc := services.NewEntityService(schemaSvc, evolutionSvc, dao, permSvc)
	authSvc := services.NewAuthService(dao, "secret", nil, nil)

	// Setup MCP User with role MCP
	_, err := authSvc.Register(ctx, "mcp@test.com", "password")
	assert.NoError(t, err)
	token, err := authSvc.Login(ctx, "mcp@test.com", "password")
	assert.NoError(t, err)
	userId, _, err := authSvc.ValidateToken(token)
	assert.NoError(t, err)
	
	// Add MCP role to user
	u, err := authSvc.Me(ctx, userId)
	assert.NoError(t, err)
	assert.NotNil(t, u)
	u.Roles = []string{"MCP"}
	err = authSvc.UpdateUser(ctx, u)
	assert.NoError(t, err)

	mcpCfg := descriptors.MCPConfig{
		Enabled: true,
		APIKeys: []descriptors.APIKeyConfig{
			{Key: "test-api-key", UserId: userId},
		},
	}
	
	mcpSvc := services.NewMCPService(schemaSvc, entitySvc, authSvc, mcpCfg)
	authApi := api.NewAuthApi(authSvc, permSvc, nil)
	mcpApi := api.NewMCPApi(mcpSvc, authApi, nil, nil)
	
	r := chi.NewRouter()
	mcpApi.Register(r)

	t.Run("MCP Authentication Success", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}},"id":1}`
		req := httptest.NewRequest("POST", "/api/mcp/", strings.NewReader(body))
		req.Header.Set("X-API-Key", "test-api-key")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "protocolVersion")
	})

	t.Run("MCP Authentication Failure (Invalid Key)", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}},"id":1}`
		req := httptest.NewRequest("POST", "/api/mcp/", strings.NewReader(body))
		req.Header.Set("X-API-Key", "wrong-key")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("MCP Authentication Failure (No MCP Role)", func(t *testing.T) {
		// Ensure different ID
		time.Sleep(1100 * time.Millisecond)

		// Create another user without MCP role
		_, err := authSvc.Register(ctx, "no-mcp@test.com", "password")
		assert.NoError(t, err)
		token2, err := authSvc.Login(ctx, "no-mcp@test.com", "password")
		assert.NoError(t, err)
		u2Id, _, err := authSvc.ValidateToken(token2)
		assert.NoError(t, err)
		
		mcpCfg2 := descriptors.MCPConfig{
			Enabled: true,
			APIKeys: []descriptors.APIKeyConfig{{Key: "no-role-key", UserId: u2Id}},
		}
		mcpSvc2 := services.NewMCPService(schemaSvc, entitySvc, authSvc, mcpCfg2)
		mcpApi2 := api.NewMCPApi(mcpSvc2, authApi, nil, nil)
		r2 := chi.NewRouter()
		mcpApi2.Register(r2)

		body := `{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}},"id":1}`
		req := httptest.NewRequest("POST", "/api/mcp/", strings.NewReader(body))
		req.Header.Set("X-API-Key", "no-role-key")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		
		rr := httptest.NewRecorder()
		r2.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "MCP role")
	})
}
