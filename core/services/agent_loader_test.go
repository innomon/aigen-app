package services

import (
	"context"
	"fmt"
	"iter"
	"testing"

	"github.com/innomon/agentic/pkg/config"
	"github.com/innomon/agentic/pkg/registry"
	"github.com/stretchr/testify/assert"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type mockAppExtensionService struct {
	routingDocs map[string]string
	configs     map[string]string
}

func (m *mockAppExtensionService) GetRoutingDocs() map[string]string {
	return m.routingDocs
}

func (m *mockAppExtensionService) LoadAgenticConfig(id string) ([]byte, error) {
	cfg, ok := m.configs[id]
	if !ok {
		return nil, fmt.Errorf("config not found")
	}
	return []byte(cfg), nil
}

type dummyConfig struct {
	Type string `yaml:"type"`
	Msg  string `yaml:"msg"`
}

func TestAppAgentLoader(t *testing.T) {
	// Register dummy agent type for testing
	registry.RegisterAgentType("dummy", func(ctx context.Context, name string, cfg *dummyConfig, models registry.ModelRegistry, tools registry.ToolRegistry, sub []agent.Agent) (agent.Agent, error) {
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

	// 1. Setup ChatService with a Registry containing a main agent
	cfg := &config.Config{
		RootAgent: "main_agent",
		Agents: map[string]registry.AgentEntry{
			"main_agent": {
				Type:   "dummy",
				Config: &dummyConfig{Msg: "I am the main agent"},
			},
		},
	}
	reg := registry.New(cfg)
	chatSvc := &ChatService{
		Registry: reg,
	}

	// 2. Setup mock app extension service
	mockExtSvc := &mockAppExtensionService{
		routingDocs: map[string]string{
			"ext_agent": "An extension agent",
		},
		configs: map[string]string{
			"ext_agent": `
agents:
  ext_agent:
    type: dummy
    msg: "I am the extension agent"
root_agent: ext_agent
`,
		},
	}

	loader := NewAppAgentLoader(chatSvc, mockExtSvc)

	t.Run("List Agents", func(t *testing.T) {
		agents := loader.ListAgents()
		assert.Contains(t, agents, "main_agent")
		assert.Contains(t, agents, "ext_agent")
		assert.Len(t, agents, 2)
	})

	t.Run("Load Main Agent", func(t *testing.T) {
		ag, err := loader.LoadAgent("main_agent")
		assert.NoError(t, err)
		assert.Equal(t, "main_agent", ag.Name())
	})

	t.Run("Load Extension Agent", func(t *testing.T) {
		ag, err := loader.LoadAgent("ext_agent")
		assert.NoError(t, err)
		assert.Equal(t, "ext_agent", ag.Name())
	})

	t.Run("Load Non-existent Agent", func(t *testing.T) {
		_, err := loader.LoadAgent("ghost_agent")
		assert.Error(t, err)
	})

	t.Run("Root Agent", func(t *testing.T) {
		root := loader.RootAgent()
		assert.NotNil(t, root)
		assert.Equal(t, "main_agent", root.Name())
	})
}
