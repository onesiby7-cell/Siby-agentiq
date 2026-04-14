package reasoning

import (
	"context"
	"fmt"
	"strings"

	"github.com/siby-agentiq/siby-agentiq/internal/provider"
	"github.com/siby-agentiq/siby-agentiq/internal/config"
)

type ThoughtPhase string

const (
	PhaseAnalysis    ThoughtPhase = "analysis"
	PhasePlanning    ThoughtPhase = "planning" 
	PhaseExecution   ThoughtPhase = "execution"
	PhaseReflection  ThoughtPhase = "reflection"
	PhaseConclusion  ThoughtPhase = "conclusion"
)

type Thought struct {
	Phase     ThoughtPhase
	Content   string
	Timestamp int64
}

type PlanStep struct {
	Order       int
	Action      string
	Description string
	Confidence  float64
	Dependencies []int
}

type ReasoningEngine struct {
	provider  provider.Provider
	config    config.ReasoningConfig
	history   []Thought
}

func NewReasoningEngine(p provider.Provider, cfg config.ReasoningConfig) *ReasoningEngine {
	return &ReasoningEngine{
		provider: p,
		config:   cfg,
		history:  make([]Thought, 0),
	}
}

func (re *ReasoningEngine) Think(ctx context.Context, userInput string, projectContext string) (*ReasoningResult, error) {
	result := &ReasoningResult{
		Thoughts: make([]Thought, 0),
		Plan:     make([]PlanStep, 0),
		Response: "",
	}

	analysis, err := re.analyze(ctx, userInput, projectContext)
	if err != nil {
		return nil, err
	}
	result.Thoughts = append(result.Thoughts, analysis)
	
	if re.config.ShowPlanning {
		plan, err := re.plan(ctx, userInput, analysis.Content)
		if err != nil {
			return nil, err
		}
		result.Plan = plan
	}
	
	if re.config.ChainOfThoughtEnabled() && re.config.ReflectionEnabled {
		reflection, err := re.reflect(ctx, result.Plan, analysis.Content)
		if err != nil {
			return nil, err
		}
		result.Thoughts = append(result.Thoughts, reflection)
	}
	
	response, err := re.generateResponse(ctx, userInput, projectContext, result)
	if err != nil {
		return nil, err
	}
	result.Response = response
	
	conclusion := Thought{
		Phase:   PhaseConclusion,
		Content: "Task completed with the above plan and response.",
	}
	result.Thoughts = append(result.Thoughts, conclusion)

	return result, nil
}

func (re *ReasoningEngine) analyze(ctx context.Context, input, projectContext string) (Thought, error) {
	systemPrompt := `You are an expert code analyst. Analyze the following user request and project context.

Provide a structured analysis covering:
1. **Intent**: What the user wants to achieve
2. **Scope**: The breadth of the request (file-level, project-level, system-level)
3. **Technologies**: Programming languages, frameworks, tools involved
4. **Complexity**: Estimated complexity (1-10) with reasoning
5. **Potential Challenges**: Edge cases, risks, considerations

Be concise but thorough.`

	messages := []provider.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Project Context:\n%s\n\nUser Request:\n%s", projectContext, input)},
	}

	resp, err := re.provider.Chat(ctx, messages)
	if err != nil {
		return Thought{}, err
	}

	return Thought{
		Phase:   PhaseAnalysis,
		Content: resp.Message.Content,
	}, nil
}

func (re *ReasoningEngine) plan(ctx context.Context, input, analysis string) ([]PlanStep, error) {
	systemPrompt := `Based on the analysis, create a step-by-step plan to accomplish the task.

For each step provide:
- Order number
- Action type (read, write, execute, analyze, etc.)
- Brief description
- Confidence level (0.0-1.0)
- Dependencies on other steps

Return your plan in a structured format that can be parsed.`

	messages := []provider.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Analysis:\n%s\n\nOriginal Request:\n%s", analysis, input)},
	}

	resp, err := re.provider.Chat(ctx, messages)
	if err != nil {
		return nil, err
	}

	return re.parsePlan(resp.Message.Content), nil
}

func (re *ReasoningEngine) parsePlan(content string) []PlanStep {
	steps := make([]PlanStep, 0)
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		if strings.Contains(line, "Step") || strings.Contains(line, "1.") || strings.Contains(line, "- ") {
			step := PlanStep{
				Order: i + 1,
				Action: extractAction(line),
				Description: line,
				Confidence: 0.8,
			}
			steps = append(steps, step)
		}
	}
	
	if len(steps) == 0 {
		steps = append(steps, PlanStep{
			Order: 1,
			Action: "execute",
			Description: content,
			Confidence: 0.7,
		})
	}
	
	return steps
}

func extractAction(line string) string {
	actions := []string{"read", "write", "execute", "analyze", "create", "modify", "delete", "search", "refactor"}
	lower := strings.ToLower(line)
	for _, action := range actions {
		if strings.Contains(lower, action) {
			return action
		}
	}
	return "execute"
}

func (re *ReasoningEngine) reflect(ctx context.Context, plan []PlanStep, analysis string) (Thought, error) {
	systemPrompt := `Critically review the proposed plan and analysis.

Consider:
1. Are there edge cases or corner cases not addressed?
2. Could the approach fail in certain scenarios?
3. What improvements or optimizations could be made?
4. Are there security concerns?
5. What would make this more robust?

Be critical but constructive.`

	messages := []provider.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Analysis:\n%s\n\nPlan:\n%+v", analysis, plan)},
	}

	resp, err := re.provider.Chat(ctx, messages)
	if err != nil {
		return Thought{}, err
	}

	return Thought{
		Phase:   PhaseReflection,
		Content: resp.Message.Content,
	}, nil
}

func (re *ReasoningEngine) generateResponse(ctx context.Context, input, projectContext string, reasoning *ReasoningResult) (string, error) {
	planSummary := formatPlanSummary(reasoning.Plan)
	thoughtsSummary := formatThoughtsSummary(reasoning.Thoughts)

	systemPrompt := fmt.Sprintf(`You are Siby-Agentiq, an expert AI coding assistant.

You have analyzed the user's request and created a plan. Now provide the final response.

Previous Analysis:
%s

Proposed Plan:
%s

Generate a response that:
1. Acknowledges the task
2. Summarizes your approach briefly
3. Provides the actual work/answer
4. Explains what you're doing

Be helpful, concise, and actionable.`, thoughtsSummary, planSummary)

	messages := []provider.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Project Context:\n%s\n\nUser Request:\n%s", projectContext, input)},
	}

	resp, err := re.provider.Chat(ctx, messages)
	if err != nil {
		return "", err
	}

	return resp.Message.Content, nil
}

func formatPlanSummary(plan []PlanStep) string {
	if len(plan) == 0 {
		return "No explicit plan generated."
	}
	
	var sb strings.Builder
	for _, step := range plan {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s (confidence: %.0f%%)\n", 
			step.Order, step.Action, step.Description, step.Confidence*100))
	}
	return sb.String()
}

func formatThoughtsSummary(thoughts []Thought) string {
	var sb strings.Builder
	for _, thought := range thoughts {
		sb.WriteString(fmt.Sprintf("\n--- %s ---\n%s\n", strings.ToUpper(string(thought.Phase)), thought.Content))
	}
	return sb.String()
}

type ReasoningResult struct {
	Thoughts []Thought
	Plan     []PlanStep
	Response string
}

func (c *config.ReasoningConfig) ChainOfThoughtEnabled() bool {
	return c.ShowThinking
}

func (c *config.ReasoningConfig) ReflectionEnabled() bool {
	return c.Reflection.CriticEnabled
}
