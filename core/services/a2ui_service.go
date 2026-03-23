package services

import (
	"context"
	"sync"
)

// A2UIComponent represents a single component in the adjacency list.
type A2UIComponent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Children   []string               `json:"children,omitempty"`
}

// A2UIMessage is the payload sent over the SSE stream.
type A2UIMessage struct {
	Components []A2UIComponent `json:"components"`
}

// IA2UIService defines the methods for managing A2UI state.
type IA2UIService interface {
	UpdateComponent(ctx context.Context, component A2UIComponent)
	GetComponent(id string) (A2UIComponent, bool)
	GetFullState() []A2UIComponent
	Subscribe() (chan []A2UIComponent, string)
	Unsubscribe(id string)
}

type A2UIService struct {
	mu          sync.RWMutex
	components  map[string]A2UIComponent
	subscribers map[string]chan []A2UIComponent
}

func NewA2UIService() *A2UIService {
	return &A2UIService{
		components:  make(map[string]A2UIComponent),
		subscribers: make(map[string]chan []A2UIComponent),
	}
}

func (s *A2UIService) UpdateComponent(ctx context.Context, component A2UIComponent) {
	s.mu.Lock()
	s.components[component.ID] = component
	fullState := s.serializeState()
	s.mu.Unlock()

	// Notify all subscribers of the update
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sub := range s.subscribers {
		// Non-blocking send
		select {
		case sub <- fullState:
		default:
		}
	}
}

func (s *A2UIService) GetComponent(id string) (A2UIComponent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.components[id]
	return c, ok
}

func (s *A2UIService) GetFullState() []A2UIComponent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.serializeState()
}

func (s *A2UIService) Subscribe() (chan []A2UIComponent, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	id := generateID() // Simple helper needed
	ch := make(chan []A2UIComponent, 10)
	s.subscribers[id] = ch
	
	// Send initial state immediately
	ch <- s.serializeState()
	
	return ch, id
}

func (s *A2UIService) Unsubscribe(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.subscribers[id]; ok {
		close(ch)
		delete(s.subscribers, id)
	}
}

func (s *A2UIService) serializeState() []A2UIComponent {
	state := make([]A2UIComponent, 0, len(s.components))
	for _, c := range s.components {
		state = append(state, c)
	}
	return state
}

// Simple ID generator for subscribers
func generateID() string {
	b := make([]byte, 8)
	// In a real app, use a proper UUID, but this works for a prototype
	// We'll just use a counter or timestamp for simplicity in this turn
	return "sub_" + string(b) 
}
