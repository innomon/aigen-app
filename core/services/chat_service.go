package services

import (
	"context"
	"fmt"

	"github.com/innomon/aigen-cms/core/agentic/agents"
	"github.com/innomon/agentic/pkg/config"
	"github.com/innomon/agentic/pkg/registry"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type ChatService struct {
	Registry       *registry.Registry
	EntityService  IEntityService
	SchemaService  *SchemaService
	A2UIService    *A2UIService
	SessionService session.Service
}

func NewChatService(configPath string, entityService IEntityService, schemaService *SchemaService, a2uiService *A2UIService) (*ChatService, error) {
	// Register custom types and tools
	agents.RegisterRouterAgent()
	RegisterCMSTools(entityService, schemaService, a2uiService)

	// Load agentic config
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load agentic config: %w", err)
	}

	// Initialize registry
	reg := registry.New(cfg)

	svc := &ChatService{
		Registry:       reg,
		EntityService:  entityService,
		SchemaService:  schemaService,
		A2UIService:    a2uiService,
		SessionService: session.InMemoryService(),
	}

	return svc, nil
}

func (s *ChatService) ProcessMessage(ctx context.Context, message string, history []string) (string, error) {
	// Get Root Agent
	rootAgent, err := s.Registry.GetRoot(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get root agent: %v", err)
	}

	// Create Runner
	rnr, err := runner.New(runner.Config{
		AppName:        "AiGenCMS",
		Agent:          rootAgent,
		SessionService: s.SessionService,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create runner: %v", err)
	}

	userContent := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: message},
		},
	}

	// Use a fixed session ID for now, or generate one
	sessionID := "default-session"
	userID := "default-user"

	var finalResponse string
	for evt, err := range rnr.Run(ctx, userID, sessionID, userContent, agent.RunConfig{}) {
		if err != nil {
			return "", fmt.Errorf("agent error: %v", err)
		}

		if evt.Content != nil {
			for _, part := range evt.Content.Parts {
				if part.Text != "" {
					finalResponse += part.Text
				}
			}
		}
	}

	return finalResponse, nil
}
