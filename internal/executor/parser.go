package executor

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type FileChange struct {
	Path     string
	Content  string
	Action   ChangeAction
	Original string
}

type ChangeAction string

const (
	ActionCreate  ChangeAction = "create"
	ActionModify  ChangeAction = "modify"
	ActionDelete  ChangeAction = "delete"
)

type ResponseParser struct{}

func NewResponseParser() *ResponseParser {
	return &ResponseParser{}
}

func (p *ResponseParser) Parse(response string) []FileChange {
	var changes []FileChange
	var currentFile *FileChange
	var inCodeBlock bool
	var codeContent strings.Builder

	fileRegex := regexp.MustCompile(`(?i)^FILE:\s*(.+)$`)
	endRegex := regexp.MustCompile(`(?i)^END_FILE\s*$`)
	actionRegex := regexp.MustCompile(`(?i)^(CREATE|MODIFY|DELETE):\s*(.+)$`)

	scanner := bufio.NewScanner(strings.NewReader(response))
	for scanner.Scan() {
		line := scanner.Text()

		if matches := fileRegex.FindStringSubmatch(line); matches != nil {
			if currentFile != nil && codeContent.Len() > 0 {
				currentFile.Content = codeContent.String()
				changes = append(changes, *currentFile)
			}

			action := ActionModify
			if currentFile == nil {
				action = ActionCreate
			}

			currentFile = &FileChange{
				Path:    strings.TrimSpace(matches[1]),
				Action:  action,
			}
			codeContent.Reset()
			inCodeBlock = true
			continue
		}

		if matches := actionRegex.FindStringSubmatch(line); matches != nil {
			if currentFile != nil {
				switch strings.ToUpper(matches[1]) {
				case "CREATE":
					currentFile.Action = ActionCreate
				case "MODIFY":
					currentFile.Action = ActionModify
				case "DELETE":
					currentFile.Action = ActionDelete
				}
				currentFile.Path = strings.TrimSpace(matches[2])
			}
			continue
		}

		if endRegex.MatchString(line) {
			if currentFile != nil {
				currentFile.Content = codeContent.String()
				changes = append(changes, *currentFile)
				currentFile = nil
			}
			inCodeBlock = false
			codeContent.Reset()
			continue
		}

		if inCodeBlock && currentFile != nil {
			if codeContent.Len() > 0 {
				codeContent.WriteString("\n")
			}
			codeContent.WriteString(line)
		}
	}

	if currentFile != nil && codeContent.Len() > 0 {
		currentFile.Content = codeContent.String()
		changes = append(changes, *currentFile)
	}

	return changes
}

func (p *ResponseParser) ValidatePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("empty file path")
	}

	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if !strings.HasPrefix(absPath, os.Getenv("PWD")) {
		return fmt.Errorf("path outside working directory: %s", path)
	}

	return nil
}

func (p *ResponseParser) ParsePlan(llmResponse string) (Plan, error) {
	var plan Plan

	scanner := bufio.NewScanner(strings.NewReader(llmResponse))
	var currentSection string
	var sectionContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(strings.ToUpper(line), "FILES:"):
			if sectionContent.Len() > 0 && currentSection != "" {
				plan.Sections[currentSection] = sectionContent.String()
				sectionContent.Reset()
			}
			currentSection = "files"

		case strings.HasPrefix(strings.ToUpper(line), "CHANGES:"):
			if sectionContent.Len() > 0 && currentSection != "" {
				plan.Sections[currentSection] = sectionContent.String()
				sectionContent.Reset()
			}
			currentSection = "changes"
			line = strings.TrimPrefix(line, "CHANGES:")

		case strings.HasPrefix(strings.ToUpper(line), "COMMANDS:"):
			if sectionContent.Len() > 0 && currentSection != "" {
				plan.Sections[currentSection] = sectionContent.String()
				sectionContent.Reset()
			}
			currentSection = "commands"
			line = strings.TrimPrefix(line, "COMMANDS:")

		case strings.HasPrefix(strings.ToUpper(line), "RATIONALE:"):
			if sectionContent.Len() > 0 && currentSection != "" {
				plan.Sections[currentSection] = sectionContent.String()
				sectionContent.Reset()
			}
			currentSection = "rationale"
			line = strings.TrimPrefix(line, "RATIONALE:")
		}

		if currentSection != "" {
			if sectionContent.Len() > 0 {
				sectionContent.WriteString("\n")
			}
			sectionContent.WriteString(line)
		}
	}

	if sectionContent.Len() > 0 && currentSection != "" {
		plan.Sections[currentSection] = sectionContent.String()
	}

	if changes := p.Parse(llmResponse); len(changes) > 0 {
		plan.ProposedChanges = changes
	}

	return plan, nil
}

type Plan struct {
	Sections       map[string]string
	ProposedChanges []FileChange
}

func (p *Plan) GetFiles() []string {
	var files []string
	for _, change := range p.ProposedChanges {
		files = append(files, change.Path)
	}
	return files
}

func (p *Plan) Summary() string {
	var parts []string
	for i, change := range p.ProposedChanges {
		parts = append(parts, fmt.Sprintf("%d. %s [%s]", i+1, change.Path, change.Action))
	}
	return strings.Join(parts, "\n")
}

type ExecutionResult struct {
	Changes  []FileChange
	Results  []FileResult
	Success  bool
	Summary  string
}

type FileResult struct {
	Path    string
	Success bool
	Error   error
	Backup  string
}

func (r *ExecutionResult) AddResult(result FileResult) {
	r.Results = append(r.Results, result)
	if !result.Success {
		r.Success = false
	}
}

func (r *ExecutionResult) GetSummary() string {
	var sb strings.Builder
	success := 0
	failed := 0
	for _, res := range r.Results {
		if res.Success {
			success++
		} else {
			failed++
		}
	}
	fmt.Fprintf(&sb, "Completed: %d succeeded, %d failed\n", success, failed)
	return sb.String()
}

func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read error: %w", err)
	}
	return string(data), nil
}

func WriteFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	return nil
}

func CreateBackup(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil
	}

	backupPath := path + ".bak"
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("backup read error: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("backup write error: %w", err)
	}

	return backupPath, nil
}

func RestoreBackup(path string) error {
	backupPath := path + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup found")
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("restore read error: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("restore write error: %w", err)
	}

	os.Remove(backupPath)
	return nil
}

type LineCounter struct {
	total   int
	current int
	reader  io.Reader
}

func CountLines(content string) int {
	return strings.Count(content, "\n") + 1
}
