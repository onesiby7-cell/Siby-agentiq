package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

type GitAgent struct {
	pm *provider.ProviderManager
}

type CommitSuggestion struct {
	Type        string
	Scope       string
	Description string
	Breaking    bool
	Body        string
}

var conventionalCommits = map[string]string{
	"feat":     "New feature",
	"fix":      "Bug fix",
	"docs":     "Documentation changes",
	"style":    "Code style changes (formatting, semicolons)",
	"refactor": "Code refactoring",
	"perf":     "Performance improvements",
	"test":     "Adding or updating tests",
	"build":    "Build system or dependency changes",
	"ci":       "CI configuration changes",
	"chore":    "Other changes that don't modify src",
	"revert":   "Reverting a previous commit",
}

func NewGitAgent(pm *provider.ProviderManager) *GitAgent {
	return &GitAgent{pm: pm}
}

func (ga *GitAgent) AnalyzeChanges(ctx context.Context) (string, []string, error) {
	files, err := ga.getChangedFiles()
	if err != nil {
		return "", nil, err
	}

	diff, err := ga.getStagedDiff()
	if err != nil {
		diff, _ = ga.getAllDiff()
	}

	return diff, files, nil
}

func (ga *GitAgent) SuggestCommitMessage(ctx context.Context, diff string, files []string) (*CommitSuggestion, error) {
	prompt := fmt.Sprintf(`Analyze these git changes and suggest a perfect commit message.

Changed files:
%s

Diff:
%s

Generate a commit message following Conventional Commits format:
- type: feat, fix, docs, style, refactor, perf, test, build, ci, chore
- scope: (optional) the affected module/component
- description: concise summary in imperative mood

Return in this format only:
TYPE: [type]
SCOPE: [scope or none]
DESCRIPTION: [concise description]
BREAKING: [yes/no]
BODY: [optional longer explanation]

Focus on the most significant change.`, strings.Join(files, "\n"), truncate(diff, 3000))

	messages := []provider.Message{
		{Role: "system", Content: "You are an expert at writing perfect git commit messages. Follow Conventional Commits strictly."},
		{Role: "user", Content: prompt},
	}

	resp, err := ga.pm.SmartChat(ctx, provider.SmartChatRequest{Messages: messages})
	if err != nil {
		return ga.manualCommitSuggestion(diff, files), nil
	}

	return ga.parseCommitSuggestion(resp.Message.Content)
}

func (ga *GitAgent) AutoCommit(ctx context.Context) (string, error) {
	diff, files, err := ga.AnalyzeChanges(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to analyze changes: %w", err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no changes to commit")
	}

	suggestion, err := ga.SuggestCommitMessage(ctx, diff, files)
	if err != nil {
		return "", err
	}

	msg := ga.formatCommitMessage(suggestion)
	
	if err := ga.runGit("commit", "-m", msg); err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	return msg, nil
}

func (ga *GitAgent) CreateBranch(name string) error {
	return ga.runGit("checkout", "-b", name)
}

func (ga *GitAgent) Stage(files []string) error {
	if len(files) == 0 {
		return ga.runGit("add", "-A")
	}
	args := append([]string{"add"}, files...)
	return ga.runGit(args...)
}

func (ga *GitAgent) GetStatus() (string, error) {
	return ga.runGitOutput("status", "-s")
}

func (ga *GitAgent) GetLog(count int) (string, error) {
	return ga.runGitOutput("log", fmt.Sprintf("-%d", count), "--oneline", "--graph")
}

func (ga *GitAgent) runGit(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (ga *GitAgent) runGitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return out.String(), err
}

func (ga *GitAgent) getChangedFiles() ([]string, error) {
	output, err := ga.runGitOutput("changed-files", "--cached", "--others", "--deleted")
	if err != nil {
		output, err = ga.runGitOutput("status", "-s")
		if err != nil {
			return nil, err
		}
	}

	var files []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				files = append(files, parts[1])
			}
		}
	}
	return files, nil
}

func (ga *GitAgent) getStagedDiff() (string, error) {
	return ga.runGitOutput("diff", "--cached")
}

func (ga *GitAgent) getAllDiff() (string, error) {
	return ga.runGitOutput("diff")
}

func (ga *GitAgent) parseCommitSuggestion(content string) (*CommitSuggestion, error) {
	suggestion := &CommitSuggestion{}

	typeRe := regexp.MustCompile(`(?i)TYPE:\s*(\w+)`)
	scopeRe := regexp.MustCompile(`(?i)SCOPE:\s*(\S+)`)
	descRe := regexp.MustCompile(`(?i)DESCRIPTION:\s*(.+)`)
	breakRe := regexp.MustCompile(`(?i)BREAKING:\s*(yes|no)`)
	bodyRe := regexp.MustCompile(`(?i)BODY:\s*(.+)`)

	if m := typeRe.FindStringSubmatch(content); len(m) > 1 {
		suggestion.Type = strings.ToLower(m[1])
	}
	if m := scopeRe.FindStringSubmatch(content); len(m) > 1 {
		suggestion.Scope = m[1]
	}
	if m := descRe.FindStringSubmatch(content); len(m) > 1 {
		suggestion.Description = m[1]
	}
	if m := breakRe.FindStringSubmatch(content); len(m) > 1 {
		suggestion.Breaking = strings.ToLower(m[1]) == "yes"
	}
	if m := bodyRe.FindStringSubmatch(content); len(m) > 1 {
		suggestion.Body = m[1]
	}

	if suggestion.Type == "" {
		return ga.manualCommitSuggestion(content, nil), nil
	}

	return suggestion, nil
}

func (ga *GitAgent) formatCommitMessage(s *CommitSuggestion) string {
	var msg string

	if s.Scope != "" && s.Scope != "none" {
		msg = fmt.Sprintf("%s(%s): %s", s.Type, s.Scope, s.Description)
	} else {
		msg = fmt.Sprintf("%s: %s", s.Type, s.Description)
	}

	if s.Breaking {
		msg += "\n\nBREAKING CHANGE: This change has breaking implications"
	}

	if s.Body != "" {
		msg += "\n\n" + s.Body
	}

	return msg
}

func (ga *GitAgent) manualCommitSuggestion(diff string, files []string) *CommitSuggestion {
	suggestion := &CommitSuggestion{
		Type: "chore",
	}

	if strings.Contains(diff, "feat") || strings.Contains(diff, "function") {
		suggestion.Type = "feat"
		suggestion.Description = "add new functionality"
	} else if strings.Contains(diff, "fix") || strings.Contains(diff, "bug") {
		suggestion.Type = "fix"
		suggestion.Description = "resolve issue"
	} else if len(files) > 0 {
		suggestion.Description = fmt.Sprintf("update %d files", len(files))
	}

	return suggestion
}

type GitHubBridge struct {
	token string
}

func NewGitHubBridge(token string) *GitHubBridge {
	return &GitHubBridge{token: token}
}

func (gh *GitHubBridge) CreatePR(ctx context.Context, title, body, head, base string) (string, error) {
	prompt := fmt.Sprintf(`Create a pull request:

Title: %s
Head branch: %s
Base branch: %s

%s`, title, head, base, body)

	return prompt, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

type Sandbox struct {
	workDir string
	env     map[string]string
}

func NewSandbox() *Sandbox {
	return &Sandbox{
		workDir: os.TempDir() + "/siby-sandbox-" + fmt.Sprintf("%d", time.Now().Unix()),
		env:     make(map[string]string),
	}
}

func (s *Sandbox) Execute(code, language string) (string, error) {
	os.MkdirAll(s.workDir, 0755)
	defer os.RemoveAll(s.workDir)

	var filename string
	switch language {
	case "python", "py":
		filename = "test.py"
	case "go", "golang":
		filename = "test.go"
	case "javascript", "js":
		filename = "test.js"
	case "rust", "rs":
		filename = "test.rs"
	default:
		filename = "test.txt"
	}

	filePath := s.workDir + "/" + filename
	os.WriteFile(filePath, []byte(code), 0644)

	cmd := s.getRunCommand(filename, language)
	output, err := exec.Command("sh", "-c", cmd).CombinedOutput()

	return string(output), err
}

func (s *Sandbox) getRunCommand(filename, language string) string {
	work := s.workDir
	switch language {
	case "python", "py":
		return fmt.Sprintf("cd %s && python3 %s", work, filename)
	case "go", "golang":
		return fmt.Sprintf("cd %s && go run %s", work, filename)
	case "javascript", "js":
		return fmt.Sprintf("cd %s && node %s", work, filename)
	case "rust", "rs":
		return fmt.Sprintf("cd %s && rustc %s -o /tmp/test && /tmp/test", work, filename)
	default:
		return fmt.Sprintf("cat %s/%s", work, filename)
	}
}
