package engine

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Agent struct {
	mu      sync.Mutex
	ID      string
	Name    string
	Role    AgentRole
	Status  AgentStatus
	Progress float32
	LastRun time.Time
}

type AgentRole string

const (
	RoleCoder    AgentRole = "coder"
	RoleTester   AgentRole = "tester"
	RoleSearcher AgentRole = "searcher"
	RoleArchitect AgentRole = "architect"
	RoleGuardian  AgentRole = "guardian"
)

type AgentStatus string

const (
	AgentIdle    AgentStatus = "idle"
	AgentWorking AgentStatus = "working"
	AgentError  AgentStatus = "error"
	AgentDone   AgentStatus = "done"
)

func NewAgent(id, name string, role AgentRole) *Agent {
	return &Agent{
		ID:     id,
		Name:   name,
		Role:   role,
		Status: AgentIdle,
	}
}

func (a *Agent) SetStatus(status AgentStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Status = status
	a.LastRun = time.Now()
}

func (a *Agent) SetProgress(progress float32) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Progress = progress
}

func (a *Agent) BuildPrompt(task string) string {
	prompts := map[AgentRole]string{
		RoleCoder: fmt.Sprintf(`Tu es %s, expert en programmation.

Génère du code propre, efficace et maintenable.
Langages: Go, Rust, Python, TypeScript, C++, Java, etc.

Code example:
```go
func main() {
    fmt.Println("Hello, World!")
}
```

Réponds avec le code complet.`, a.Name),

		RoleTester: fmt.Sprintf(`Tu es %s, expert en testing.

Écris des tests complets:
- Tests unitaires
- Tests d'intégration
- Tests de performance
- Cas limites

Format:
```go
func TestXxx(t *testing.T) {
    // test
}
````, a.Name),

		RoleSearcher: fmt.Sprintf(`Tu es %s, expert en recherche.

Analyse la tâche et trouve:
- Documentation pertinente
- Solutions sur StackOverflow
- Patterns similaires
- Bonnes pratiques

Fournis les liens et résumés utiles.`, a.Name),

		RoleArchitect: fmt.Sprintf(`Tu es %s, expert en architecture.

Analyse et propose:
- Structure du projet
- Patterns de design
- Dépendances
- Scalabilité

Fournis un diagramme textuel et des recommandations.`, a.Name),

		RoleGuardian: fmt.Sprintf(`Tu es %s, expert en sécurité et qualité.

Vérifie:
- Vulnérabilités (OWASP)
- Performance
- Gestion d'erreurs
- Tests manquants

Liste les problèmes et solutions.`, a.Name),
	}

	return prompts[a.Role]
}

type Task struct {
	ID          string
	Input       string
	Status      TaskStatus
	Parallel    bool
	Result      string
	Error       error
	CreatedAt   time.Time
	CompletedAt time.Time
	SubTasks    []string
}

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskComplete TaskStatus = "complete"
	TaskFailed   TaskStatus = "failed"
	TaskCanceled TaskStatus = "canceled"
)

type AgentResult struct {
	AgentID  string
	Agent    string
	Output   string
	Error    error
	Duration time.Duration
}

type Event struct {
	Type    EventType
	TaskID  string
	AgentID string
	Message string
	Time    time.Time
}

type EventType string

const (
	EventEngineStart    EventType = "engine_start"
	EventEngineStop     EventType = "engine_stop"
	EventTaskStart      EventType = "task_start"
	EventTaskComplete   EventType = "task_complete"
	EventTaskFailed     EventType = "task_failed"
	EventAgentStart     EventType = "agent_start"
	EventAgentComplete  EventType = "agent_complete"
)

type TaskResult struct {
	TaskID  string
	Status  TaskStatus
	Result  string
	Error   error
}

type Dashboard struct {
	ID          string
	Version     string
	Status      string
	Uptime      time.Duration
	Agents      []*AgentInfo
	ActiveTasks int
	TotalTasks  int
	MemoryUsage uint64
}

type AgentInfo struct {
	ID       string
	Name     string
	Role     string
	Status   string
	Progress float32
}

func (d *Dashboard) Format() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("  ╔══════════════════════════════════════════════════════════╗\n")
	sb.WriteString("  ║              SIBY-AGENTIQ DASHBOARD v" + d.Version + "           ║\n")
	sb.WriteString("  ╠══════════════════════════════════════════════════════════╣\n")

	statusIcon := "●"
	if d.Status == "running" {
		statusIcon = "◉"
	} else if d.Status == "error" {
		statusIcon = "✗"
	}

	sb.WriteString(fmt.Sprintf("  ║ %s Status: %-12s | Uptime: %-10s           ║\n",
		statusIcon, d.Status, d.Uptime.Round(time.Second)))
	sb.WriteString("  ╠══════════════════════════════════════════════════════════╣\n")
	sb.WriteString("  ║                    SUB-AGENTS                              ║\n")

	for _, agent := range d.Agents {
		statusIcon := "○"
		if agent.Status == "working" {
			statusIcon = "◐"
		} else if agent.Status == "done" {
			statusIcon = "●"
		} else if agent.Status == "error" {
			statusIcon = "✗"
		}

		progressBar := ""
		progress := int(agent.Progress * 40)
		for i := 0; i < 40; i++ {
			if i < progress {
				progressBar += "█"
			} else {
				progressBar += "░"
			}
		}

		sb.WriteString(fmt.Sprintf("  ║ %s %-10s │ %-8s │ [%s] %3.0f%%    ║\n",
			statusIcon, agent.Name, agent.Role, progressBar, agent.Progress*100))
	}

	sb.WriteString("  ╠══════════════════════════════════════════════════════════╣\n")
	sb.WriteString(fmt.Sprintf("  ║ Tasks: %d active │ %d total                              ║\n",
		d.ActiveTasks, d.TotalTasks))
	sb.WriteString("  ╚══════════════════════════════════════════════════════════╝\n")

	return sb.String()
}
