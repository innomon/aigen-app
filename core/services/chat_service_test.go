package services

import (
	"context"
	"encoding/json"
	"fmt"
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
	if m.history == nil {
		m.history = make(map[string][]*descriptors.Interaction)
	}
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

type mockLLM struct {
	response string
}

func (m *mockLLM) Name() string { return "mock_llm" }
func (m *mockLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		yield(&model.LLMResponse{Content: &genai.Content{Parts: []*genai.Part{{Text: m.response}}}}, nil)
	}
}

type mockModelRegistry struct {
	models map[string]model.LLM
}

func (m *mockModelRegistry) Get(ctx context.Context, name string) (model.LLM, error) {
	if mdl, ok := m.models[name]; ok {
		return mdl, nil
	}
	return nil, fmt.Errorf("model not found")
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
				SubAgents: []string{"bizdef1", "bizdef2"},
				Config: &agents.RouterAgentConfig{
					Description:     "Root Router",
					SelectionPrompt: "Choose: ${options}",
				},
			},
			"bizdef1": {
				Type: "mock",
				Config: &MockConfig{Response: "Hello from BizDef 1"},
			},
			"bizdef2": {
				Type: "mock",
				Config: &MockConfig{Response: "Hello from BizDef 2"},
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
	agents.RegisterRouterAgent(intSvc, nil)

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
		assert.Contains(t, resp, "bizdef1")
		assert.Contains(t, resp, "bizdef2")

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
			Options: []string{"bizdef1", "bizdef2"},
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
		assert.Equal(t, "Hello from BizDef 1", resp)
	})

	t.Run("Direct keyword match", func(t *testing.T) {
		resp, err := chatSvc.ProcessMessage(ctx, "user2", "Go to bizdef2")
		assert.NoError(t, err)
		assert.Equal(t, "Hello from BizDef 2", resp)
	})

	t.Run("LLM-based classification", func(t *testing.T) {
		mockMdl := &mockLLM{response: "bizdef1"}

		// Wait, ChatService uses registry.New(cfg).
		// I'll create a new ChatService for this test case.
		
		cfgClassify := &config.Config{
			RootAgent: "router",
			Models: map[string]registry.ModelEntry{
				"mock_mdl": {
					Provider: "mock",
				},
			},
			Agents: map[string]registry.AgentEntry{
				"router": {
					Type: "router",
					SubAgents: []string{"bizdef1"},
					Config: &agents.RouterAgentConfig{
						Classifier: agents.ClassifierConfig{
							Model: "mock_mdl",
							Prompt: "Classify: ${userInput} options: ${options}",
						},
					},
				},
				"bizdef1": {
					Type: "mock",
					Config: &MockConfig{Response: "Classified to BizDef 1"},
				},
			},
		}
		
		registry.RegisterModelProvider("mock", func(ctx context.Context, cfg *any) (model.LLM, error) {
			return mockMdl, nil
		})
		
		reg := registry.New(cfgClassify)
		chatSvcClassify := &ChatService{
			Registry:           reg,
			InteractionService: intSvc,
			SessionService:     session.InMemoryService(),
		}
		
		resp, err := chatSvcClassify.ProcessMessage(ctx, "user3", "Something vague")
		assert.NoError(t, err)
		assert.Equal(t, "Classified to BizDef 1", resp)
	})

	t.Run("Agent memory from InteractionService", func(t *testing.T) {
		identifier := "user-with-history"
		
		// 1. Pre-populate InteractionService with history
		intSvc.Log(ctx, &descriptors.Interaction{
			Identifier:  identifier,
			Direction:   "inbound",
			Content:     "My name is Alice",
			CreatedAt:   time.Now().Add(-10 * time.Minute),
		})
		intSvc.Log(ctx, &descriptors.Interaction{
			Identifier:  identifier,
			Direction:   "outbound",
			Content:     "Nice to meet you Alice",
			CreatedAt:   time.Now().Add(-9 * time.Minute),
		})

		// 2. Setup a mock agent that reports history
		cfgMem := &config.Config{
			RootAgent: "memory_agent",
			Agents: map[string]registry.AgentEntry{
				"memory_agent": {
					Type:   "memory_mock",
					Config: &MockConfig{},
				},
			},
		}

		registry.RegisterAgentType("memory_mock", func(ctx context.Context, name string, cfg *MockConfig, models registry.ModelRegistry, tools registry.ToolRegistry, sub []agent.Agent) (agent.Agent, error) {
			return agent.New(agent.Config{
				Name: name,
				Run: func(ic agent.InvocationContext) iter.Seq2[*session.Event, error] {
					return func(yield func(*session.Event, error) bool) {
						events := ic.Session().Events()
						count := 0
						historyText := ""
						for evt := range events.All() {
							if evt.Content != nil && len(evt.Content.Parts) > 0 {
								historyText += evt.Content.Parts[0].Text + "|"
								count++
							}
						}
						yield(&session.Event{LLMResponse: model.LLMResponse{Content: &genai.Content{Parts: []*genai.Part{{Text: fmt.Sprintf("History count: %d, items: %s", count, historyText)}}}}}, nil)
					}
				},
			})
		})

		reg := registry.New(cfgMem)
		chatSvcMem := &ChatService{
			Registry:           reg,
			InteractionService: intSvc,
			SessionService:     session.InMemoryService(),
		}

		resp, err := chatSvcMem.ProcessMessage(ctx, identifier, "Who am I?")
		assert.NoError(t, err)
		
		// The history should contain: "My name is Alice", "Nice to meet you Alice", AND the current "Who am I?"
		assert.Contains(t, resp, "History count: 3")
		assert.Contains(t, resp, "My name is Alice")
		assert.Contains(t, resp, "Nice to meet you Alice")
		assert.Contains(t, resp, "Who am I?")
	})
}
