package services

import (
	"context"
	"encoding/json"
	"iter"
	"testing"
	"time"

	"github.com/innomon/aigen-app/core/agentic/agents"
	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/agentic/pkg/config"
	"github.com/innomon/agentic/pkg/registry"
	"github.com/stretchr/testify/assert"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type mockInteractionService struct {
	history map[string][]*descriptors.Interaction
}

func (m *mockInteractionService) Log(ctx context.Context, interaction *descriptors.Interaction) error {
	m.history[interaction.Identifier] = append(m.history[interaction.Identifier], interaction)
	return nil
}

func (m *mockInteractionService) GetHistory(ctx context.Context, identifier string, limit int) ([]*descriptors.Interaction, error) {
	h := m.history[identifier]
	if len(h) == 0 {
		return nil, nil
	}
	reversed := make([]*descriptors.Interaction, len(h))
	for i, item := range h {
		reversed[len(h)-1-i] = item
	}
	if len(reversed) > limit {
		return reversed[:limit], nil
	}
	return reversed, nil
}

func (m *mockInteractionService) UpdateStatus(ctx context.Context, id string, status string, errStr string) error {
	return nil
}

func (m *mockInteractionService) GetPendingOutbound(ctx context.Context, channel descriptors.ChannelType) ([]*descriptors.Interaction, error) {
	return nil, nil
}

type MockConfig struct {
	Response string `yaml:"response"`
}

func TestChatServiceAndRouter(t *testing.T) {
	ctx := context.Background()
	intSvc := &mockInteractionService{history: make(map[string][]*descriptors.Interaction)}
	
	// Setup Registry with mock agents
	cfg := &config.Config{
		RootAgent: "router",
		Agents: map[string]registry.AgentEntry{
			"router": {
				Type: "router",
				SubAgents: []string{"app1", "app2"},
				Config: &agents.RouterAgentConfig{
					Description:     "Root Router",
					SelectionPrompt: "Choose: ${options}",
				},
			},
			"app1": {
				Type: "mock",
				Config: &MockConfig{Response: "Hello from App 1"},
			},
			"app2": {
				Type: "mock",
				Config: &MockConfig{Response: "Hello from App 2"},
			},
		},
	}

	registry.RegisterAgentType("mock", func(ctx context.Context, name string, cfg *MockConfig, models registry.ModelRegistry, tools registry.ToolRegistry, sub []agent.Agent) (agent.Agent, error) {
		resp := cfg.Response
		return agent.New(agent.Config{
			Name: name,
			Run: func(ic agent.InvocationContext) iter.Seq2[*session.Event, error] {
				return func(yield func(*session.Event, error) bool) {
					yield(&session.Event{LLMResponse: model.LLMResponse{Content: &genai.Content{Parts: []*genai.Part{{Text: resp}}}}}, nil)
				}
			},
		})
	})

	// We need to register "router" too, ChatService.NewChatService calls agents.RegisterRouterAgent
	agents.RegisterRouterAgent(intSvc)

	reg := registry.New(cfg)
	chatSvc := &ChatService{
		Registry:           reg,
		InteractionService: intSvc,
		SessionService:     session.InMemoryService(),
	}

	t.Run("Router asks for selection", func(t *testing.T) {
		resp, err := chatSvc.ProcessMessage(ctx, "user1", "I need help")
		assert.NoError(t, err)
		assert.Contains(t, resp, "Choose:")
		assert.Contains(t, resp, "app1")
		assert.Contains(t, resp, "app2")

		// Verify state key
		stateKey := "router:user1:state"
		history, _ := intSvc.GetHistory(ctx, stateKey, 1)
		assert.Equal(t, "pending_selection", history[0].Status)
	})

	t.Run("Router routes after selection", func(t *testing.T) {
		// Mock state from previous turn
		stateKey := "router:user1:state"
		state := struct {
			Status  string   `json:"status"`
			Options []string `json:"options"`
		}{
			Status:  "pending_selection",
			Options: []string{"app1", "app2"},
		}
		stateData, _ := json.Marshal(state)
		intSvc.Log(ctx, &descriptors.Interaction{
			Identifier: stateKey,
			Status:     "pending_selection",
			Content:    string(stateData),
			CreatedAt:  time.Now(),
		})

		resp, err := chatSvc.ProcessMessage(ctx, "user1", "1")
		assert.NoError(t, err)
		assert.Equal(t, "Hello from App 1", resp)
	})

	t.Run("Direct keyword match", func(t *testing.T) {
		resp, err := chatSvc.ProcessMessage(ctx, "user2", "Go to app2")
		assert.NoError(t, err)
		assert.Equal(t, "Hello from App 2", resp)
	})
}
