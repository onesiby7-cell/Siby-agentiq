package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/deepmemory"
)

type SelfHealingLoop struct {
	mu         sync.RWMutex
	maxRetries int
	enabled    bool
	brain      *deepmemory.Brain
}

func NewSelfHealingLoop(brain *deepmemory.Brain) *SelfHealingLoop {
	return &SelfHealingLoop{
		maxRetries: 5,
		enabled:    true,
		brain:      brain,
	}
}

type ExecutionResult struct {
	Success     bool
	Output      string
	Error       error
	Attempts    int
	Fixes       []string
	Duration    time.Duration
}

func (sh *SelfHealingLoop) ExecuteWithHealing(ctx context.Context, execFn func() error, task, initialCode string) *ExecutionResult {
	result := &ExecutionResult{
		StartTime: time.Now(),
	}

	currentCode := initialCode
	var lastError error

	for attempt := 0; attempt < sh.maxRetries; attempt++ {
		result.Attempts = attempt + 1

		if attempt > 0 {
			fix := sh.analyzeAndFix(ctx, lastError, currentCode, task)
			result.Fixes = append(result.Fixes, fix)
			currentCode = sh.applyFix(currentCode, fix)
		}

		err := execFn()
		if err == nil {
			result.Success = true
			result.Duration = time.Since(result.StartTime)
			sh.learnSuccess(task, currentCode)
			return result
		}

		lastError = err
		result.Error = err

		sh.learnFromError(task, err, attempt)
	}

	result.Success = false
	result.Duration = time.Since(result.StartTime)
	return result
}

func (sh *SelfHealingLoop) analyzeAndFix(ctx context.Context, err error, code, task string) string {
	if sh.brain != nil {
		if fix := sh.brain.RecallFix(err.Error()); fix != "" {
			return fix
		}
	}

	return fmt.Sprintf(`Analyze this error and provide a fix:

Error: %v

Code:
%s

Task: %s

Return a JSON:
{
  "analysis": "root cause",
  "fix": "specific change needed",
  "code": "corrected code"
}`, err, code, task)
}

func (sh *SelfHealingLoop) applyFix(code, fix string) string {
	return code
}

func (sh *SelfHealingLoop) learnFromError(task string, err error, attempt int) {
	if sh.brain == nil {
		return
	}

	sh.brain.RememberError(
		task,
		fmt.Sprintf("Attempt %d: %v", attempt, err),
	)
}

func (sh *SelfHealingLoop) learnSuccess(task, code string) {
	if sh.brain == nil {
		return
	}

	sh.brain.Remember(task, code)
}

type StartTime time.Time

type BuildExecutor struct {
	mu    sync.RWMutex
	types map[string]*BuildConfig
}

type BuildConfig struct {
	Type        string
	InstallCmd []string
	BuildCmd   []string
	TestCmd    []string
	RunCmd     string
	DetectFile string
}

func NewBuildExecutor() *BuildExecutor {
	be := &BuildExecutor{
		types: make(map[string]*BuildConfig),
	}
	be.registerDefaults()
	return be
}

func (be *BuildExecutor) registerDefaults() {
	be.types["go"] = &BuildConfig{
		Type:      "go",
		InstallCmd: []string{"go mod download", "go mod tidy"},
		BuildCmd:   []string{"go build -o {output} ./..."},
		TestCmd:    []string{"go test ./..."},
		RunCmd:    "go run .",
		DetectFile: "go.mod",
	}

	be.types["node"] = &BuildConfig{
		Type:      "node",
		InstallCmd: []string{"npm install"},
		BuildCmd:   []string{"npm run build"},
		TestCmd:    []string{"npm test"},
		RunCmd:    "npm start",
		DetectFile: "package.json",
	}

	be.types["python"] = &BuildConfig{
		Type:      "python",
		InstallCmd: []string{"pip install -r requirements.txt"},
		BuildCmd:   []string{"python setup.py build"},
		TestCmd:    []string{"pytest"},
		RunCmd:    "python main.py",
		DetectFile: "requirements.txt",
	}

	be.types["rust"] = &BuildConfig{
		Type:      "rust",
		InstallCmd: []string{"cargo fetch"},
		BuildCmd:   []string{"cargo build --release"},
		TestCmd:    []string{"cargo test"},
		RunCmd:    "cargo run",
		DetectFile: "Cargo.toml",
	}

	be.types["android"] = &BuildConfig{
		Type:      "android",
		InstallCmd: []string{"./gradlew dependencies", "./gradlew --refresh-dependencies"},
		BuildCmd:   []string{"./gradlew assembleDebug"},
		TestCmd:    []string{"./gradlew test"},
		RunCmd:    "./gradlew installDebug",
		DetectFile: "build.gradle",
	}

	be.types["flutter"] = &BuildConfig{
		Type:      "flutter",
		InstallCmd: []string{"flutter pub get"},
		BuildCmd:   []string{"flutter build apk --debug", "flutter build apk --release"},
		TestCmd:    []string{"flutter test"},
		RunCmd:    "flutter run",
		DetectFile: "pubspec.yaml",
	}

	be.types["cpp"] = &BuildConfig{
		Type:      "cpp",
		InstallCmd: []string{"cmake .", "make"},
		BuildCmd:   []string{"make"},
		TestCmd:    []string{},
		RunCmd:    "./a.out",
		DetectFile: "CMakeLists.txt",
	}
}

func (be *BuildExecutor) Detect(rootPath string) string {
	for _, cfg := range be.types {
		if fileExists(rootPath + "/" + cfg.DetectFile) {
			return cfg.Type
		}
	}
	return "unknown"
}

func (be *BuildExecutor) Execute(ctx context.Context, buildType, action string) *BuildResult {
	be.mu.RLock()
	cfg, ok := be.types[buildType]
	be.mu.RUnlock()

	if !ok {
		return &BuildResult{
			Success: false,
			Error:   fmt.Errorf("unknown build type: %s", buildType),
		}
	}

	result := &BuildResult{
		StartTime: time.Now(),
		Type:      buildType,
	}

	var cmds []string
	switch action {
	case "install":
		cmds = cfg.InstallCmd
	case "build":
		cmds = cfg.BuildCmd
	case "test":
		cmds = cfg.TestCmd
	default:
		cmds = cfg.BuildCmd
	}

	healer := NewSelfHealingLoop(nil)

	for i, cmd := range cmds {
		execResult := healer.ExecuteWithHealing(ctx, func() error {
			return be.runCommand(ctx, cmd)
		}, fmt.Sprintf("%s %s", buildType, action), "")

		result.Steps = append(result.Steps, &BuildStep{
			Command: cmd,
			Success: execResult.Success,
			Error:   execResult.Error,
			Output:  execResult.Output,
			Attempt: execResult.Attempts,
		})

		if !execResult.Success {
			result.Success = false
			result.Error = execResult.Error
			result.FailedStep = i
			break
		}
	}

	if result.Success && action == "build" {
		result.Artifact = be.findArtifact(buildType)
	}

	result.Duration = time.Since(result.StartTime)

	return result
}

func (be *BuildExecutor) runCommand(ctx context.Context, cmd string) error {
	return nil
}

func (be *BuildExecutor) findArtifact(buildType string) string {
	artifacts := map[string][]string{
		"go":      {"bin/siby", "siby"},
		"node":    {"dist", "build"},
		"android": {"app/build/outputs/apk"},
		"flutter": {"build/app/outputs/flutter-apk"},
		"rust":    {"target/release"},
	}

	if paths, ok := artifacts[buildType]; ok {
		return strings.Join(paths, ", ")
	}
	return ""
}

type BuildResult struct {
	Success     bool
	Type        string
	Steps       []*BuildStep
	Artifact    string
	FailedStep  int
	Error       error
	Duration    time.Duration
	StartTime   time.Time
}

type BuildStep struct {
	Command string
	Success bool
	Error   error
	Output  string
	Attempt int
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

import "os"
