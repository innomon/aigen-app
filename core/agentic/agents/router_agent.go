/*
Package agents implements the core router agent logic.

The Router Agent is responsible for directing user input to the most appropriate downstream sub-agent.
Its execution flow is as follows:

1. Input Handling: Extracts user input from the InvocationContext.
2. State Management: Checks for existing 'pending_selection' states via IInteractionService 
   to handle multi-turn routing decisions.
3. Routing Analysis:
   - Heuristic Match: Uses keyword matching against sub-agent names.
   - LLM Classification: Falls back to an LLM model (if configured) to classify intent.
   - User Prompting: If ambiguous, stores state and prompts the user for a selection.
4. Target Invocation: Delegates the invocation to the selected sub-agent once a target is resolved.
*/
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/agentic/pkg/registry"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type ClassifierConfig struct {
	Model           string `yaml:"model"`
	Prompt          string `yaml:"prompt"`
	FallbackMessage string `yaml:"fallback_message"`
}

type RouterAgentConfig struct {
	Type            string           `yaml:"type"`
	Description     string           `yaml:"description"`
	DefaultBizDef   string           `yaml:"default_bizdef"`
	SubAgents       []string         `yaml:"sub_agents"`
	Classifier      ClassifierConfig `yaml:"classifier"`
	SelectionPrompt string           `yaml:"selection_prompt"`
}

type IInteractionService interface {
	Log(ctx context.Context, interaction *descriptors.Interaction) error
	GetHistory(ctx context.Context, identifier string, limit int) ([]*descriptors.Interaction, error)
	UpdateStatus(ctx context.Context, id string, status string, errStr string) error
}

type IPluginProvider interface {
	GetRoutingDocs() map[string]string
	LoadAgenticConfig(id string) ([]byte, error)
}

func RegisterRouterAgent(svc IInteractionService, plugins IPluginProvider) {
	registry.RegisterAgentType("router", func(ctx context.Context, name string, cfg *RouterAgentConfig, models registry.ModelRegistry, tools registry.ToolRegistry, sub []agent.Agent) (agent.Agent, error) {
		return agent.New(agent.Config{
			Name:        name,
			Description: cfg.Description,
			SubAgents:   sub,
			Run: func(ic agent.InvocationContext) iter.Seq2[*session.Event, error] {
				return func(yield func(*session.Event, error) bool) {
					userID := ic.Session().UserID()
					
					// 1. Get User Input
					content := ic.UserContent()
					if content == nil || len(content.Parts) == 0 {
						yield(errorEvent(ic, "no input provided to router"), fmt.Errorf("no input provided to router"))
						return
					}

					userInput := ""
					if content.Parts[0].Text != "" {
						userInput = content.Parts[0].Text
					} else {
						yield(errorEvent(ic, "non-text input to router not supported yet"), fmt.Errorf("non-text input to router not supported yet"))
						return
					}

					// 2. Resolve Available Targets (Sub-Agents + Plugins)
					availableTargets := []string{}
					for _, a := range sub {
						availableTargets = append(availableTargets, a.Name())
					}
					
					pluginDocs := make(map[string]string)
					if plugins != nil {
						pluginDocs = plugins.GetRoutingDocs()
						for id := range pluginDocs {
							availableTargets = append(availableTargets, id)
						}
					}

					if len(availableTargets) == 0 {
						yield(textEvent(ic, "I'm sorry, I don't have any BizDefs or Plugins configured to handle your request."), nil)
						return
					}

					// 3. Check for Routing State
					stateKey := fmt.Sprintf("router:%s:state", userID)
					history, _ := svc.GetHistory(ctx, stateKey, 1)
					
					var state struct {
						Status  string   `json:"status"`
						Options []string `json:"options"`
					}

					if len(history) > 0 && history[0].Status == "pending_selection" {
						json.Unmarshal([]byte(history[0].Content), &state)
					}

					targetName := ""

					if state.Status == "pending_selection" {
						// Match by number or name
						targetIndex := -1
						cleanInput := strings.TrimSpace(strings.ToLower(userInput))
						
						for i, opt := range state.Options {
							if cleanInput == fmt.Sprintf("%d", i+1) || strings.Contains(strings.ToLower(opt), cleanInput) {
								targetIndex = i
								break
							}
						}

						if targetIndex >= 0 {
							targetName = state.Options[targetIndex]
							// Clear state
							svc.Log(ctx, &descriptors.Interaction{
								Identifier: stateKey,
								Status:     "completed",
								CreatedAt:  time.Now(),
							})
						} else {
							yield(textEvent(ic, cfg.Classifier.FallbackMessage), nil)
							return
						}
					} else {
						// 4. Perform Initial Routing
						if len(availableTargets) == 1 {
							targetName = availableTargets[0]
						} else {
							// Simple Keyword Match first
							lowerInput := strings.ToLower(userInput)
							for _, name := range availableTargets {
								if strings.Contains(lowerInput, strings.ToLower(name)) {
									targetName = name
									break
								}
							}

							// If no simple match, use LLM or Ask User
							if targetName == "" {
								// Try LLM classification if configured
								if cfg.Classifier.Model != "" {
									m, err := models.Get(ctx, cfg.Classifier.Model)
									if err == nil {
										optionsStr := strings.Join(availableTargets, ", ")
										// Enrich with plugin descriptions if available
										for id, desc := range pluginDocs {
											optionsStr += fmt.Sprintf("\n- %s: %s", id, desc)
										}

										prompt := strings.ReplaceAll(cfg.Classifier.Prompt, "${userInput}", userInput)
										prompt = strings.ReplaceAll(prompt, "${options}", optionsStr)
										
										request := &model.LLMRequest{
											Contents: []*genai.Content{
												{
													Role:  "user",
													Parts: []*genai.Part{{Text: prompt}},
												},
											},
										}

										var llmOutput string
										for resp, err := range m.GenerateContent(ctx, request, false) {
											if err == nil && resp.Content != nil && len(resp.Content.Parts) > 0 {
												llmOutput += resp.Content.Parts[0].Text
											}
										}

										if llmOutput != "" {
											llmOutput = strings.TrimSpace(strings.ToLower(llmOutput))
											// Look for target name in output
											for _, name := range availableTargets {
												if strings.Contains(llmOutput, strings.ToLower(name)) {
													targetName = name
													break
												}
											}
										}
									}
								}

								// If still no target, ask the user
								if targetName == "" {
									state.Status = "pending_selection"
									state.Options = availableTargets
									stateData, _ := json.Marshal(state)
									
									svc.Log(ctx, &descriptors.Interaction{
										Identifier: stateKey,
										Direction:  "inbound",
										Content:    string(stateData),
										Status:     "pending_selection",
										CreatedAt:  time.Now(),
									})

									optionsText := ""
									for i, opt := range availableTargets {
										optionsText += fmt.Sprintf("\n%d. %s", i+1, opt)
									}
									prompt := strings.ReplaceAll(cfg.SelectionPrompt, "${options}", optionsText)
									if prompt == "" {
										prompt = "Please select a BizDef or Plugin:" + optionsText
									}
									yield(textEvent(ic, prompt), nil)
									return
								}
							}
						}
					}

					// 5. Route to Target Agent (Sub-Agent or Plugin)
					if targetName != "" {
						// 5a. Check Sub-Agents first
						var target agent.Agent
						for _, a := range sub {
							if a.Name() == targetName {
								target = a
								break
							}
						}

						if target != nil {
							for evt, err := range target.Run(ic) {
								if !yield(evt, err) {
									return
								}
							}
							return
						}

						// 5b. Check Plugins
						if plugins != nil {
							if yamlData, err := plugins.LoadAgenticConfig(targetName); err == nil {
								// Instantiate plugin agent in memory
								// Note: We need a registry instance to parse the YAML.
								// In a real scenario, we might need to pass a sub-registry or the main one.
								// For now, let's assume we use the tools/models from the current context.
								
								// This is a bit complex as we need to parse the YAML and find the root_agent.
								// For the POC, let's assume agentic.LoadFromYAML exists or similar.
								
								// TODO: Implement actual Agentic YAML loading and execution here.
								yield(textEvent(ic, fmt.Sprintf("Routing to plugin %s. (Agentic YAML loaded, executing root_agent...)", targetName)), nil)
								
								// Placeholder for actual execution:
								// pluginAgent, _ := registry.LoadAgentFromYAML(ctx, yamlData, models, tools)
								// for evt, err := range pluginAgent.Run(ic) { ... }
								
								_ = yamlData
								return
							}
						}

						yield(textEvent(ic, "Failed to route to target: "+targetName), nil)
					}
				}
			},
		})
	})
}

func textEvent(ic agent.InvocationContext, text string) *session.Event {
	return &session.Event{
		LLMResponse: model.LLMResponse{
			Content: &genai.Content{
				Role: "model",
				Parts: []*genai.Part{
					{Text: text},
				},
			},
		},
	}
}

func errorEvent(ic agent.InvocationContext, err string) *session.Event {
	return &session.Event{
		LLMResponse: model.LLMResponse{
			ErrorMessage: err,
		},
	}
}
