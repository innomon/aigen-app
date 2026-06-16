package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"iter"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/core/services"
	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/innomon/agentic/pkg/config"
	"github.com/innomon/agentic/pkg/registry"
	"github.com/stretchr/testify/assert"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type mockAuthService struct {
	validateTokenFn func(token string) (int64, []string, error)
	loginByChannelFn func(ctx context.Context, channelType descriptors.ChannelType, identifier string, token string, ip, ua string) (string, error)
}

func (m *mockAuthService) ValidateToken(token string) (int64, []string, error) {
	if m.validateTokenFn != nil {
		return m.validateTokenFn(token)
	}
	return 42, []string{"user"}, nil
}

func (m *mockAuthService) LoginByChannel(ctx context.Context, channelType descriptors.ChannelType, identifier string, token string, ip, ua string) (string, error) {
	if m.loginByChannelFn != nil {
		return m.loginByChannelFn(ctx, channelType, identifier, token, ip, ua)
	}
	return "mocked-user-token", nil
}

func (m *mockAuthService) Register(ctx context.Context, email, password string) (*descriptors.User, error) {
	return nil, nil
}
func (m *mockAuthService) Login(ctx context.Context, email, password string) (string, error) {
	return "", nil
}
func (m *mockAuthService) LinkChannel(ctx context.Context, userId int64, channelType descriptors.ChannelType, identifier string) error {
	return nil
}
func (m *mockAuthService) UpdateUser(ctx context.Context, user *descriptors.User) error {
	return nil
}
func (m *mockAuthService) Me(ctx context.Context, userId int64) (*descriptors.User, error) {
	return &descriptors.User{Id: userId, Email: "test@aigen.local"}, nil
}
func (m *mockAuthService) GetRoleByName(ctx context.Context, name string) (*descriptors.Role, error) {
	return nil, nil
}
func (m *mockAuthService) BootstrapAdmin(ctx context.Context, defaultEmail, defaultPassword string, isTestEnv bool) error {
	return nil
}

type mockPermissionService struct {
	hasAccessFn func(ctx context.Context, userId int64, roles []string, entityName, action string) (bool, error)
}

func (m *mockPermissionService) HasAccess(ctx context.Context, userId int64, roles []string, entityName, action string) (bool, error) {
	if m.hasAccessFn != nil {
		return m.hasAccessFn(ctx, userId, roles, entityName, action)
	}
	return true, nil
}

func (m *mockPermissionService) GetRowFilters(ctx context.Context, userId int64, entityName string) ([]datamodels.Filter, error) {
	return nil, nil
}

func (m *mockPermissionService) GetFieldPermissions(ctx context.Context, entityName string, roles []string) (map[string]map[string]bool, error) {
	return nil, nil
}

type mockAppExtensionService struct {
	routingDocs map[string]string
	configs     map[string]string
}

func (m *mockAppExtensionService) GetRoutingDocs() map[string]string {
	return m.routingDocs
}

func (m *mockAppExtensionService) LoadAgenticConfig(id string) ([]byte, error) {
	if cfg, ok := m.configs[id]; ok {
		return []byte(cfg), nil
	}
	return nil, fmt.Errorf("config not found")
}

type dummyAgentConfig struct {
	Type string `yaml:"type"`
	Msg  string `yaml:"msg"`
}

func TestADK2AppApi_Integration(t *testing.T) {
	// 1. Generate RSA key pair for gateway verification
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	assert.NoError(t, err)
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	pubPEM := pem.EncodeToMemory(pubBlock)

	whatsappSvc, err := services.NewWhatsAppService(descriptors.ChannelConfig{
		PublicKey: string(pubPEM),
	})
	assert.NoError(t, err)

	// 2. Setup mock agent types in registry
	registry.RegisterAgentType("test_agent", func(ctx context.Context, name string, cfg *dummyAgentConfig, models registry.ModelRegistry, tools registry.ToolRegistry, sub []agent.Agent) (agent.Agent, error) {
		msg := cfg.Msg
		return agent.New(agent.Config{
			Name: name,
			Run: func(ic agent.InvocationContext) iter.Seq2[*session.Event, error] {
				return func(yield func(*session.Event, error) bool) {
					yield(&session.Event{LLMResponse: model.LLMResponse{Content: &genai.Content{Parts: []*genai.Part{{Text: msg}}}}}, nil)
				}
			},
		})
	})

	cfg := &config.Config{
		RootAgent: "my_agent",
		Agents: map[string]registry.AgentEntry{
			"my_agent": {
				Type:   "test_agent",
				Config: &dummyAgentConfig{Msg: "Hello from the Main Agent!"},
			},
		},
	}
	reg := registry.New(cfg)

	chatSvc := &services.ChatService{
		Registry:       reg,
		SessionService: session.InMemoryService(),
	}

	authSvc := &mockAuthService{}
	permSvc := &mockPermissionService{}
	mockExtSvc := &mockAppExtensionService{}

	apiSvc, err := NewADK2AppApi(chatSvc, authSvc, permSvc, whatsappSvc, mockExtSvc)
	assert.NoError(t, err)

	// 3. Setup router
	router := chi.NewRouter()
	apiSvc.Register(router)

	// 4. Create a valid JWT signed by the gateway
	now := time.Now()
	claims := services.ADKClaims{
		UserID:  "919876543210",
		Channel: "whatsapp",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "whatsadk-gateway",
			Audience:  jwt.ClaimStrings{"adk-agent"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(key)
	assert.NoError(t, err)

	t.Run("Initialize Session - Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/adk2app/apps/my_agent/users/919876543210/sessions/session123", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("Authorization", "Bearer "+tokenString)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Run Agent - Success", func(t *testing.T) {
		// Create session first so validation succeeds
		_, _ = chatSvc.SessionService.Create(context.Background(), &session.CreateRequest{
			AppName:   "my_agent",
			UserID:    "919876543210",
			SessionID: "session123",
		})

		runReqBody := map[string]interface{}{
			"appName":   "my_agent",
			"userId":    "919876543210",
			"sessionId": "session123",
			"newMessage": map[string]interface{}{
				"role":  "user",
				"parts": []interface{}{map[string]interface{}{"text": "Ping"}},
			},
		}
		bodyBytes, _ := json.Marshal(runReqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/adk2app/run", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+tokenString)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Hello from the Main Agent!")
	})

	t.Run("Run Agent - Forbidden Impersonation", func(t *testing.T) {
		runReqBody := map[string]interface{}{
			"appName":   "my_agent",
			"userId":    "919876543219", // Different userID from claims
			"sessionId": "session123",
			"newMessage": map[string]interface{}{
				"role":  "user",
				"parts": []interface{}{map[string]interface{}{"text": "Ping"}},
			},
		}
		bodyBytes, _ := json.Marshal(runReqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/adk2app/run", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+tokenString)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "user ID mismatch")
	})

	t.Run("Run Agent - Forbidden RBAC", func(t *testing.T) {
		permSvc.hasAccessFn = func(ctx context.Context, userId int64, roles []string, entityName, action string) (bool, error) {
			return false, nil
		}
		defer func() { permSvc.hasAccessFn = nil }()

		runReqBody := map[string]interface{}{
			"appName":   "my_agent",
			"userId":    "919876543210",
			"sessionId": "session123",
			"newMessage": map[string]interface{}{
				"role":  "user",
				"parts": []interface{}{map[string]interface{}{"text": "Ping"}},
			},
		}
		bodyBytes, _ := json.Marshal(runReqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/adk2app/run", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+tokenString)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "access denied")
	})
}
