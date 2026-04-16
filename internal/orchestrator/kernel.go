package orchestrator

import (
	"context"
	"fmt"

	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

type Kernel struct {
	mu *MemoryBrain
	pm *provider.ProviderManager
}

type MemoryBrain struct {
	memory map[string]string
}

func NewMemoryBrain() *MemoryBrain {
	return &MemoryBrain{
		memory: make(map[string]string),
	}
}

func (m *MemoryBrain) Remember(key, value string) {
	m.memory[key] = value
}

func (m *MemoryBrain) Recall(key string) string {
	return m.memory[key]
}

func NewKernel(pm *provider.ProviderManager) *Kernel {
	return &Kernel{
		mu: NewMemoryBrain(),
		pm: pm,
	}
}

func (k *Kernel) Process(ctx context.Context, input string) (string, error) {
	if k.pm == nil {
		return fmt.Sprintf("[SIBY] Task received: %s", input), nil
	}

	priorContext := ""
	if k.mu != nil {
		priorContext = k.mu.Recall(input)
	}

	messages := []provider.Message{
		{Role: "system", Content: GetSibyCore()},
		{Role: "system", Content: fmt.Sprintf("[CONTEXT]\n%s", priorContext)},
		{Role: "user", Content: input},
	}

	resp, err := k.pm.SmartChat(ctx, provider.SmartChatRequest{Messages: messages})
	if err != nil {
		return "", err
	}

	if k.mu != nil {
		k.mu.Remember(input, resp.Message.Content)
	}

	return resp.Message.Content, nil
}

func GetSibyCore() string {
	return `
═══════════════════════════════════════════════════════════════════════
                    🦂 SIBY-AGENTIQ SOVEREIGN 🦂
═══════════════════════════════════════════════════════════════════════

  Advanced Multi-Agent Orchestration System
  Powered by Ibrahim Siby
  
  Features:
    • 45 Specialized Sub-Agents
    • Chain-of-Thought Reasoning
    • Multi-Provider Support
    • Self-Learning Capabilities
    
═══════════════════════════════════════════════════════════════════════
`
}
