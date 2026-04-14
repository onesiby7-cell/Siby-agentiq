package executor

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

type ProjectType string

const (
	ProjectGo        ProjectType = "go"
	ProjectNode      ProjectType = "node"
	ProjectRust      ProjectType = "rust"
	ProjectPython    ProjectType = "python"
	ProjectAndroid   ProjectType = "android"
	ProjectFlutter   ProjectType = "flutter"
	ProjectJava      ProjectType = "java"
	ProjectCPP       ProjectType = "cpp"
	ProjectDotnet    ProjectType = "dotnet"
	ProjectUnknown   ProjectType = "unknown"
)

type ProjectDetector struct {
	mu   sync.RWMutex
	root string
}

func NewProjectDetector(root string) *ProjectDetector {
	return &ProjectDetector{root: root}
}

func (pd *ProjectDetector) Detect() (ProjectType, BuildConfig) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	files, _ := filepath.Glob(filepath.Join(pd.root, "*"))
	detected := ProjectUnknown
	var config BuildConfig

	for _, f := range files {
		name := filepath.Base(f)
		switch name {
		case "go.mod":
			detected = ProjectGo
			config = BuildConfig{
				Install:  []string{"go mod download", "go mod tidy"},
				Build:    []string{"go build -o {output} ./..."},
				Test:     []string{"go test ./..."},
				Dev:      "go run ./cmd/...",
				FileExt:  ".go",
			}
		case "Cargo.toml":
			detected = ProjectRust
			config = BuildConfig{
				Install:  []string{"cargo fetch"},
				Build:    []string{"cargo build --release"},
				Test:     []string{"cargo test"},
				Dev:      "cargo run",
				FileExt:  ".rs",
			}
		case "package.json":
			if pd.isFlutter() {
				detected = ProjectFlutter
				config = pd.flutterConfig()
			} else {
				detected = ProjectNode
				config = pd.nodeConfig(f)
			}
		case "build.gradle", "build.gradle.kts":
			detected = ProjectAndroid
			config = pd.androidConfig()
		case "pom.xml":
			detected = ProjectJava
			config = pd.javaConfig()
		case "CMakeLists.txt":
			detected = ProjectCPP
			config = pd.cppConfig()
		case "*.csproj":
			detected = ProjectDotnet
			config = BuildConfig{
				Install:  []string{"dotnet restore"},
				Build:    []string{"dotnet build"},
				Test:     []string{"dotnet test"},
				Dev:      "dotnet run",
				FileExt:  ".cs",
			}
		case "requirements.txt", "pyproject.toml":
			detected = ProjectPython
			config = pd.pythonConfig()
		}
	}

	return detected, config
}

func (pd *ProjectDetector) isFlutter() bool {
	_, err := os.Stat(filepath.Join(pd.root, "pubspec.yaml"))
	return err == nil
}

func (pd *ProjectDetector) flutterConfig() BuildConfig {
	return BuildConfig{
		Install:  []string{"flutter pub get"},
		Build:    []string{"flutter build apk --debug", "flutter build apk --release"},
		Test:     []string{"flutter test"},
		Dev:      "flutter run",
		FileExt:  ".dart",
	}
}

func (pd *ProjectDetector) nodeConfig(pkgJSON string) BuildConfig {
	data, _ := os.ReadFile(pkgJSON)
	isNext := strings.Contains(string(data), "next")
	isReact := strings.Contains(string(data), "react")

	install := []string{"npm install"}
	build := []string{"npm run build"}

	if isNext || isReact {
		build = []string{"npm run build"}
	}

	return BuildConfig{
		Install:  install,
		Build:    build,
		Test:     []string{"npm test"},
		Dev:      "npm run dev",
		FileExt:  ".js",
	}
}

func (pd *ProjectDetector) androidConfig() BuildConfig {
	hasGradleWrapper := pd.fileExists("gradlew")
	buildCmd := "./gradlew assembleDebug"
	if !hasGradleWrapper {
		buildCmd = "gradle assembleDebug"
	}

	return BuildConfig{
		Install:  []string{"./gradlew dependencies", "./gradlew --refresh-dependencies"},
		Build:    []string{buildCmd},
		Test:     []string{"./gradlew test"},
		Dev:      "./gradlew installDebug",
		FileExt:  ".kt",
	}
}

func (pd *ProjectDetector) javaConfig() BuildConfig {
	return BuildConfig{
		Install:  []string{"mvn dependency:resolve"},
		Build:    []string{"mvn clean package"},
		Test:     []string{"mvn test"},
		Dev:      "mvn spring-boot:run",
		FileExt:  ".java",
	}
}

func (pd *ProjectDetector) cppConfig() BuildConfig {
	return BuildConfig{
		Install:  []string{"cmake .", "make"},
		Build:    []string{"make"},
		Test:     []string{},
		Dev:      "./a.out",
		FileExt:  ".cpp",
	}
}

func (pd *ProjectDetector) pythonConfig() BuildConfig {
	return BuildConfig{
		Install:  []string{"pip install -r requirements.txt"},
		Build:    []string{"python setup.py build"},
		Test:     []string{"pytest"},
		Dev:      "python main.py",
		FileExt:  ".py",
	}
}

func (pd *ProjectDetector) fileExists(name string) bool {
	_, err := os.Stat(filepath.Join(pd.root, name))
	return err == nil
}

type BuildConfig struct {
	Install  []string
	Build    []string
	Test     []string
	Dev      string
	FileExt  string
}

type AutonomousExecutor struct {
	detector     *ProjectDetector
	pm           *provider.ProviderManager
	buildConfig  BuildConfig
	maxRetries   int
	confirmFn    func(string) bool
	feedbackFn   func(string)
}

func NewAutonomousExecutor(pm *provider.ProviderManager, workDir string) *AutonomousExecutor {
	return &AutonomousExecutor{
		detector:   NewProjectDetector(workDir),
		pm:         pm,
		maxRetries: 5,
	}
}

func (ae *AutonomousExecutor) SetConfirmFn(fn func(string) bool) {
	ae.confirmFn = fn
}

func (ae *AutonomousExecutor) SetFeedbackFn(fn func(string)) {
	ae.feedbackFn = fn
}

func (ae *AutonomousExecutor) DetectAndConfigure() (ProjectType, BuildConfig) {
	projectType, config := ae.detector.Detect()
	ae.buildConfig = config
	return projectType, config
}

func (ae *AutonomousExecutor) ExecuteBuild(ctx context.Context) (*BuildResult, error) {
	if ae.confirmFn != nil {
		if !ae.confirmFn("Run build for " + ae.detector.root + "?") {
			return &BuildResult{Success: false, Message: "Cancelled by user"}, nil
		}
	}

	ae.feedback("Detected project type: " + string(ae.detector.root))

	var lastErr error
	for attempt := 0; attempt < ae.maxRetries; attempt++ {
		if attempt > 0 {
			ae.feedback(fmt.Sprintf("Retry %d/%d after fix...", attempt+1, ae.maxRetries))
			
			if err := ae.analyzeAndFix(ctx, lastErr); err != nil {
				ae.feedback(fmt.Sprintf("Fix attempt failed: %v", err))
			}
		}

		result, err := ae.runBuildCommands(ctx)
		if err == nil && result.Success {
			return result, nil
		}

		lastErr = err
		if result != nil {
			lastErr = fmt.Errorf("%s: %s", lastErr.Error(), result.Output)
		}
	}

	return &BuildResult{
		Success: false,
		Message: fmt.Sprintf("Failed after %d attempts: %v", ae.maxRetries, lastErr),
	}, lastErr
}

func (ae *AutonomousExecutor) runBuildCommands(ctx context.Context) (*BuildResult, error) {
	var allOutput strings.Builder
	allOutput.WriteString(fmt.Sprintf("Building: %s\n", ae.detector.root))

	for _, cmd := range ae.buildConfig.Install {
		ae.feedback(fmt.Sprintf("  $ %s", cmd))
		result := ae.runCommand(ctx, cmd)
		allOutput.WriteString(result.Output)
		if !result.Success {
			return result, fmt.Errorf("install failed: %s", result.Error)
		}
	}

	for _, cmd := range ae.buildConfig.Build {
		ae.feedback(fmt.Sprintf("  $ %s", cmd))
		result := ae.runCommand(ctx, cmd)
		allOutput.WriteString(result.Output)
		if !result.Success {
			return result, fmt.Errorf("build failed: %s", result.Error)
		}
	}

	allOutput.WriteString("\n✓ Build successful\n")
	return &BuildResult{
		Success: true,
		Output:  allOutput.String(),
		Message: "Build completed successfully",
	}, nil
}

func (ae *AutonomousExecutor) runCommand(ctx context.Context, cmdStr string) *BuildResult {
	start := time.Now()
	
	shell := "/bin/sh"
	shellFlag := "-c"
	if runtime.GOOS == "windows" {
		shell = "cmd.exe"
		shellFlag = "/c"
	}

	cmd := exec.CommandContext(ctx, shell, shellFlag, cmdStr)
	cmd.Dir = ae.detector.root

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	return &BuildResult{
		Command:   cmdStr,
		Output:    string(output),
		ExitCode:  cmd.ProcessState.ExitCode(),
		Duration:  duration,
		Success:   err == nil,
	}
}

func (ae *AutonomousExecutor) analyzeAndFix(ctx context.Context, lastErr error) error {
	if lastErr == nil {
		return nil
	}

	errorMsg := lastErr.Error()
	ae.feedback(fmt.Sprintf("Analyzing error: %s", truncate(errorMsg, 200)))

	prompt := fmt.Sprintf(`Analyze this build error and provide a fix:

Error: %s

Project directory: %s

Return a JSON response:
{
  "analysis": "root cause of the error",
  "fix": "specific command or file change to fix it",
  "verification": "command to verify the fix worked"
}

Only return valid JSON, no markdown.`, errorMsg, ae.detector.root)

	messages := []provider.Message{
		{Role: "system", Content: getFixerSystemPrompt()},
		{Role: "user", Content: prompt},
	}

	resp, err := ae.pm.SmartChat(ctx, provider.SmartChatRequest{Messages: messages})
	if err != nil {
		return fmt.Errorf("LLM analysis failed: %w", err)
	}

	var fix FixSuggestion
	if err := parseFixSuggestion(resp.Message.Content, &fix); err != nil {
		return fmt.Errorf("parse fix failed: %w", err)
	}

	ae.feedback(fmt.Sprintf("Analysis: %s", fix.Analysis))
	ae.feedback(fmt.Sprintf("Applying fix: %s", fix.Fix))

	if fix.Command != "" {
		result := ae.runCommand(ctx, fix.Command)
		if !result.Success {
			return fmt.Errorf("fix command failed: %s", result.Error)
		}
	}

	return nil
}

func (ae *AutonomousExecutor) feedback(msg string) {
	if ae.feedbackFn != nil {
		ae.feedbackFn(msg)
	}
}

type BuildResult struct {
	Command  string
	Output   string
	Error    string
	ExitCode int
	Duration time.Duration
	Success  bool
	Message  string
}

type FixSuggestion struct {
	Analysis   string
	Fix        string
	Command     string
	FilesToCreate []string
	FilesToModify []string
}

func parseFixSuggestion(content string, fix *FixSuggestion) error {
	content = strings.TrimSpace(content)
	
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}") + 1
	if jsonStart == -1 || jsonEnd == 0 {
		fix.Analysis = "Could not parse JSON response"
		fix.Fix = content
		return nil
	}

	jsonStr := content[jsonStart:jsonEnd]

	analysisRe := regexp.MustCompile(`"analysis"\s*:\s*"([^"]*)"`)
	fixRe := regexp.MustCompile(`"fix"\s*:\s*"([^"]*)"`)
	cmdRe := regexp.MustCompile(`"command"\s*:\s*"([^"]*)"`)

	if m := analysisRe.FindStringSubmatch(jsonStr); len(m) > 1 {
		fix.Analysis = m[1]
	}
	if m := fixRe.FindStringSubmatch(jsonStr); len(m) > 1 {
		fix.Fix = m[1]
	}
	if m := cmdRe.FindStringSubmatch(jsonStr); len(m) > 1 {
		fix.Command = m[1]
	}

	if fix.Analysis == "" {
		fix.Analysis = "Error analysis complete"
	}
	if fix.Fix == "" {
		fix.Fix = jsonStr
	}

	return nil
}

func getFixerSystemPrompt() string {
	return `You are an expert build system troubleshooter. Your job is to:

1. ANALYZE the error message to understand the root cause
2. PROVIDE a specific fix (command or file change)
3. SUGGEST verification steps

Be concise and actionable. Focus on the most likely cause.

Common fixes:
- Missing dependencies: "npm install" / "go mod tidy" / "pip install -r requirements.txt"
- Environment issues: "export VAR=value"
- Build tool version: Update build command
- Configuration errors: Fix the config file

Return your response as valid JSON only.`
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (ae *AutonomousExecutor) FullAutoMode(ctx context.Context, task string) (*AutonomousResult, error) {
	ae.feedback("Starting autonomous execution for: " + task)

	projectType, config := ae.DetectAndConfigure()
	ae.feedback(fmt.Sprintf("Detected: %s", projectType))

	messages := []provider.Message{
		{Role: "system", Content: getDeepReasoningPrompt()},
		{Role: "user", Content: fmt.Sprintf("Task: %s\n\nProject: %s\nDirectory: %s\n\n%s", 
			task, projectType, ae.detector.root, ae.detector.root)},
	}

	ae.feedback("Analyzing task with deep reasoning...")
	resp, err := ae.pm.SmartChat(ctx, provider.SmartChatRequest{Messages: messages})
	if err != nil {
		return nil, err
	}

	changes := ParseFilePlan(resp.Message.Content)
	
	result := &AutonomousResult{
		Task:        task,
		ProjectType: projectType,
		Plan:        resp.Message.Content,
		Files:       changes,
	}

	if len(changes) > 0 {
		ae.feedback(fmt.Sprintf("Executing %d file changes...", len(changes)))
		for i, change := range changes {
			ae.feedback(fmt.Sprintf("  [%d/%d] %s %s", i+1, len(changes), change.Action, change.Path))
			if err := WriteFile(change.Path, change.Content); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Write %s failed: %v", change.Path, err))
			}
		}
	}

	buildResult, _ := ae.ExecuteBuild(ctx)
	result.BuildResult = buildResult

	return result, nil
}

type AutonomousResult struct {
	Task        string
	ProjectType ProjectType
	Plan        string
	Files       []FilePlan
	BuildResult *BuildResult
	Errors      []string
}

func getDeepReasoningPrompt() string {
	return `You are Siby-2035, an AI with superhuman coding abilities. Think deeply.

PHASE 1: DECONSTRUCTION
- Map ALL hidden dependencies
- Identify version conflicts before they happen
- Find the smallest change that solves the problem

PHASE 2: SIMULATION
- Anticipate what WILL break the build
- Pre-emptively add error handling
- Test edge cases mentally before writing

PHASE 3: OPTIMIZATION
- Write code that uses 50% less memory
- Use idiomatic patterns for the language
- Remove unnecessary abstractions

PHASE 4: LIBERATION
- Explore RADICAL solutions, not just safe ones
- Question assumptions
- If there's a better architecture, suggest it

When modifying files, use this format:
CREATE: path/to/file
```language
code
```

MODIFY: path/to/file
```language
code
```

DELETE: path/to/file

After code, include:
BUILD_COMMANDS:
- command to build
- command to verify

Think. Then act.`
}
