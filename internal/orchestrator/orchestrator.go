package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/agents"
	"github.com/siby-agentiq/siby-agentiq/internal/deepmemory"
)

type Orchestrator struct {
	id       string
	mu       sync.RWMutex
	status   OrchestratorStatus

	bus        *MessageBus
	kernel    *Kernel
	brain     *deepmemory.Brain

	squads     map[string]*Squad
	squadOrder []string

	events     chan *SquadEvent
	dashboard *SquadDashboard
}

type OrchestratorStatus string

const (
	OrchestratorIdle       OrchestratorStatus = "idle"
	OrchestratorRunning    OrchestratorStatus = "running"
	OrchestratorThinking   OrchestratorStatus = "thinking"
	OrchestratorBuilding  OrchestratorStatus = "building"
)

type MessageBus struct {
	mu       sync.RWMutex
	channels map[string]chan *Message
}

type Message struct {
	From      string
	To        string
	Type      MessageType
	Payload   interface{}
	Timestamp time.Time
	Ack       bool
}

type MessageType string

const (
	MsgTask         MessageType = "task"
	MsgResult       MessageType = "result"
	MsgHeartbeat    MessageType = "heartbeat"
	MsgStatus       MessageType = "status"
	MsgError        MessageType = "error"
	MsgBroadcast    MessageType = "broadcast"
)

func NewMessageBus() *MessageBus {
	return &MessageBus{
		channels: make(map[string]chan *Message),
	}
}

func (mb *MessageBus) Subscribe(agentID string) chan *Message {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	ch := make(chan *Message, 100)
	mb.channels[agentID] = ch
	return ch
}

func (mb *MessageBus) Unsubscribe(agentID string) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if ch, ok := mb.channels[agentID]; ok {
		close(ch)
		delete(mb.channels, agentID)
	}
}

func (mb *MessageBus) Send(msg *Message) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	if ch, ok := mb.channels[msg.To]; ok {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (mb *MessageBus) Broadcast(msg *Message) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	for agentID, ch := range mb.channels {
		if agentID != msg.From {
			select {
			case ch <- msg:
			default:
			}
		}
	}
}

func NewOrchestrator() *Orchestrator {
	o := &Orchestrator{
		id:    fmt.Sprintf("orch-%d", time.Now().UnixNano()),
		bus:   NewMessageBus(),
		brain: deepmemory.NewBrain(),
		squads: make(map[string]*Squad),
		events: make(chan *SquadEvent, 1000),
		status: OrchestratorIdle,
	}

	o.kernel = NewKernel(o.brain, nil)
	o.dashboard = NewSquadDashboard(o)

	o.initSquads()

	return o
}

func (o *Orchestrator) initSquads() {
	o.squads["planning"] = NewSquad("planning", "SQUAD PLANIFICATION", agents.CatArchitect, o.bus, 10)
	o.squads["reasoning"] = NewSquad("reasoning", "SQUAD RAISONNEMENT", agents.CatThinker, o.bus, 10)
	o.squads["design"] = NewSquad("design", "SQUAD DESIGN & BINAIRE", agents.CatStylist, o.bus, 10)
	o.squads["research"] = NewSquad("research", "SQUAD RECHERCHE", agents.CatScout, o.bus, 5)
	o.squads["sovereignty"] = NewSquad("sovereignty", "SQUAD SOUVERAINETÉ", agents.CatEnforcer, o.bus, 10)

	o.squadOrder = []string{"planning", "reasoning", "design", "research", "sovereignty"}
}

func (o *Orchestrator) Execute(task string) *ExecutionResult {
	o.mu.Lock()
	o.status = OrchestratorRunning
	o.mu.Unlock()

	result := &ExecutionResult{
		Task:      task,
		StartTime: time.Now(),
		Squads:    make(map[string]*SquadResult),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	o.broadcast(&Message{
		Type:    MsgBroadcast,
		Payload: map[string]interface{}{"task": task, "action": "start"},
	})

	var wg sync.WaitGroup
	squadResults := make(chan *SquadResult, len(o.squads))

	for squadID := range o.squads {
		wg.Add(1)
		go func(sID string) {
			defer wg.Done()

			squad := o.squads[sID]
			squadResult := squad.Execute(ctx, task)

			o.broadcast(&Message{
				To:      "dashboard",
				Type:    MsgStatus,
				Payload: squadResult,
			})

			squadResults <- squadResult
		}(squadID)
	}

	go func() {
		wg.Wait()
		close(squadResults)
	}()

	var completed int
	for sr := range squadResults {
		result.Squads[sr.SquadID] = sr
		completed++

		if completed == 3 {
			break
		}
	}

	result.Synthesis = o.synthesize(result)
	result.Duration = time.Since(result.StartTime)
	result.Success = completed > 0

	o.mu.Lock()
	o.status = OrchestratorIdle
	o.mu.Unlock()

	o.broadcast(&Message{
		Type:    MsgBroadcast,
		Payload: map[string]interface{}{"task": task, "action": "complete"},
	})

	return result
}

func (o *Orchestrator) ExecuteSequential(task string) *ExecutionResult {
	o.mu.Lock()
	o.status = OrchestratorRunning
	o.mu.Unlock()

	result := &ExecutionResult{
		Task:      task,
		StartTime: time.Now(),
		Squads:    make(map[string]*SquadResult),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	for i, squadID := range o.squadOrder {
		squad := o.squads[squadID]
		squadResult := squad.Execute(ctx, task)

		result.Squads[squadID] = squadResult
		result.Synthesis += fmt.Sprintf("\n[SQUAD %d: %s]\n%s\n", i+1, squad.Name, squadResult.Output)

		o.dashboard.Update()

		if !squadResult.Success {
			break
		}
	}

	result.Duration = time.Since(result.StartTime)
	result.Success = true

	o.mu.Lock()
	o.status = OrchestratorIdle
	o.mu.Unlock()

	return result
}

func (o *Orchestrator) broadcast(msg *Message) {
	msg.Timestamp = time.Now()
	o.bus.Broadcast(msg)
}

func (o *Orchestrator) synthesize(result *ExecutionResult) string {
	var sb strings.Builder

	sb.WriteString("═══════════════════════════════════════════════════════════\n")
	sb.WriteString("         SYNTHÈSE SIBY-AGENTIQ MULTI-AGENTS\n")
	sb.WriteString("═══════════════════════════════════════════════════════════\n\n")

	sb.WriteString("┌─────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│                    LOYAUTÉ ABSOLUE                      │\n")
	sb.WriteString("│        Je sers Ibrahim Siby avec excellence            │\n")
	sb.WriteString("└─────────────────────────────────────────────────────────┘\n\n")

	for squadID, sr := range result.Squads {
		squad := o.squads[squadID]
		sb.WriteString(fmt.Sprintf("[%s] %s\n", squad.Symbol, squad.Name))
		if sr.Success {
			sb.WriteString("  ✓ Opération réussie\n")
		} else {
			sb.WriteString("  ✗ Opération échouée\n")
		}
		sb.WriteString(fmt.Sprintf("  Output: %s\n\n", truncate(sr.Output, 200)))
	}

	sb.WriteString("═══════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("Durée totale: %v\n", result.Duration))
	sb.WriteString("═══════════════════════════════════════════════════════════\n")

	return sb.String()
}

func (o *Orchestrator) GetDashboard() *SquadDashboard {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.dashboard
}

func (o *Orchestrator) GetSquadStatus() map[string]*SquadStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()

	status := make(map[string]*SquadStatus)
	for id, squad := range o.squads {
		status[id] = squad.GetStatus()
	}
	return status
}

type ExecutionResult struct {
	Task      string
	Squads    map[string]*SquadResult
	Synthesis string
	Duration  time.Duration
	Success   bool
}

type SquadResult struct {
	SquadID  string
	Output   string
	Success  bool
	Duration time.Duration
	Agents   int
}

import "strings"

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
