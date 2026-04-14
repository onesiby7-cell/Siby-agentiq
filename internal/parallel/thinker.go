package parallel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/memory"
	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

type ThinkingAgent struct {
	ID       string
	Name     string
	Role     string
	Status   AgentStatus
	Result   string
	Progress float32
	ctx      context.Context
	cancel   context.CancelFunc
}

type AgentStatus string

const (
	AgentIdle     AgentStatus = "idle"
	AgentThinking AgentStatus = "thinking"
	AgentDone     AgentStatus = "done"
	AgentError    AgentStatus = "error"
)

type ParallelBrain struct {
	mu      sync.RWMutex
	agents  map[string]*ThinkingAgent
	kernel  *Kernel
	mem     *memory.Memory
	results map[string]interface{}
}

type Kernel struct {
	pm *provider.ProviderManager
}

func NewParallelBrain(pm *provider.ProviderManager, mem *memory.Memory) *ParallelBrain {
	pb := &ParallelBrain{
		agents:  make(map[string]*ThinkingAgent),
		mem:     mem,
		results: make(map[string]interface{}),
		kernel:  &Kernel{pm: pm},
	}
	pb.initAgents()
	return pb
}

func (pb *ParallelBrain) initAgents() {
	pb.agents = map[string]*ThinkingAgent{
		"architect": {
			ID:   "architect",
			Name: "L'Architecte",
			Role: "Analyse l'architecture, conçoit les structures optimales",
			Status: AgentIdle,
		},
		"coder": {
			ID:   "coder",
			Name: "Le Codeur",
			Role: "Écrit du code propre, efficace et maintenable",
			Status: AgentIdle,
		},
		"guardian": {
			ID:   "guardian",
			Name: "Le Gardien",
			Role: "Vérifie la sécurité, les tests et les performances",
			Status: AgentIdle,
		},
		"sage": {
			ID:   "sage",
			Name: "Le Sage",
			Role: "Apporte la perspective globale, anticipe les problèmes",
			Status: AgentIdle,
		},
	}
}

func (pb *ParallelBrain) Think(task string) *ThoughtResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pb.mu.Lock()
	for id := range pb.agents {
		pb.agents[id].Status = AgentThinking
		pb.agents[id].ctx, pb.agents[id].cancel = context.WithCancel(ctx)
	}
	pb.mu.Unlock()

	var wg sync.WaitGroup
	results := make(chan *AgentThought, 4)
	errors := make(chan error, 4)

	wg.Add(4)
	go pb.agentThink(ctx, "architect", task, results, errors, &wg)
	go pb.agentThink(ctx, "coder", task, results, errors, &wg)
	go pb.agentThink(ctx, "guardian", task, results, errors, &wg)
	go pb.agentThink(ctx, "sage", task, results, errors, &wg)

	var thoughts []*AgentThought
	completed := 0

	for completed < 4 {
		select {
		case t := <-results:
			thoughts = append(thoughts, t)
			completed++
		case err := <-errors:
			completed++
		case <-ctx.Done():
			break
		}
	}
	wg.Wait()

	pb.mu.Lock()
	for id := range pb.agents {
		if pb.agents[id].cancel != nil {
			pb.agents[id].cancel()
		}
		pb.agents[id].Status = AgentDone
	}
	pb.mu.Unlock()

	return &ThoughtResult{
		Task:    task,
		Thoughts: thoughts,
		Final:   pb.synthesize(ctx, thoughts),
	}
}

func (pb *ParallelBrain) agentThink(ctx context.Context, id, task string, results chan<- *AgentThought, errors chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	start := time.Now()
	agent := pb.getAgent(id)

	prompt := pb.buildAgentPrompt(agent, task)

	messages := []provider.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: task},
	}

	pb.updateProgress(id, 0.3)

	resp, err := pb.kernel.pm.SmartChat(ctx, provider.SmartChatRequest{Messages: messages})

	pb.updateProgress(id, 0.9)

	if err != nil {
		errors <- err
		agent.Status = AgentError
		return
	}

	thought := &AgentThought{
		AgentID:   id,
		AgentName: agent.Name,
		AgentRole: agent.Role,
		Insight:   resp.Message.Content,
		Duration:  time.Since(start),
	}

	results <- thought
	agent.Result = resp.Message.Content
	agent.Status = AgentDone
}

func (pb *ParallelBrain) buildAgentPrompt(agent *ThinkingAgent, task string) string {
	prompts := map[string]string{
		"architect": fmt.Sprintf(`Tu es %s, un expert en architecture logicielle.

Ta mission: Analyser %s sous l'angle architectural.

Questions à explorer:
1. Quelle est la meilleure structure pour ce projet?
2. Quels patterns de design sont les plus appropriés?
3. Comment minimiser les dépendances?
4. Quelle scalabilité est nécessaire?
5. Quels sont les risques architecturaux?

Fournis des recommandations concrètes et justifiées.`, agent.Name, task),

		"coder": fmt.Sprintf(`Tu es %s, un développeur de classe mondiale.

Ta mission: Écrire le code optimal pour %s

Principes:
- Code propre, lisible, idiomatique
- Pas de premature optimization
- Tests unitaires inclus
- Documentation minimale mais efficace
- Respect des best practices du langage

Utilise ce format:
CODE:
```language
// ton code ici
```
END_CODE

EXPLANATION:
[explication concise]`, agent.Name, task),

		"guardian": fmt.Sprintf(`Tu es %s, le gardien de la qualité.

Ta mission: Protéger %s des problèmes

，检查清单:
- Sécurité: injections, authentification, authorization
- Tests: couverture, cas limites, erreurs
- Performance: complexité, mémoire, latence
- Robustesse: gestion d'erreurs, retry, fallbacks

Liste les problèmes potentiels et leurs solutions.`, agent.Name, task),

		"sage": fmt.Sprintf(`Tu es %s, le penseur profond.

Ta mission: Apporter une sagesse ancienne à %s

Réflexions:
1. Cette solution est-elle durable?
2. Quels problèmes voyons-nous venir?
3. Y a-t-il une approche plus simple?
4. Qu'est-ce qu'on apprend de cette tâche?
5. Comment améliorer pour le futur?

Sois philosophique mais pratique.`, agent.Name, task),
	}

	return prompts[id]
}

func (pb *ParallelBrain) synthesize(ctx context.Context, thoughts []*AgentThought) string {
	if len(thoughts) == 0 {
		return "No insights generated."
	}

	var sb strings.Builder
	sb.WriteString("═══════════════════════════════════════════════════════════\n")
	sb.WriteString("                    SYNTHÈSE SIBY-AGENTIQ\n")
	sb.WriteString("═══════════════════════════════════════════════════════════\n\n")

	for _, t := range thoughts {
		sb.WriteString(fmt.Sprintf("[%s - %s]\n%s\n\n", t.AgentName, t.AgentRole, t.Insight))
	}

	return sb.String()
}

func (pb *ParallelBrain) getAgent(id string) *ThinkingAgent {
	pb.mu.RLock()
	defer pb.mu.RUnlock()
	return pb.agents[id]
}

func (pb *ParallelBrain) updateProgress(id string, progress float32) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	if agent, ok := pb.agents[id]; ok {
		agent.Progress = progress
	}
}

func (pb *ParallelBrain) GetStatus() map[string]AgentStatus {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	status := make(map[string]AgentStatus)
	for id, agent := range pb.agents {
		status[id] = agent.Status
	}
	return status
}

type ThoughtResult struct {
	Task    string
	Thoughts []*AgentThought
	Final   string
}

type AgentThought struct {
	AgentID   string
	AgentName string
	AgentRole string
	Insight   string
	Duration  time.Duration
}

import "strings"
