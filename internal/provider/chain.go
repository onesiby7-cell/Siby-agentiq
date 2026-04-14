package provider

import (
	"context"
	"strings"
)

type ChainPhase string

const (
	PhaseSystem     ChainPhase = "system"
	PhaseAnalyze    ChainPhase = "analyze"
	PhasePlan       ChainPhase = "plan"
	PhaseExecute    ChainPhase = "execute"
	PhaseReview     ChainPhase = "review"
	PhaseFinal      ChainPhase = "final"
)

type ChainStep struct {
	Phase     ChainPhase
	Content   string
	Completed bool
}

type ChainBuilder struct {
	cfg           ChainConfig
	systemPrompt  string
	context       string
	steps         []ChainStep
	maxIterations int
}

type ChainConfig struct {
	Enabled        bool
	ReasoningDepth string
	ShowThinking   bool
}

func NewChainBuilder(cfg ChainConfig, systemPrompt, projectContext string) *ChainBuilder {
	return &ChainBuilder{
		cfg:           cfg,
		systemPrompt:  systemPrompt,
		context:       projectContext,
		steps:         make([]ChainStep, 0),
		maxIterations: 5,
	}
}

func (cb *ChainBuilder) BuildInitialMessages(task string) []Message {
	var sb strings.Builder
	sb.WriteString(cb.systemPrompt)
	sb.WriteString("\n\n## Project Context\n")
	sb.WriteString(cb.context)
	sb.WriteString("\n\n## Task\n")
	sb.WriteString(task)
	sb.WriteString("\n\n## Reasoning Mode: ")
	sb.WriteString(cb.cfg.ReasoningDepth)
	
	if cb.cfg.Enabled {
		sb.WriteString("\n\nFollow the Chain of Thought process:")
		if cb.cfg.ReasoningDepth == "deep" {
			sb.WriteString("\n1. [ANALYZE] Understand the problem deeply")
			sb.WriteString("\n2. [PLAN] Create a detailed execution plan")
			sb.WriteString("\n3. [EXECUTE] Implement the solution")
			sb.WriteString("\n4. [REVIEW] Verify and optimize")
		} else if cb.cfg.ReasoningDepth == "medium" {
			sb.WriteString("\n1. [ANALYZE] Quick problem understanding")
			sb.WriteString("\n2. [IMPLEMENT] Direct solution")
		} else {
			sb.WriteString("\n1. [SOLVE] Direct answer")
		}
	}
	return []Message{
		{Role: "system", Content: sb.String()},
		{Role: "user", Content: task},
	}
}

func (cb *ChainBuilder) ExtractPhases(response string) []ChainStep {
	steps := make([]ChainStep, 0)
	phases := []struct {
		prefix  string
		phase   ChainPhase
	}{
		{"[ANALYZE]", PhaseAnalyze},
		{"[ANALYSIS]", PhaseAnalyze},
		{"[PLAN]", PhasePlan},
		{"[PLANNING]", PhasePlan},
		{"[EXECUTE]", PhaseExecute},
		{"[IMPLEMENT]", PhaseExecute},
		{"[REVIEW]", PhaseReview},
		{"[FINAL]", PhaseFinal},
	}
	
	currentPhase := PhaseSystem
	var currentContent strings.Builder
	
	for _, line := range strings.Split(response, "\n") {
		matched := false
		for _, p := range phases {
			if strings.Contains(strings.ToUpper(line), p.prefix) {
				if currentContent.Len() > 0 {
					steps = append(steps, ChainStep{Phase: currentPhase, Content: currentContent.String()})
					currentContent.Reset()
				}
				currentPhase = p.phase
				matched = true
				break
			}
		}
		if !matched {
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}
	
	if currentContent.Len() > 0 {
		steps = append(steps, ChainStep{Phase: currentPhase, Content: currentContent.String()})
	}
	return steps
}

func (cb *ChainBuilder) ShouldContinue(steps []ChainStep) bool {
	if !cb.cfg.Enabled {
		return false
	}
	reviewStep := findStep(steps, PhaseReview)
	return reviewStep == nil || !reviewStep.Completed
}

func findStep(steps []ChainStep, phase ChainPhase) *ChainStep {
	for i := range steps {
		if steps[i].Phase == phase {
			return &steps[i]
		}
	}
	return nil
}

type ContextFormatter struct{}

func NewContextFormatter() *ContextFormatter { return &ContextFormatter{} }

func (cf *ContextFormatter) FormatFileTree(files []FileInfo) string {
	var sb strings.Builder
	sb.WriteString("## Project Structure\n\n")
	for _, f := range files {
		indent := strings.Repeat("  ", f.Depth)
		if f.IsDir {
			sb.WriteString(indent + "📁 " + f.Name + "/\n")
		} else {
			ext := getExt(f.Name)
			icon := getFileIcon(ext)
			sb.WriteString(indent + icon + " " + f.Name + "\n")
		}
	}
	return sb.String()
}

func (cf *ContextFormatter) FormatFileContent(files []FileContent) string {
	var sb strings.Builder
	sb.WriteString("## File Contents\n\n")
	for _, f := range files {
		sb.WriteString("```" + f.Language + " [" + f.Path + "]\n")
		sb.WriteString(f.Content + "\n")
		sb.WriteString("```\n\n")
	}
	return sb.String()
}

type FileInfo struct {
	Name    string
	IsDir   bool
	Depth   int
}

type FileContent struct {
	Path      string
	Content   string
	Language  string
	MaxLines  int
}

func getExt(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

func getFileIcon(ext string) string {
	icons := map[string]string{
		"go":     "🐹",
		"rs":     "🦀",
		"py":     "🐍",
		"ts":     "📘",
		"tsx":    "⚛️",
		"js":     "📜",
		"jsx":    "⚛️",
		"java":   "☕",
		"cpp":    "⚙️",
		"c":      "⚙️",
		"rs":     "🦀",
		"md":     "📝",
		"yaml":   "⚙️",
		"yml":    "⚙️",
		"json":   "📋",
		"toml":   "📋",
		"toml":   "📋",
	}
	if icon, ok := icons[ext]; ok {
		return icon
	}
	return "📄"
}
