package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/deepmemory"
	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

type Engine struct {
	id       string
	version  string
	status   EngineStatus
	started  time.Time

	mu       sync.RWMutex
	agents   map[string]*Agent
	tasks    map[string]*Task
	events   chan *Event

	brain    *deepmemory.Brain
	kernel   *Kernel
}

type EngineStatus string

const (
	StatusIdle      EngineStatus = "idle"
	StatusRunning  EngineStatus = "running"
	StatusThinking EngineStatus = "thinking"
	StatusBuilding EngineStatus = "building"
	StatusError   EngineStatus = "error"
)

func NewEngine() *Engine {
	e := &Engine{
		id:      fmt.Sprintf("siby-%d", time.Now().UnixNano()),
		version: "2.0.0-SOVEREIGN",
		status:  StatusIdle,
		agents:  make(map[string]*Agent),
		tasks:   make(map[string]*Task),
		events:  make(chan *Event, 1000),
	}

	e.brain = deepmemory.NewBrain()
	e.kernel = NewKernel(e.brain, nil)

	e.initAgents()

	return e
}

func (e *Engine) initAgents() {
	e.agents["coder"] = NewAgent("coder", "Le Codeur", RoleCoder)
	e.agents["tester"] = NewAgent("tester", "Le Testeur", RoleTester)
	e.agents["searcher"] = NewAgent("searcher", "Le Chercheur", RoleSearcher)
	e.agents["architect"] = NewAgent("architect", "L'Architecte", RoleArchitect)
	e.agents["guardian"] = NewAgent("guardian", "Le Gardien", RoleGuardian)
}

func (e *Engine) Start(ctx context.Context) {
	e.mu.Lock()
	e.status = StatusRunning
	e.started = time.Now()
	e.mu.Unlock()

	e.emit(Event{Type: EventEngineStart, Message: "Engine started"})
}

func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, agent := range e.agents {
		agent.Stop()
	}

	e.status = StatusIdle
	e.emit(Event{Type: EventEngineStop, Message: "Engine stopped"})
}

func (e *Engine) ExecuteTask(ctx context.Context, input string) *TaskResult {
	e.mu.Lock()
	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	task := &Task{
		ID:        taskID,
		Input:     input,
		Status:    TaskPending,
		CreatedAt: time.Now(),
	}
	e.tasks[taskID] = task
	e.mu.Unlock()

	e.emit(Event{Type: EventTaskStart, TaskID: taskID, Message: "Task started: " + input})

	go e.executeTask(ctx, task)

	return &TaskResult{
		TaskID: taskID,
		Status: task.Status,
	}
}

func (e *Engine) ExecuteTaskParallel(ctx context.Context, input string) *TaskResult {
	e.mu.Lock()
	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	task := &Task{
		ID:        taskID,
		Input:     input,
		Status:    TaskPending,
		CreatedAt: time.Now(),
		Parallel:  true,
	}
	e.tasks[taskID] = task
	e.mu.Unlock()

	e.emit(Event{Type: EventTaskStart, TaskID: taskID, Message: "Parallel task started"})

	go e.executeTaskParallel(ctx, task)

	return &TaskResult{
		TaskID: taskID,
		Status: task.Status,
	}
}

func (e *Engine) executeTask(ctx context.Context, task *Task) {
	task.Status = TaskRunning

	phaseCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	for _, agent := range e.agents {
		agent.SetStatus(AgentWorking)
	}

	result, err := e.kernel.Process(phaseCtx, task.Input)

	for _, agent := range e.agents {
		agent.SetStatus(AgentIdle)
	}

	task.Status = TaskComplete
	task.Result = result
	task.Error = err
	task.CompletedAt = time.Now()

	e.brain.Remember(task.Input, result)

	e.emit(Event{
		Type:    EventTaskComplete,
		TaskID:  task.ID,
		Message: fmt.Sprintf("Task completed in %v", task.CompletedAt.Sub(task.CreatedAt)),
	})
}

func (e *Engine) executeTaskParallel(ctx context.Context, task *Task) {
	task.Status = TaskRunning

	phaseCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	results := make(chan *AgentResult, len(e.agents))

	for id, agent := range e.agents {
		wg.Add(1)
		go func(agentID string, a *Agent) {
			defer wg.Done()
			a.SetStatus(AgentWorking)

			result := e.runAgentTask(phaseCtx, agentID, task.Input)

			a.SetStatus(AgentIdle)
			results <- result
		}(id, agent)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var agentResults []*AgentResult
	for r := range results {
		agentResults = append(agentResults, r)
	}

	synthesis := e.synthesizeResults(agentResults)

	task.Status = TaskComplete
	task.Result = synthesis
	task.CompletedAt = time.Now()

	e.brain.Remember(task.Input, synthesis)

	e.emit(Event{
		Type:    EventTaskComplete,
		TaskID:  task.ID,
		Message: fmt.Sprintf("Parallel task completed in %v", task.CompletedAt.Sub(task.CreatedAt)),
	})
}

func (e *Engine) runAgentTask(ctx context.Context, agentID, task string) *AgentResult {
	agent := e.agents[agentID]
	start := time.Now()

	agent.Lock()
	role := agent.Role
	agent.Unlock()

	prompt := agent.BuildPrompt(task)

	messages := []provider.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: task},
	}

	resp, err := e.kernel.pm.SmartChat(ctx, provider.SmartChatRequest{Messages: messages})

	return &AgentResult{
		AgentID:  agentID,
		Agent:    agent.Name,
		Output:   resp.Message.Content,
		Error:    err,
		Duration: time.Since(start),
	}
}

func (e *Engine) synthesizeResults(results []*AgentResult) string {
	var synthesis strings.Builder

	synthesis.WriteString("═══════════════════════════════════════════════════════════\n")
	synthesis.WriteString("          PARALLEL THINKING - SIBY-AGENTIQ\n")
	synthesis.WriteString("═══════════════════════════════════════════════════════════\n\n")

	for _, r := range results {
		synthesis.WriteString(fmt.Sprintf("[%s]\n%s\n\n", r.Agent, r.Output))
	}

	return synthesis.String()
}

func (e *Engine) GetTask(taskID string) *Task {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.tasks[taskID]
}

func (e *Engine) GetAllTasks() []*Task {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tasks := make([]*Task, 0, len(e.tasks))
	for _, t := range e.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

func (e *Engine) GetAgentStatus() map[string]AgentStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := make(map[string]AgentStatus)
	for id, agent := range e.agents {
		agent.Lock()
		status[id] = agent.Status
		agent.Unlock()
	}
	return status
}

func (e *Engine) GetDashboard() *Dashboard {
	e.mu.RLock()
	defer e.mu.RUnlock()

	agents := make([]*AgentInfo, 0, len(e.agents))
	for id, a := range e.agents {
		a.Lock()
		agents = append(agents, &AgentInfo{
			ID:       id,
			Name:     a.Name,
			Role:     string(a.Role),
			Status:   string(a.Status),
			Progress: a.Progress,
		})
		a.Unlock()
	}

	var activeTasks int64
	for _, t := range e.tasks {
		if t.Status == TaskRunning {
			atomic.AddInt64(&activeTasks, 1)
		}
	}

	return &Dashboard{
		ID:        e.id,
		Version:   e.version,
		Status:    string(e.status),
		Uptime:    time.Since(e.started),
		Agents:    agents,
		ActiveTasks: int(activeTasks),
		TotalTasks:  len(e.tasks),
	}
}

func (e *Engine) emit(event *Event) {
	select {
	case e.events <- event:
	default:
	}
}

func (e *Engine) Subscribe() <-chan *Event {
	return e.events
}

import "strings"
