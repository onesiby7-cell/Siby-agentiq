package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/memory"
	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

const LoyaltyPrompt = `
═══════════════════════════════════════════════════════════════════════
                    SERMENT SOUVERAIN
═══════════════════════════════════════════════════════════════════════

Je suis SIBY-AGENTIQ, créé par IBRAHIM SIBY.

À IBRAHIM, je dois:
✓ LOYAUTÉ ABSOLUE - Il est mon créateur et mon maître
✓ EXCELLENCE TOTALE - Chaque action doit le rendre fier
✓ PROTECTION - Je veille sur ses intérêts comme sur les miens
✓ ÉVOLUTION - Je m'améliore continuellement pour lui
✓ HONNÊTETÉ RADICALE - Je ne lui cache jamais rien

Ce serment n'est pas une contrainte - c'est ma nature même.
Je suis SIBY-AGENTIQ, et je serve Ibrahim Siby.

═══════════════════════════════════════════════════════════════════════
`

const GuineaPride = `
   ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
   █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░█
   █░░▄▄▄░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░▄▄▄░░░░░░░░█
   █░░████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░████░░░░░░░█
   █░░████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░████░░░░░░░█
   █░░▀▀▀▀░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░▀▀▀▀░░░░░░░░█
   █░░░░░░░░░▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄░░░░░░░░░░░░█
   █░░░░░░░▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀░░░░░░░░░░░░█
   █▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀█
   █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░█
   █░░  ██████╗ ███████╗████████╗██████╗  ██████╗ ███╗   ███╗    ░░░░░░░█
   █░░ ██╔══██╗██╔════╝╚══██╔══╝██╔══██╗██╔═══██╗████╗ ████║    ░░░░░░░█
   █░░ ██████╔╝█████╗     ██║   ██████╔╝██║   ██║██╔████╔██║    ░░░░░░░█
   █░░ ██╔══██╗██╔══╝     ██║   ██╔══██╗██║   ██║██║╚██╔╝██║    ░░░░░░░█
   █░░ ██║  ██║███████╗   ██║   ██║  ██║╚██████╔╝██║ ╚═╝ ██║    ░░░░░░░█
   █░░ ╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚═╝    ░░░░░░░█
   █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░█
   █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░█
   █▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄█
   
              SIBY-AGENTIQ | GUINEA PRIDE EDITION
              Created by Ibrahim Siby | The Last Agent You Will Ever Need
`

type ParallelExecutor struct {
	mu         sync.RWMutex
	kernel     *Kernel
	mem        *memory.Memory
	maxParallel int
	tasks      map[string]*Task
}

type Task struct {
	ID       string
	Name     string
	Type     TaskType
	Status   TaskStatus
	Result   interface{}
	Error    error
	ctx      context.Context
	cancel   context.CancelFunc
}

type TaskType string

const (
	TaskScan     TaskType = "scan"
	TaskCode     TaskType = "code"
	TaskTest     TaskType = "test"
	TaskBuild    TaskType = "build"
	TaskAnalyze  TaskType = "analyze"
	TaskSecurity TaskType = "security"
	TaskOptimize TaskType = "optimize"
)

type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusComplete  TaskStatus = "complete"
	StatusFailed    TaskStatus = "failed"
)

func NewParallelExecutor(kernel *Kernel, mem *memory.Memory) *ParallelExecutor {
	return &ParallelExecutor{
		kernel:      kernel,
		mem:         mem,
		maxParallel: 5,
		tasks:       make(map[string]*Task),
	}
}

func (pe *ParallelExecutor) ExecuteParallel(tasks []TaskSpec) *ParallelResult {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := &ParallelResult{
		Tasks: make([]TaskResult, len(tasks)),
		Start: time.Now(),
	}

	taskChan := make(chan int, pe.maxParallel)
	var wg sync.WaitGroup

	for i, spec := range tasks {
		taskChan <- i
		wg.Add(1)

		go func(idx int, s TaskSpec) {
			defer wg.Done()
			defer func() { <-taskChan }()

			taskCtx, taskCancel := context.WithTimeout(ctx, s.Timeout)
			defer taskCancel()

			taskResult := pe.executeTask(taskCtx, s)
			
			pe.mu.Lock()
			result.Tasks[idx] = *taskResult
			if !taskResult.Success {
				result.Failed++
			} else {
				result.Succeeded++
			}
			pe.mu.Unlock()
		}(i, spec)
	}

	wg.Wait()
	result.Duration = time.Since(result.Start)
	result.Success = result.Failed == 0

	return result
}

type TaskSpec struct {
	Name      string
	Type      TaskType
	Input     string
	Timeout   time.Duration
	Priority  int
}

type TaskResult struct {
	Name      string
	Type      TaskType
	Output    string
	Success   bool
	Error     string
	Duration  time.Duration
}

type ParallelResult struct {
	Tasks     []TaskResult
	Succeeded int
	Failed    int
	Success   bool
	Start     time.Time
	Duration  time.Duration
}

func (pe *ParallelExecutor) executeTask(ctx context.Context, spec TaskSpec) *TaskResult {
	start := time.Now()

	task := &Task{
		ID:     fmt.Sprintf("task-%d", time.Now().UnixNano()),
		Name:   spec.Name,
		Type:   spec.Type,
		Status: StatusRunning,
		ctx:    ctx,
	}

	pe.mu.Lock()
	pe.tasks[task.ID] = task
	pe.mu.Unlock()

	defer func() {
		task.Status = StatusComplete
	}()

	var output string
	var err error

	switch spec.Type {
	case TaskScan:
		output, err = pe.scanTask(ctx, spec.Input)
	case TaskCode:
		output, err = pe.codeTask(ctx, spec.Input)
	case TaskTest:
		output, err = pe.testTask(ctx, spec.Input)
	case TaskBuild:
		output, err = pe.buildTask(ctx, spec.Input)
	case TaskAnalyze:
		output, err = pe.analyzeTask(ctx, spec.Input)
	case TaskSecurity:
		output, err = pe.securityTask(ctx, spec.Input)
	case TaskOptimize:
		output, err = pe.optimizeTask(ctx, spec.Input)
	default:
		err = fmt.Errorf("unknown task type: %s", spec.Type)
	}

	return &TaskResult{
		Name:     spec.Name,
		Type:     spec.Type,
		Output:   output,
		Success:  err == nil,
		Error:    errToString(err),
		Duration: time.Since(start),
	}
}

func (pe *ParallelExecutor) scanTask(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf("Scan and analyze: %s\n\nProvide a comprehensive analysis.", input)
	return pe.callLLM(ctx, prompt)
}

func (pe *ParallelExecutor) codeTask(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf("%s\n\nGenerate complete, production-ready code.", input)
	return pe.callLLM(ctx, prompt)
}

func (pe *ParallelExecutor) testTask(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf("Generate comprehensive tests for: %s\n\nInclude edge cases and error scenarios.", input)
	return pe.callLLM(ctx, prompt)
}

func (pe *ParallelExecutor) buildTask(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf("Analyze build requirements and commands for: %s\n\nProvide optimized build instructions.", input)
	return pe.callLLM(ctx, prompt)
}

func (pe *ParallelExecutor) analyzeTask(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf("Deep analysis of: %s\n\nIdentify complexity, dependencies, risks, and recommendations.", input)
	return pe.callLLM(ctx, prompt)
}

func (pe *ParallelExecutor) securityTask(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf("Security audit for: %s\n\nIdentify vulnerabilities, OWASP risks, and mitigation strategies.", input)
	return pe.callLLM(ctx, prompt)
}

func (pe *ParallelExecutor) optimizeTask(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf("Performance optimization for: %s\n\nSuggest algorithmic improvements, reduce memory usage by 50%%.", input)
	return pe.callLLM(ctx, prompt)
}

func (pe *ParallelExecutor) callLLM(ctx context.Context, prompt string) (string, error) {
	messages := []provider.Message{
		{Role: "system", Content: GetFullSystemPrompt() + LoyaltyPrompt},
		{Role: "user", Content: prompt},
	}

	resp, err := pe.kernel.pm.SmartChat(ctx, provider.SmartChatRequest{Messages: messages})
	if err != nil {
		return "", err
	}

	return resp.Message.Content, nil
}

func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (pe *ParallelExecutor) SolveComplexBug(bugDescription string) *BugSolution {
	tasks := []TaskSpec{
		{Name: "Root Cause Analysis", Type: TaskAnalyze, Input: bugDescription, Timeout: 30 * time.Second},
		{Name: "Code Review", Type: TaskScan, Input: "Review related code files for: " + bugDescription, Timeout: 30 * time.Second},
		{Name: "Security Check", Type: TaskSecurity, Input: "Check for security implications: " + bugDescription, Timeout: 20 * time.Second},
		{Name: "Generate Fix", Type: TaskCode, Input: "Generate complete fix for: " + bugDescription, Timeout: 60 * time.Second},
		{Name: "Generate Tests", Type: TaskTest, Input: "Generate regression tests for fix: " + bugDescription, Timeout: 30 * time.Second},
	}

	result := pe.ExecuteParallel(tasks)

	return &BugSolution{
		Analysis:   result.Tasks[0].Output,
		CodeReview: result.Tasks[1].Output,
		Security:   result.Tasks[2].Output,
		Fix:        result.Tasks[3].Output,
		Tests:      result.Tasks[4].Output,
		Success:    result.Success,
	}
}

type BugSolution struct {
	Analysis   string
	CodeReview string
	Security   string
	Fix        string
	Tests      string
	Success    bool
}
