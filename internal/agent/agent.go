package agent

import (
	"context"
	"fmt"

	"github.com/siby-agentiq/siby-agentiq/internal/config"
	"github.com/siby-agentiq/siby-agentiq/internal/filesystem"
	"github.com/siby-agentiq/siby-agentiq/internal/provider"
	"github.com/siby-agentiq/siby-agentiq/internal/reasoning"
)

type Agent struct {
	provider  provider.Provider
	reasoning *reasoning.ReasoningEngine
	scanner   *filesystem.ProjectScanner
	config    config.AgentConfig
}

func NewAgent(p provider.Provider, cfg config.AgentConfig, reasoningCfg config.ReasoningConfig, contextCfg config.ContextConfig) *Agent {
	return &Agent{
		provider:  p,
		reasoning: reasoning.NewReasoningEngine(p, reasoningCfg),
		scanner:   filesystem.NewProjectScanner(contextCfg),
		config:    cfg,
	}
}

func (a *Agent) ProcessRequest(ctx context.Context, input string, workingDir string) (*AgentResponse, error) {
	projectContext, err := a.scanner.ScanProject(workingDir)
	if err != nil {
		projectContext = fmt.Sprintf("Error scanning project: %v", err)
	}

	if a.config.ChainOfThought.Enabled {
		return a.processWithReasoning(ctx, input, projectContext)
	}

	return a.processDirect(ctx, input, projectContext)
}

func (a *Agent) processWithReasoning(ctx context.Context, input string, projectContext string) (*AgentResponse, error) {
	result, err := a.reasoning.Think(ctx, input, projectContext)
	if err != nil {
		return nil, fmt.Errorf("reasoning failed: %w", err)
	}

	return &AgentResponse{
		Thoughts: result.Thoughts,
		Plan:     result.Plan,
		Output:   result.Response,
	}, nil
}

func (a *Agent) processDirect(ctx context.Context, input string, projectContext string) (*AgentResponse, error) {
	systemPrompt := `You are Siby-Agentiq, an expert AI coding assistant that helps developers with their projects.

You have access to the project context below. Answer the user's questions and help with coding tasks.
Be concise, accurate, and helpful. Provide code examples when relevant.`

	messages := []provider.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Project Context:\n%s\n\nUser Request:\n%s", projectContext, input)},
	}

	resp, err := a.provider.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("provider chat failed: %w", err)
	}

	return &AgentResponse{
		Output: resp.Message.Content,
	}, nil
}

func (a *Agent) Chat(ctx context.Context, messages []provider.Message) (*provider.ChatResponse, error) {
	return a.provider.Chat(ctx, messages)
}

func (a *Agent) ChatStream(ctx context.Context, messages []provider.Message) (<-chan string, error) {
	return a.provider.ChatStream(ctx, messages)
}

type AgentResponse struct {
	Thoughts []reasoning.Thought
	Plan     []reasoning.PlanStep
	Output   string
}
