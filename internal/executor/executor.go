package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

type Executor struct {
	parser   *ResponseParser
	safety   *SafetyManager
	pm       *provider.ProviderManager
	confirmFn func(string) bool
}

type ExecutorConfig struct {
	AutoBackup   bool
	ConfirmAll   bool
	DryRun       bool
	MaxFileSize  int64
}

func NewExecutor(pm *provider.ProviderManager, cfg ExecutorConfig) *Executor {
	return &Executor{
		parser: NewResponseParser(),
		safety: NewSafetyManager(cfg),
		pm:     pm,
	}
}

func (e *Executor) SetConfirmFunc(fn func(string) bool) {
	e.confirmFn = fn
}

type TaskRequest struct {
	Task        string
	Context     string
	UsePlanning bool
}

func (e *Executor) ExecuteTask(ctx context.Context, req TaskRequest) (*ExecutionResult, error) {
	if req.UsePlanning {
		plan, err := e.CreatePlan(ctx, req.Task, req.Context)
		if err != nil {
			return nil, fmt.Errorf("planning failed: %w", err)
		}

		if e.confirmFn != nil && !e.confirmFn(plan.Summary()) {
			return &ExecutionResult{Success: false, Summary: "Cancelled by user"}, nil
		}

		return e.ExecutePlan(ctx, plan)
	}

	messages := []provider.Message{
		{Role: "system", Content: getCodeGenerationPrompt()},
		{Role: "user", Content: req.Context + "\n\n" + req.Task},
	}

	ch, err := e.pm.SmartStream(ctx, provider.SmartChatRequest{Messages: messages})
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	var response strings.Builder
	for chunk := range ch {
		if chunk.Done {
			break
		}
		response.WriteString(chunk.Content)
	}

	changes := e.parser.Parse(response.String())
	return e.ExecuteChanges(changes)
}

func (e *Executor) CreatePlan(ctx context.Context, task, context string) (*Plan, error) {
	planningPrompt := fmt.Sprintf(`Analyze this task and create a detailed execution plan.

Task: %s

Context:
%s

Provide your response in this format:

FILES:
- List each file that needs to be created or modified
- Include the full path from project root

CHANGES:
- For each file, describe what needs to be added/removed/modified
- Be specific about line numbers when possible

COMMANDS:
- Any terminal commands needed after the changes (e.g., go mod tidy, npm install)
- Only if necessary

RATIONALE:
- Brief explanation of why these changes are needed`, task, context)

	messages := []provider.Message{
		{Role: "system", Content: "You are a software architect. Create clear, minimal plans."},
		{Role: "user", Content: planningPrompt},
	}

	resp, err := e.pm.SmartChat(ctx, provider.SmartChatRequest{Messages: messages})
	if err != nil {
		return nil, err
	}

	return e.parser.ParsePlan(resp.Message.Content)
}

func (e *Executor) ExecutePlan(ctx context.Context, plan *Plan) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Changes: plan.ProposedChanges,
		Success: true,
	}

	for i, change := range plan.ProposedChanges {
		execResult, err := e.safety.ExecuteWithProtection(change, func(c FileChange) error {
			return e.applyChange(c)
		})
		
		if err != nil {
			result.AddResult(FileResult{
				Path:    change.Path,
				Success: false,
				Error:   err,
			})
		} else {
			result.AddResult(FileResult{
				Path:    change.Path,
				Success: true,
				Backup:  execResult.BackupPath,
			})
		}
		
		if progress := float64(i+1) / float64(len(plan.ProposedChanges)); progressCallback != nil {
			progressCallback(progress, change.Path)
		}
	}

	result.Summary = result.GetSummary()
	return result, nil
}

func (e *Executor) ExecuteChanges(changes []FileChange) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Changes: changes,
		Success: true,
	}

	for i, change := range changes {
		if err := e.parser.ValidatePath(change.Path); err != nil {
			result.AddResult(FileResult{
				Path:    change.Path,
				Success: false,
				Error:   err,
			})
			continue
		}

		if e.safety.cfg.ConfirmAll && e.confirmFn != nil {
			msg := fmt.Sprintf("%s %s?", change.Action, change.Path)
			if !e.confirmFn(msg) {
				continue
			}
		}

		execResult, err := e.safety.ExecuteWithProtection(change, func(c FileChange) error {
			return e.applyChange(c)
		})

		if err != nil {
			result.AddResult(FileResult{
				Path:  change.Path,
				Success: false,
				Error: err,
			})
		} else {
			result.AddResult(FileResult{
				Path:   change.Path,
				Success: true,
				Backup: execResult.BackupPath,
			})
		}

		if progress := float64(i+1) / float64(len(changes)); progressCallback != nil {
			progressCallback(progress, change.Path)
		}
	}

	result.Summary = result.GetSummary()
	return result, nil
}

func (e *Executor) applyChange(change FileChange) error {
	switch change.Action {
	case ActionCreate, ActionModify:
		return WriteFile(change.Path, change.Content)
	case ActionDelete:
		return os.Remove(change.Path)
	default:
		return fmt.Errorf("unknown action: %s", change.Action)
	}
}

func getCodeGenerationPrompt() string {
	return `You are Siby, an expert coding assistant.

When writing code, use this format:

FILE: path/to/file.go
```language
// your code here
```
END_FILE

For multiple files:
FILE: file1.go
```go
// code
```
END_FILE

FILE: file2.go
```go
// code
```
END_FILE

Use CREATE: path/to/file for new files
Use MODIFY: path/to/file for existing files
Use DELETE: path/to/file to remove files

Always provide complete, working code. No placeholders.`
}

var progressCallback func(float64, string)
func SetProgressCallback(fn func(float64, string)) {
	progressCallback = fn
}
