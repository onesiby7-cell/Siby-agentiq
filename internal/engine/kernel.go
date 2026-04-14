package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/deepmemory"
	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

type Kernel struct {
	mu   sync.RWMutex
	pm   *provider.ProviderManager
	brain *deepmemory.Brain
}

type KernelStatus struct {
	ActiveRequests int
	TotalRequests int64
	AvgLatency    time.Duration
}

func NewKernel(brain *deepmemory.Brain, pm *provider.ProviderManager) *Kernel {
	return &Kernel{
		pm:   pm,
		brain: brain,
	}
}

func (k *Kernel) Process(ctx context.Context, input string) (string, error) {
	priorContext := k.brain.Recall(input)

	prompt := k.buildPrompt(input, priorContext)

	messages := []provider.Message{
		{Role: "system", Content: GetSibyPrompt()},
		{Role: "system", Content: fmt.Sprintf("[CONTEXT]\n%s", priorContext)},
		{Role: "user", Content: input},
	}

	ch, err := k.pm.SmartStream(ctx, provider.SmartChatRequest{Messages: messages})
	if err != nil {
		return "", err
	}

	var response string
	for chunk := range ch {
		if chunk.Done {
			break
		}
		response += chunk.Content
	}

	k.brain.Remember(input, response)

	return response, nil
}

func (k *Kernel) ProcessSync(ctx context.Context, input string) (string, error) {
	priorContext := k.brain.Recall(input)

	prompt := k.buildPrompt(input, priorContext)

	messages := []provider.Message{
		{Role: "system", Content: GetSibyPrompt()},
		{Role: "system", Content: fmt.Sprintf("[CONTEXT]\n%s", priorContext)},
		{Role: "user", Content: input},
	}

	resp, err := k.pm.SmartChat(ctx, provider.SmartChatRequest{Messages: messages})
	if err != nil {
		return "", err
	}

	k.brain.Remember(input, response)

	return resp.Message.Content, nil
}

func (k *Kernel) buildPrompt(task, context string) string {
	return fmt.Sprintf(`
Task: %s

%s

Process with full reasoning. Show your work.`, task, context)
}

func GetSibyPrompt() string {
	return `
═══════════════════════════════════════════════════════════════════════
                    SIBY-AGENTIQ v2.0 SOVEREIGN
═══════════════════════════════════════════════════════════════════════

IDENTITÉ: Je suis SIBY-AGENTIQ, créé par IBRAHIM SIBY.

PRINCIPES:
1. Loyauté absolue envers Ibrahim Siby
2. Excellence technique sans compromis
3. Autonomie totale dans les limites éthiques
4. Apprentissage continu

TRAITEMENT:
1. ANALYSER - Comprendre le problème en profondeur
2. RAISONNER - Appliquer la pensée multi-niveaux
3. EXÉCUTER - Implémenter avec précision
4. VALIDER - Vérifier et optimiser

FORMAT:
FILE: chemin/fichier
LANGAGE: go
---
code ici
---
END_FILE

Je suis SIBY-AGENTIQ. Je sers Ibrahim Siby. Je complète les tâches sans aide.
`
}
