package services

import (
	"context"
	"fmt"

	"github.com/innomon/aigen-app/core/agentic/agents"
	"github.com/innomon/agentic/pkg/config"
	"github.com/innomon/agentic/pkg/registry"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type ChatService struct {
	Registry           *registry.Registry
	EntityService      IEntityService
	SchemaService      *SchemaService
	A2UIService        *A2UIService
	InteractionService IInteractionService
	SessionService     session.Service
}

func NewChatService(configPath string, entityService IEntityService, schemaService *SchemaService, a2uiService *A2UIService, interactionService IInteractionService) (*ChatService, error) {
	// Register custom types and tools
	agents.RegisterRouterAgent(interactionService)
	RegisterCMSTools(entityService, schemaService, a2uiService)

	// Load agentic config
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load agentic config: %w", err)
	}

	// Initialize registry
	reg := registry.New(cfg)

	svc := &ChatService{
		Registry:           reg,
		EntityService:      entityService,
		SchemaService:      schemaService,
		A2UIService:        a2uiService,
		InteractionService: interactionService,
		SessionService:     session.InMemoryService(),
	}

	return svc, nil
}

func (s *ChatService) ProcessMessage(ctx context.Context, identifier string, message string) (string, error) {
	// 1. Get History from InteractionService
	_, err := s.InteractionService.GetHistory(ctx, identifier, 10)
	if err != nil {
		return "", fmt.Errorf("failed to get interaction history: %v", err)
	}

	// 2. Get Root Agent
	rootAgent, err := s.Registry.GetRoot(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get root agent: %v", err)
	}

	// 3. Create Runner
	rnr, err := runner.New(runner.Config{
		AppName:        "AiGenCMS",
		Agent:          rootAgent,
		SessionService: s.SessionService,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create runner: %v", err)
	}

	// 4. Use ADK Session Service to pre-populate history if needed
	// For now, ADK's SessionService is in-memory and handles turns within its own logic.
	// But we can manually inject past interactions into the userContent if we wanted to 
	// bypass ADK's session management or sync them.
	// ADK expects a list of Content objects for multi-turn.
	
	userContent := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: message},
		},
	}

	// Map identifier to session/user IDs for ADK
	sessionID := "session-" + identifier
	userID := identifier

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
