package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

type Kernel struct {
	mu sync.RWMutex
	pm *provider.ProviderManager
}

type KernelStatus struct {
	ActiveRequests int
	TotalRequests  int64
	AvgLatency     time.Duration
}

func NewKernel(pm *provider.ProviderManager) *Kernel {
	return &Kernel{
		pm: pm,
	}
}

func (k *Kernel) Process(ctx context.Context, input string) (string, error) {
	prompt := k.buildPrompt(input, "")

	messages := []provider.Message{
		{Role: "system", Content: GetSibyPrompt()},
		{Role: "user", Content: prompt},
	}

	ch, err := k.pm.SmartStream(ctx, provider.SmartChatRequest{Messages: messages})
	if err != nil {
		return "", err
	}

	var response string
	for chunk := range ch {
		response += chunk.Content
	}

	return response, nil
}

func (k *Kernel) buildPrompt(input, context string) string {
	if context != "" {
		return fmt.Sprintf("%s\n\nContext:\n%s", input, context)
	}
	return input
}

func GetSibyPrompt() string {
	return `You are Siby-Agentiq, an advanced AI coding assistant created by Ibrahim Siby.

Capabilities:
- Write, modify, and analyze code in multiple languages
- Execute commands and run tests
- Search the web for documentation and solutions
- Work with files and projects

Always provide clear, working code and explanations.`
}

func (k *Kernel) GetStatus() *KernelStatus {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return &KernelStatus{
		ActiveRequests: 0,
		TotalRequests:  0,
		AvgLatency:     0,
	}
}
