package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

type Orchestrator struct {
	id     string
	mu     sync.RWMutex
	status OrchestratorStatus
	bus    *MessageBus
	kernel *Kernel
	squads map[string]*Squad
	pm     *provider.ProviderManager
}

type OrchestratorStatus string

const (
	OrchestratorIdle     OrchestratorStatus = "idle"
	OrchestratorRunning  OrchestratorStatus = "running"
	OrchestratorThinking OrchestratorStatus = "thinking"
	OrchestratorBuilding OrchestratorStatus = "building"
)

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		id:     "main",
		status: OrchestratorIdle,
		bus:    NewMessageBus(),
		squads: make(map[string]*Squad),
	}
}

func (o *Orchestrator) SetProviderManager(pm *provider.ProviderManager) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.pm = pm
	o.kernel = NewKernel(pm)
}

func (o *Orchestrator) Start() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.status = OrchestratorRunning
}

func (o *Orchestrator) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.status = OrchestratorIdle
}

func (o *Orchestrator) GetStatus() OrchestratorStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.status
}

func (o *Orchestrator) RegisterSquad(squad *Squad) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.squads[squad.ID] = squad
}

func (o *Orchestrator) GetSquad(id string) *Squad {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.squads[id]
}

func (o *Orchestrator) ListSquads() []*Squad {
	o.mu.RLock()
	defer o.mu.RUnlock()
	squads := make([]*Squad, 0, len(o.squads))
	for _, s := range o.squads {
		squads = append(squads, s)
	}
	return squads
}

func (o *Orchestrator) Execute(ctx context.Context, task string) (*ExecutionResult, error) {
	o.mu.Lock()
	o.status = OrchestratorThinking
	o.mu.Unlock()

	result := &ExecutionResult{
		Task:     task,
		Start:    time.Now(),
		Duration: 0,
	}

	if o.kernel != nil {
		output, err := o.kernel.Process(ctx, task)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			return result, err
		}
		result.Success = true
		result.Output = output
	} else {
		result.Success = true
		result.Output = fmt.Sprintf("[SIBY] Task executed: %s", task)
	}

	o.mu.Lock()
	o.status = OrchestratorRunning
	o.mu.Unlock()

	result.Duration = time.Since(result.Start)
	return result, nil
}

type ExecutionResult struct {
	Task     string
	Output   string
	Success  bool
	Error    string
	Start    time.Time
	Duration time.Duration
	Squads   int
}

func (o *Orchestrator) CreateSquad(id, name, category string) *Squad {
	configs := []AgentConfig{
		{Name: "Agent-1", Specialty: "general"},
		{Name: "Agent-2", Specialty: "review"},
	}
	squad := NewSquad(id, name, category, configs, o.bus)
	o.RegisterSquad(squad)
	return squad
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
