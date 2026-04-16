package orchestrator

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Squad struct {
	ID       string
	Name     string
	Symbol   string
	Category string
	mu       sync.RWMutex

	agents  []*SquadAgent
	bus     *MessageBus
	status  SquadStatus
	load    float32
	tasks   int
	results []*SquadResult
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
	AgentWorkIdle  AgentWorkStatus = "idle"
	AgentWorkBusy  AgentWorkStatus = "busy"
	AgentWorkDone  AgentWorkStatus = "done"
	AgentWorkError AgentWorkStatus = "error"
)

type AgentConfig struct {
	Name      string
	Specialty string
}

type SquadResult struct {
	AgentID  string
	Output   string
	Duration time.Duration
	Success  bool
}

type MessageBus struct {
	messages chan *AgentMessage
	mu       sync.RWMutex
}

type AgentMessage struct {
	From    string
	To      string
	Content string
	Type    string
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		messages: make(chan *AgentMessage, 100),
	}
}

func NewSquad(id, name, category string, agentConfigs []AgentConfig, bus *MessageBus) *Squad {
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

	for i, cfg := range agentConfigs {
		agent := &SquadAgent{
			ID:     fmt.Sprintf("%s-%d", id, i),
			Name:   cfg.Name,
			Role:   cfg.Specialty,
			Status: AgentWorkIdle,
		}
		squad.agents = append(squad.agents, agent)
	}

	return squad
}

func (s *Squad) getSymbol() string {
	symbols := map[string]string{
		"architects": "🏛️",
		"thinkers":   "🧠",
		"stylists":   "🎨",
		"warriors":   "⚔️",
		"builders":   "🔨",
	}
	if sym, ok := symbols[strings.ToLower(s.Category)]; ok {
		return sym
	}
	return "🦂"
}

func (s *Squad) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = SquadActive
}

func (s *Squad) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = SquadIdle
}

func (s *Squad) GetStatus() SquadStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *Squad) GetAgents() []*SquadAgent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agents
}

func (s *Squad) GetResults() []*SquadResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.results
}

func (s *Squad) BroadcastTask(task string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, agent := range s.agents {
		agent.mu.Lock()
		agent.Status = AgentWorkBusy
		agent.Result = ""
		agent.mu.Unlock()

		msg := &AgentMessage{
			From:    "squad",
			To:      agent.ID,
			Content: task,
			Type:    "task",
		}
		s.bus.messages <- msg
	}
}

func (s *Squad) CollectResults() []*SquadResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.results
}

func (s *Squad) Execute(task string) []*SquadResult {
	s.mu.Lock()
	s.status = SquadWorking
	s.mu.Unlock()

	s.BroadcastTask(task)

	time.Sleep(100 * time.Millisecond)

	s.mu.Lock()
	s.status = SquadDone
	s.mu.Unlock()

	return s.GetResults()
}
