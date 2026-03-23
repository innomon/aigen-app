package services

import (
	"context"
	"fmt"
	"iter"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
)

type A2AService struct {
	chatService *ChatService
	handler     a2asrv.RequestHandler
	agentCard   *a2a.AgentCard
}

type cmsAgentExecutor struct {
	chatService *ChatService
}

func NewA2AService(chatService *ChatService, domain string) *A2AService {
	executor := &cmsAgentExecutor{chatService: chatService}
	handler := a2asrv.NewHandler(executor)

	addr := fmt.Sprintf("https://%s/api/a2a", domain)
	if domain == "" || domain == "localhost" {
		addr = "http://localhost:5000/api/a2a"
	}

	agentCard := &a2a.AgentCard{
		Name:        "AiGen CMS Agent",
		Description: "Aigen CMS intelligent agent for data management and UI orchestration",
		SupportedInterfaces: []*a2a.AgentInterface{
			a2a.NewAgentInterface(addr, a2a.TransportProtocolJSONRPC),
		},
		DefaultInputModes:  []string{"text"},
		DefaultOutputModes: []string{"text"},
		Capabilities:       a2a.AgentCapabilities{Streaming: true},
	}

	return &A2AService{
		chatService: chatService,
		handler:     handler,
		agentCard:   agentCard,
	}
}

func (s *A2AService) GetHandler() a2asrv.RequestHandler {
	return s.handler
}

func (s *A2AService) GetAgentCard() *a2a.AgentCard {
	return s.agentCard
}

func (e *cmsAgentExecutor) Execute(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		if e.chatService == nil {
			yield(nil, fmt.Errorf("chat service not initialized"))
			return
		}

		// Extract message from A2A request
		var userMessage string
		for _, part := range execCtx.Message.Parts {
			userMessage += part.Text()
		}

		// Call ChatService (this is where we'd bridge A2A to our internal Agentic system)
		// For now, simple response or stream if supported
		// In a real scenario, we'd wrap the ChatService stream into A2A events
		
		resp, err := e.chatService.ProcessMessage(ctx, userMessage, nil)
		if err != nil {
			yield(nil, err)
			return
		}

		// Yield the response as an A2A Message
		a2aResp := a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart(resp))
		yield(a2aResp, nil)
	}
}

func (e *cmsAgentExecutor) Cancel(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		// Implementation for cancellation if needed
	}
}
