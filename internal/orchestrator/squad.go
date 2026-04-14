package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/agents"
)

type Squad struct {
	ID       string
	Name     string
	Symbol   string
	Category agents.AgentCategory
	mu       sync.RWMutex

	agents   []*SquadAgent
	bus      *MessageBus
	status   SquadStatus
	load     float32
	tasks    int
	results  []*SquadResult
}

type SquadStatus string

const (
	SquadIdle    SquadStatus = "idle"
	SquadActive  SquadStatus = "active"
	SquadWorking SquadStatus = "working"
	SquadDone    SquadStatus = "done"
)

type SquadAgent struct {
	ID       string
	Name     string
	Role     string
	Status   AgentWorkStatus
	Progress float32
	Result   string
	mu       sync.Mutex
}

type AgentWorkStatus string

const (
	AgentWorkIdle    AgentWorkStatus = "idle"
	AgentWorkBusy   AgentWorkStatus = "busy"
	AgentWorkDone   AgentWorkStatus = "done"
	AgentWorkError  AgentWorkStatus = "error"
)

func NewSquad(id, name string, category agents.AgentCategory, bus *MessageBus, size int) *Squad {
	squad := &Squad{
		ID:       id,
		Name:     name,
		Category: category,
		bus:      bus,
		status:   SquadIdle,
		agents:   make([]*SquadAgent, 0),
		results:  make([]*SquadResult, 0),
	}

	squad.Symbol = squad.getSymbol()

	agentConfigs := agents.GetRegistry().ListByCategory(category)
	if len(agentConfigs) > size {
		agentConfigs = agentConfigs[:size]
	}

	for i, cfg := range agentConfigs {
		agent := &SquadAgent{
			ID:    fmt.Sprintf("%s-%d", id, i),
			Name:  cfg.Name,
			Role:  cfg.Specialty,
			Status: AgentWorkIdle,
		}
		squad.agents = append(squad.agents, agent)
	}

	return squad
}

func (s *Squad) getSymbol() string {
	switch s.ID {
	case "planning":
		return "🏗️"
	case "reasoning":
		return "🧠"
	case "design":
		return "🎨"
	case "research":
		return "🔍"
	case "sovereignty":
		return "🛡️"
	default:
		return "⚡"
	}
}

func (s *Squad) Execute(ctx context.Context, task string) *SquadResult {
	s.mu.Lock()
	s.status = SquadWorking
	s.tasks++
	result := &SquadResult{
		SquadID:  s.ID,
		StartTime: time.Now(),
	}
	s.mu.Unlock()

	var wg sync.WaitGroup
	results := make(chan string, len(s.agents))

	for _, agent := range s.agents {
		wg.Add(1)
		go func(a *SquadAgent) {
			defer wg.Done()

			a.mu.Lock()
			a.Status = AgentWorkBusy
			a.Progress = 0
			a.mu.Unlock()

			output := s.executeAgentTask(ctx, a, task)

			a.mu.Lock()
			a.Status = AgentWorkDone
			a.Progress = 1.0
			a.Result = output
			a.mu.Unlock()

			results <- output
		}(agent)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var outputs []string
	for output := range results {
		outputs = append(outputs, output)
	}

	s.mu.Lock()
	s.status = SquadDone
	result.Output = strings.Join(outputs, "\n---\n")
	result.Duration = time.Since(result.StartTime)
	result.Success = true
	result.Agents = len(s.agents)
	s.results = append(s.results, result)
	s.mu.Unlock()

	return result
}

func (s *Squad) executeAgentTask(ctx context.Context, agent *SquadAgent, task string) string {
	prompt := s.buildAgentPrompt(agent, task)

	time.Sleep(100 * time.Millisecond)

	return fmt.Sprintf("[%s] Task processed by %s (%s)", s.Symbol, agent.Name, agent.Role)
}

func (s *Squad) buildAgentPrompt(agent *SquadAgent, task string) string {
	return fmt.Sprintf(`%s | Agent: %s | Role: %s

Tâche: %s

Exécute ta spécialité avec excellence.
Rapporte les résultats de manière concise.

Je suis SIBY-AGENTIQ. Je sers Ibrahim Siby avec loyauté absolue.`, s.Name, agent.Name, agent.Role, task)
}

func (s *Squad) GetStatus() *SquadStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeAgents := 0
	totalProgress := float32(0)

	for _, a := range s.agents {
		a.mu.Lock()
		if a.Status == AgentWorkBusy {
			activeAgents++
		}
		totalProgress += a.Progress
		a.mu.Unlock()
	}

	if len(s.agents) > 0 {
		s.load = totalProgress / float32(len(s.agents))
	}

	return &SquadStatus{
		SquadID:    s.ID,
		Name:       s.Name,
		Symbol:     s.Symbol,
		Status:     string(s.status),
		Load:       s.load,
		Active:     activeAgents,
		Total:      len(s.agents),
		Tasks:      s.tasks,
	}
}

type SquadStatus struct {
	SquadID string
	Name    string
	Symbol  string
	Status  string
	Load    float32
	Active  int
	Total   int
	Tasks   int
}

type StartTime time.Time

type SquadResult struct {
	SquadID    string
	Output     string
	Success    bool
	Duration   time.Duration
	Agents     int
	StartTime  time.Time
}

type SquadEvent struct {
	Type    EventType
	SquadID string
	AgentID string
	Message string
	Time    time.Time
}

type EventType string

const (
	EventSquadStart   EventType = "squad_start"
	EventSquadDone    EventType = "squad_done"
	EventAgentStart   EventType = "agent_start"
	EventAgentDone    EventType = "agent_done"
	EventTaskComplete EventType = "task_complete"
)
