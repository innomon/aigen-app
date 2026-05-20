package services

import (
	"context"
	"fmt"
	"log"

	"github.com/innomon/agentic/pkg/config"
	"github.com/innomon/agentic/pkg/registry"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type ChatService struct {
	Registry           *registry.Registry
	EntityService      IEntityService
	SchemaService      *SchemaService
	EvolutionService   IEvolutionService
	A2UIService        *A2UIService
	InteractionService IInteractionService
	SessionService     session.Service
}

func NewChatService(configPath string, entityService IEntityService, schemaService *SchemaService, evolutionService IEvolutionService, a2uiService *A2UIService, interactionService IInteractionService, commerceService ICommerceService) (*ChatService, error) {
	// Register custom types and tools
	RegisterCMSTools(entityService, schemaService, evolutionService, a2uiService, commerceService)

	// Load agentic config
	var cfg *config.Config
	var err error
	if configPath != "" {
		cfg, err = config.Load(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load agentic config: %w", err)
		}
	} else {
		cfg = &config.Config{}
	}

	// Initialize registry
	reg := registry.New(cfg)

	svc := &ChatService{
		Registry:           reg,
		EntityService:      entityService,
		SchemaService:      schemaService,
		EvolutionService:   evolutionService,
		A2UIService:        a2uiService,
		InteractionService: interactionService,
		SessionService:     session.InMemoryService(),
	}

	return svc, nil
}

func (s *ChatService) RegisterTool(name string, handler func(context.Context, map[string]any) (any, error)) {
	registry.RegisterToolHandler(name, handler)
}

func (s *ChatService) AddAgenticConfig(cfg *config.Config) error {
	// In a real implementation, we'd merge this config into the registry.
	// For now, we assume the registry is updated or we re-initialize parts of it.
	log.Printf("Merging agentic config for plugin...")
	return nil
}

func (s *ChatService) ProcessMessage(ctx context.Context, identifier string, message string) (string, error) {
	// Map identifier to session/user IDs for ADK
	sessionID := "session-" + identifier
	userID := identifier

	// 1. Ensure session exists
	resp, err := s.SessionService.Get(ctx, &session.GetRequest{
		AppName:   "AiGenCMS",
		UserID:    userID,
		SessionID: sessionID,
	})

	isNewSession := false
	var sess session.Session
	if err != nil {
		createResp, err := s.SessionService.Create(ctx, &session.CreateRequest{
			AppName:   "AiGenCMS",
			UserID:    userID,
			SessionID: sessionID,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create session: %v", err)
		}
		sess = createResp.Session
		isNewSession = true
	} else {
		sess = resp.Session
	}

	// 2. If new session, populate with history from InteractionService
	if isNewSession {
		history, err := s.InteractionService.GetHistory(ctx, identifier, 20)
		if err == nil && len(history) > 0 {
			// history is desc (newest first), we need asc for AppendEvent
			for i := len(history) - 1; i >= 0; i-- {
				item := history[i]
				role := "user"
				if item.Direction == "outbound" {
					role = "model"
				}

				evt := &session.Event{
					LLMResponse: model.LLMResponse{
						Content: &genai.Content{
							Role:  role,
							Parts: []*genai.Part{{Text: item.Content}},
						},
					},
					Timestamp: item.CreatedAt,
				}
				s.SessionService.AppendEvent(ctx, sess, evt)
			}
		}
	}

	// 3. Get Root Agent
	rootAgent, err := s.Registry.GetRoot(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get root agent: %v", err)
	}

	// 4. Create Runner
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
