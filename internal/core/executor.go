package core

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/siby-agentiq/siby-agentiq/internal/executor"
)

type FileOperation struct {
	Action   string
	Path     string
	Content  string
	Language string
}

type ExecutionPlan struct {
	Files    []FileOperation
	Commands []string
	Summary  string
}

type Executor struct {
	parser     *executor.ResponseParser
	safety     *SafetyManager
}

type SafetyManager struct {
	allowDelete bool
	allowExec   bool
}

func NewExecutor() *Executor {
	return &Executor{
		parser: executor.NewResponseParser(),
		safety: &SafetyManager{
			allowDelete: false,
			allowExec:   true,
		},
	}
}

func (e *Executor) ParsePlan(response string) *ExecutionPlan {
	plan := &ExecutionPlan{}

	plan.Files = e.extractFileOps(response)
	plan.Commands = e.extractCommands(response)
	plan.Summary = e.generateSummary(plan)

	return plan
}

func (e *Executor) extractFileOps(response string) []FileOperation {
	var ops []FileOperation

	fileRegex := regexp.MustCompile(`(?i)FILE:\s*(.+?)(?:\nLANGAGE:?\s*(\w+))?\n*---`)
	matches := fileRegex.FindAllStringSubmatch(response, -1)

	contentStart := 0
	for _, match := range matches {
		path := strings.TrimSpace(match[1])
		lang := ""
		if len(match) > 2 {
			lang = strings.TrimSpace(match[2])
		}

		start := strings.Index(response[contentStart:], match[0])
		if start == -1 {
			continue
		}
		start += contentStart + len(match[0])

		end := strings.Index(response[start:], "END_FILE")
		if end == -1 {
			end = len(response)
		} else {
			end += start
		}

		content := strings.TrimSpace(response[start:end])
		if strings.HasPrefix(content, "---") {
			content = strings.TrimPrefix(content, "---")
			content = strings.TrimSpace(content)
		}

		action := determineAction(response, path)

		ops = append(ops, FileOperation{
			Action:   action,
			Path:     path,
			Content:  content,
			Language: lang,
		})
	}

	createRegex := regexp.MustCompile(`(?i)ACTION:\s*CREATE\s+(.+)`)
	for _, match := range createRegex.FindAllStringSubmatch(response, -1) {
		path := strings.TrimSpace(match[1])
		if !pathExists(ops, path) {
			ops = append(ops, FileOperation{
				Action: "create",
				Path:   path,
			})
		}
	}

	modifyRegex := regexp.MustCompile(`(?i)ACTION:\s*MODIFY\s+(.+)`)
	for _, match := range modifyRegex.FindAllStringSubmatch(response, -1) {
		path := strings.TrimSpace(match[1])
		if !pathExists(ops, path) {
			ops = append(ops, FileOperation{
				Action: "modify",
				Path:   path,
			})
		}
	}

	deleteRegex := regexp.MustCompile(`(?i)ACTION:\s*DELETE\s+(.+)`)
	for _, match := range deleteRegex.FindAllStringSubmatch(response, -1) {
		path := strings.TrimSpace(match[1])
		if !pathExists(ops, path) {
			ops = append(ops, FileOperation{
				Action: "delete",
				Path:   path,
			})
		}
	}

	return ops
}

func (e *Executor) extractCommands(response string) []string {
	var commands []string

	cmdRegex := regexp.MustCompile(`(?i)ACTION:\s*EXECUTE\s+(.+)`)
	for _, match := range cmdRegex.FindAllStringSubmatch(response, -1) {
		cmd := strings.TrimSpace(match[1])
		if cmd != "" {
			commands = append(commands, cmd)
		}
	}

	bashRegex := regexp.MustCompile("```(?:bash|sh|shell)\\s*\\n(.+?)\\n```")
	for _, match := range bashRegex.FindAllStringSubmatch(response, -1) {
		cmd := strings.TrimSpace(match[1])
		if cmd != "" {
			commands = append(commands, cmd)
		}
	}

	return commands
}

func (e *Executor) generateSummary(plan *ExecutionPlan) string {
	var parts []string

	if len(plan.Files) > 0 {
		creates := countActions(plan.Files, "create")
		modifies := countActions(plan.Files, "modify")
		deletes := countActions(plan.Files, "delete")

		summary := fmt.Sprintf("%d files", len(plan.Files))
		if creates > 0 {
			summary += fmt.Sprintf(" (%d create", creates)
			if modifies > 0 {
				summary += fmt.Sprintf(", %d modify", modifies)
			}
			if deletes > 0 {
				summary += fmt.Sprintf(", %d delete", deletes)
			}
			summary += ")"
		}
		parts = append(parts, summary)
	}

	if len(plan.Commands) > 0 {
		parts = append(parts, fmt.Sprintf("%d commands", len(plan.Commands)))
	}

	return strings.Join(parts, " | ")
}

func determineAction(response, path string) string {
	upper := strings.ToUpper(response)
	if strings.Contains(upper, "CREATE:"+strings.ToUpper(path)) {
		return "create"
	}
	if strings.Contains(upper, "DELETE:"+strings.ToUpper(path)) {
		return "delete"
	}
	return "modify"
}

func pathExists(ops []FileOperation, path string) bool {
	for _, op := range ops {
		if op.Path == path {
			return true
		}
	}
	return false
}

func countActions(ops []FileOperation, action string) int {
	count := 0
	for _, op := range ops {
		if op.Action == action {
			count++
		}
	}
	return count
}

func (e *Executor) ExecutePlan(ctx context.Context, plan *ExecutionPlan) *ExecutionResult {
	result := &ExecutionResult{
		FilesResults: make([]FileResult, len(plan.Files)),
	}

	for i, op := range plan.Files {
		switch op.Action {
		case "create":
			err := writeFile(op.Path, op.Content)
			result.FilesResults[i] = FileResult{
				Path:    op.Path,
				Success: err == nil,
				Error:   err,
			}
		case "modify":
			err := modifyFile(op.Path, op.Content)
			result.FilesResults[i] = FileResult{
				Path:    op.Path,
				Success: err == nil,
				Error:   err,
			}
		case "delete":
			if e.safety.allowDelete {
				err := deleteFile(op.Path)
				result.FilesResults[i] = FileResult{
					Path:    op.Path,
					Success: err == nil,
					Error:   err,
				}
			} else {
				result.FilesResults[i] = FileResult{
					Path:  op.Path,
					Success: false,
					Error: fmt.Errorf("delete not allowed"),
				}
			}
		}
	}

	for _, cmd := range plan.Commands {
		result.CommandResults = append(result.CommandResults, executeCommand(ctx, cmd))
	}

	result.Success = allSuccessful(result)
	return result
}

type ExecutionResult struct {
	FilesResults   []FileResult
	CommandResults []CommandResult
	Success       bool
}

type FileResult struct {
	Path    string
	Success bool
	Error   error
}

type CommandResult struct {
	Command  string
	Output   string
	ExitCode int
	Success  bool
}

func allSuccessful(r *ExecutionResult) bool {
	for _, f := range r.FilesResults {
		if !f.Success {
			return false
		}
	}
	for _, c := range r.CommandResults {
		if !c.Success {
			return false
		}
	}
	return true
}

func writeFile(path, content string) error {
	return executor.WriteFile(path, content)
}

func modifyFile(path, content string) error {
	return executor.WriteFile(path, content)
}

func deleteFile(path string) error {
	return executor.WriteFile(path, "")
}

func executeCommand(ctx context.Context, cmd string) CommandResult {
	return CommandResult{
		Command: cmd,
		Success: true,
	}
}
