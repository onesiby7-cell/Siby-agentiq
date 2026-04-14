package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/memory"
	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

type Kernel struct {
	mu           sync.RWMutex
	pm           *provider.ProviderManager
	mem          *memory.Memory
	sessionID    string
	turnCount    int
	startTime    time.Time
	context      *ExecutionContext
}

type ExecutionContext struct {
	Task         string
	Phase        ExecutionPhase
	Progress     float32
	FilesModified []string
	LastError    error
	Strategy     string
}

type ExecutionPhase string

const (
	PhaseInit       ExecutionPhase = "init"
	PhaseAnalyze    ExecutionPhase = "analyze"
	PhasePlan       ExecutionPhase = "plan"
	PhaseExecute    ExecutionPhase = "execute"
	PhaseValidate   ExecutionPhase = "validate"
	PhaseComplete   ExecutionPhase = "complete"
	PhaseError      ExecutionPhase = "error"
)

func NewKernel(pm *provider.ProviderManager, mem *memory.Memory) *Kernel {
	return &Kernel{
		pm:        pm,
		mem:       mem,
		sessionID: generateSessionID(),
		startTime: time.Now(),
		context: &ExecutionContext{
			Phase: PhaseInit,
		},
	}
}

func (k *Kernel) ProcessRequest(ctx context.Context, input string) (*Response, error) {
	k.mu.Lock()
	k.turnCount++
	k.context.Task = input
	k.context.Phase = PhaseAnalyze
	k.mu.Unlock()

	priorContext := ""
	if k.mem != nil {
		priorContext = k.mem.GetContextForQuery(input)
	}

	checkPreviousErrors := ""
	if k.mem != nil {
		if fix, found := k.mem.RecallFix(input); found {
			checkPreviousErrors = fmt.Sprintf("\n[SIBY MEMORY] Previous similar error was fixed with: %s\n", fix)
		}
	}

	messages := []provider.Message{
		{Role: "system", Content: GetFullSystemPrompt()},
		{Role: "system", Content: fmt.Sprintf("[CONTEXT] Session: %s | Turn: %d | Time: %s\n%s",
			k.sessionID, k.turnCount, time.Since(k.startTime).String(), priorContext)},
		{Role: "system", Content: checkPreviousErrors},
		{Role: "user", Content: input},
	}

	k.mu.Lock()
	k.context.Phase = PhaseExecute
	k.mu.Unlock()

	ch, err := k.pm.SmartStream(ctx, provider.SmartChatRequest{Messages: messages})
	if err != nil {
		k.setError(err)
		return nil, err
	}

	var response strings.Builder
	for chunk := range ch {
		if chunk.Done {
			break
		}
		response.WriteString(chunk.Content)
	}

	result := response.String()

	if k.mem != nil {
		k.learnFromExecution(input, result)
	}

	k.mu.Lock()
	k.context.Phase = PhaseComplete
	k.context.Progress = 1.0
	k.mu.Unlock()

	return &Response{
		Content:    result,
		SessionID:  k.sessionID,
		TurnCount:  k.turnCount,
		Duration:   time.Since(k.startTime),
	}, nil
}

func (k *Kernel) learnFromExecution(task, response string) {
	if k.mem == nil {
		return
	}

	if strings.Contains(strings.ToLower(response), "error") {
		k.mem.RememberError(task, response)
	}

	if strings.Contains(response, "FILE:") {
		k.mem.RememberDecision(task, "File modification successful")
	}
}

func (k *Kernel) SetStrategy(strategy string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.context.Strategy = strategy
}

func (k *Kernel) GetContext() *ExecutionContext {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.context
}

func (k *Kernel) setError(err error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.context.Phase = PhaseError
	k.context.LastError = err
}

func (k *Kernel) GetStatus() KernelStatus {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return KernelStatus{
		SessionID:    k.sessionID,
		TurnCount:    k.turnCount,
		Uptime:       time.Since(k.startTime),
		CurrentPhase: string(k.context.Phase),
		Progress:     k.context.Progress,
		Strategy:     k.context.Strategy,
	}
}

type Response struct {
	Content   string
	SessionID string
	TurnCount int
	Duration  time.Duration
}

type KernelStatus struct {
	SessionID    string
	TurnCount    int
	Uptime       time.Duration
	CurrentPhase string
	Progress     float32
	Strategy     string
}

func generateSessionID() string {
	return fmt.Sprintf("siby-%d", time.Now().UnixNano())
}

func (k *Kernel) Analyze(task string) (*Analysis, error) {
	prompt := fmt.Sprintf(`Analyse cette tâche en profondeur:

TACHE: %s

Fournis:
1. Complexité (1-10)
2. Dépendances identifiées
3. Risques potentiels
4. Stratégie recommandée
5. Temps estimé

Sois précis et exhaustif.`, task)

	messages := []provider.Message{
		{Role: "system", Content: GetQuickPrompt()},
		{Role: "user", Content: prompt},
	}

	resp, err := k.pm.SmartChat(context.Background(), provider.SmartChatRequest{Messages: messages})
	if err != nil {
		return nil, err
	}

	return &Analysis{
		Task:     task,
		Analysis: resp.Message.Content,
	}, nil
}

type Analysis struct {
	Task           string
	Complexity     int
	Dependencies   []string
	Risks         []string
	Strategy      string
	EstimatedTime  string
	RawAnalysis   string `json:"analysis"`
}

func (k *Kernel) Reset() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.sessionID = generateSessionID()
	k.turnCount = 0
	k.startTime = time.Now()
	k.context = &ExecutionContext{Phase: PhaseInit}
}
