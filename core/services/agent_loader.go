package services

import (
	"context"
	"fmt"

	"github.com/innomon/agentic/pkg/registry"
	"google.golang.org/adk/agent"
	yaml "gopkg.in/yaml.v3"
)

type AppAgentLoader struct {
	chatService      *ChatService
	extensionService IAppExtensionService
}

func NewAppAgentLoader(chatService *ChatService, extensionService IAppExtensionService) *AppAgentLoader {
	return &AppAgentLoader{
		chatService:      chatService,
		extensionService: extensionService,
	}
}

// ListAgents returns a list of names of all agents from the main registry and app extensions.
func (l *AppAgentLoader) ListAgents() []string {
	agents := []string{}
	if l.chatService != nil && l.chatService.Registry != nil {
		for name := range l.chatService.Registry.Config().Agents {
			agents = append(agents, name)
		}
	}
	if l.extensionService != nil {
		for id := range l.extensionService.GetRoutingDocs() {
			agents = append(agents, id)
		}
	}
	return agents
}

// LoadAgent returns an agent by its name.
func (l *AppAgentLoader) LoadAgent(name string) (agent.Agent, error) {
	ctx := context.Background()

	// 1. Try to load from the main ChatService Registry
	if l.chatService != nil && l.chatService.Registry != nil {
		if _, exists := l.chatService.Registry.Config().Agents[name]; exists {
			return registry.Get[agent.Agent](ctx, l.chatService.Registry, name)
		}
	}

	// 2. Try to load from the App Extensions
	if l.extensionService != nil {
		yamlData, err := l.extensionService.LoadAgenticConfig(name)
		if err == nil {
			var raw registry.RawConfig
			if err := yaml.Unmarshal(yamlData, &raw); err != nil {
				return nil, fmt.Errorf("failed to unmarshal agentic config for extension %s: %w", name, err)
			}
			cfg, err := registry.ParseRaw(&raw)
			if err != nil {
				return nil, fmt.Errorf("failed to parse agentic config for extension %s: %w", name, err)
			}
			extensionReg := registry.New(cfg)
			root, err := extensionReg.GetRoot(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get root agent for extension %s: %w", name, err)
			}
			return root, nil
		}
	}

	return nil, fmt.Errorf("agent %s not found in registry or extensions", name)
}

// RootAgent returns the root agent from the main registry.
func (l *AppAgentLoader) RootAgent() agent.Agent {
	if l.chatService != nil && l.chatService.Registry != nil {
		root, err := l.chatService.Registry.GetRoot(context.Background())
		if err == nil {
			return root
		}
	}
	return nil
}
