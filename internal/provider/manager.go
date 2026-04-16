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
		providers:  make(map[string]Provider),
		activeName: cfg.Active,
		cfg:        cfg,
		fallbacks:  []string{"ollama", "groq", "openai", "anthropic"},
	}
	return pm
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
		if pm.providers[name] != nil && pm.providers[name].IsAvailable() {
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
			if fallback != nil && fallback.IsAvailable() {
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
