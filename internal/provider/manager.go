package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/config"
)

type ProviderManager struct {
	mu         sync.RWMutex
	providers  map[string]Provider
	activeName string
	fallbacks  []string
	cfg        config.ProviderConfig
}

func NewProviderManager(cfg config.ProviderConfig) *ProviderManager {
	pm := &ProviderManager{
		providers: make(map[string]Provider),
		activeName: cfg.Active,
		cfg:        cfg,
	}
	pm.registerProviders()
	pm.buildFallbackChain()
	return pm
}

func (pm *ProviderManager) registerProviders() {
	if pm.cfg.Ollama.Enabled {
		pm.providers["ollama"] = NewOllamaProvider(pm.cfg.Ollama)
	}
	if pm.cfg.Groq.Enabled {
		pm.providers["groq"] = NewGroqProvider(pm.cfg.Groq)
	}
	if pm.cfg.Anthropic.Enabled {
		pm.providers["anthropic"] = NewAnthropicProvider(pm.cfg.Anthropic)
	}
	if pm.cfg.OpenAI.Enabled {
		pm.providers["openai"] = NewOpenAIProvider(pm.cfg.OpenAI)
	}
}

func (pm *ProviderManager) buildFallbackChain() {
	var ranked []struct {
		name     string
		provider Provider
		priority int
	}
	for name, p := range pm.providers {
		ranked = append(ranked, struct {
			name     string
			provider Provider
			priority int
		}{name, p, p.Priority()})
	}
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].priority < ranked[i].priority {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
	pm.fallbacks = make([]string, len(ranked))
	for i, r := range ranked {
		pm.fallbacks[i] = r.name
	}
}

func (pm *ProviderManager) GetActiveProvider() Provider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.providers[pm.activeName]
}

func (pm *ProviderManager) GetActiveName() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.activeName
}

func (pm *ProviderManager) SwitchProvider(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if _, ok := pm.providers[name]; !ok {
		return fmt.Errorf("unknown provider: %s", name)
	}
	pm.activeName = name
	return nil
}

func (pm *ProviderManager) ListProviders() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	names := make([]string, 0, len(pm.providers))
	for name := range pm.providers {
		names = append(names, name)
	}
	return names
}

func (pm *ProviderManager) CheckAllAvailability() map[string]bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	status := make(map[string]bool)
	for name, provider := range pm.providers {
		status[name] = provider.IsAvailable()
	}
	return status
}

func (pm *ProviderManager) GetBestAvailable() Provider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for _, name := range pm.fallbacks {
		if pm.providers[name].IsAvailable() {
			return pm.providers[name]
		}
	}
	return nil
}

func (pm *ProviderManager) SmartSwitch() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	available := pm.GetBestAvailable()
	if available == nil {
		return fmt.Errorf("no provider available")
	}
	pm.activeName = available.Name()
	return nil
}

type SmartChatRequest struct {
	Messages []Message
	UseChain bool
	Timeout  time.Duration
}

func (pm *ProviderManager) SmartChat(ctx context.Context, req SmartChatRequest) (*ChatResponse, error) {
	provider := pm.GetActiveProvider()
	if provider == nil || !provider.IsAvailable() {
		if err := pm.SmartSwitch(); err != nil {
			return nil, fmt.Errorf("no provider available: %w", err)
		}
		provider = pm.GetActiveProvider()
	}
	resp, err := provider.Chat(ctx, req.Messages)
	if err != nil {
		for _, fallbackName := range pm.fallbacks {
			if fallbackName == pm.activeName {
				continue
			}
			fallback := pm.providers[fallbackName]
			if fallback.IsAvailable() {
				pm.activeName = fallbackName
				return fallback.Chat(ctx, req.Messages)
			}
		}
		return nil, err
	}
	return resp, nil
}

func (pm *ProviderManager) SmartStream(ctx context.Context, req SmartChatRequest) (<-chan StreamChunk, error) {
	provider := pm.GetActiveProvider()
	if provider == nil || !provider.IsAvailable() {
		if err := pm.SmartSwitch(); err != nil {
			return nil, fmt.Errorf("no provider available: %w", err)
		}
		provider = pm.GetActiveProvider()
	}
	return provider.ChatStream(ctx, req.Messages)
}
