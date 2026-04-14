package model

import (
	"fmt"
	"sync"
)

type ModelRegistry struct {
	mu      sync.RWMutex
	models  map[string]*ModelInfo
	aliases map[string]string
}

type ModelInfo struct {
	ID          string
	Name        string
	Provider    string
	ContextLen  int
	Speed       string
	Quality     string
	Cost        string
	Freshness   string
	UseCase     string
}

var registry = &ModelRegistry{
	models:  make(map[string]*ModelInfo),
	aliases: make(map[string]string),
}

func init() {
	registerOllamaModels()
	registerGroqModels()
	registerOpenAI()
	registerAnthropic()
	registerOther()
}

func registerOllamaModels() {
	models := []ModelInfo{
		{ID: "llama3.2:1b", Name: "Llama 3.2 1B", Provider: "ollama", ContextLen: 128000, Speed: "ultra", Quality: "basic", Cost: "free", Freshness: "2024", UseCase: "quick-tasks"},
		{ID: "llama3.2:3b", Name: "Llama 3.2 3B", Provider: "ollama", ContextLen: 128000, Speed: "fast", Quality: "medium", Cost: "free", Freshness: "2024", UseCase: "general"},
		{ID: "llama3.2:latest", Name: "Llama 3.2", Provider: "ollama", ContextLen: 128000, Speed: "fast", Quality: "high", Cost: "free", Freshness: "2024", UseCase: "coding"},
		{ID: "llama3.3:70b", Name: "Llama 3.3 70B", Provider: "ollama", ContextLen: 128000, Speed: "medium", Quality: "ultra", Cost: "free", Freshness: "2024", UseCase: "complex-reasoning"},
		{ID: "codellama:7b", Name: "Code Llama 7B", Provider: "ollama", ContextLen: 128000, Speed: "fast", Quality: "high", Cost: "free", Freshness: "2024", UseCase: "code"},
		{ID: "codellama:13b", Name: "Code Llama 13B", Provider: "ollama", ContextLen: 128000, Speed: "medium", Quality: "ultra", Cost: "free", Freshness: "2024", UseCase: "code-advanced"},
		{ID: "mistral:7b", Name: "Mistral 7B", Provider: "ollama", ContextLen: 32000, Speed: "fast", Quality: "high", Cost: "free", Freshness: "2023", UseCase: "balanced"},
		{ID: "mixtral:8x7b", Name: "Mixtral 8x7B", Provider: "ollama", ContextLen: 32000, Speed: "medium", Quality: "ultra", Cost: "free", Freshness: "2024", UseCase: "reasoning"},
		{ID: "phi3:3.8b", Name: "Phi-3 3.8B", Provider: "ollama", ContextLen: 128000, Speed: "ultra", Quality: "medium", Cost: "free", Freshness: "2024", UseCase: "lightweight"},
		{ID: "qwen2.5:7b", Name: "Qwen 2.5 7B", Provider: "ollama", ContextLen: 32000, Speed: "fast", Quality: "high", Cost: "free", Freshness: "2024", UseCase: "multilingual"},
		{ID: "deepseek-coder:6.7b", Name: "DeepSeek Coder 6.7B", Provider: "ollama", ContextLen: 128000, Speed: "fast", Quality: "ultra", Cost: "free", Freshness: "2024", UseCase: "code-expert"},
		{ID: "nomic-embed-text", Name: "Nomic Embed", Provider: "ollama", ContextLen: 8192, Speed: "fast", Quality: "high", Cost: "free", Freshness: "2024", UseCase: "embeddings"},
	}

	for _, m := range models {
		registry.models["ollama/"+m.ID] = &m
	}
}

func registerGroqModels() {
	models := []ModelInfo{
		{ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B Versatile", Provider: "groq", ContextLen: 128000, Speed: "ultra", Quality: "ultra", Cost: "low", Freshness: "2024", UseCase: "all-rounder"},
		{ID: "mixtral-8x7b-32768", Name: "Mixtral 8x7B", Provider: "groq", ContextLen: 32768, Speed: "ultra", Quality: "high", Cost: "low", Freshness: "2024", UseCase: "fast-reasoning"},
		{ID: "llama-3.1-8b-instant", Name: "Llama 3.1 8B Instant", Provider: "groq", ContextLen: 128000, Speed: "ultra", Quality: "medium", Cost: "low", Freshness: "2024", UseCase: "quick"},
		{ID: "gemma2-9b-it", Name: "Gemma 2 9B", Provider: "groq", ContextLen: 8192, Speed: "ultra", Quality: "high", Cost: "low", Freshness: "2024", UseCase: "efficient"},
	}

	for _, m := range models {
		registry.models["groq/"+m.ID] = &m
	}
}

func registerOpenAI() {
	models := []ModelInfo{
		{ID: "gpt-4o", Name: "GPT-4o", Provider: "openai", ContextLen: 128000, Speed: "fast", Quality: "ultra", Cost: "high", Freshness: "2024", UseCase: "all-rounder"},
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Provider: "openai", ContextLen: 128000, Speed: "ultra", Quality: "high", Cost: "low", Freshness: "2024", UseCase: "fast-efficient"},
		{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Provider: "openai", ContextLen: 128000, Speed: "medium", Quality: "ultra", Cost: "high", Freshness: "2024", UseCase: "complex"},
		{ID: "o1-preview", Name: "o1 Preview", Provider: "openai", ContextLen: 128000, Speed: "slow", Quality: "ultra", Cost: "ultra", Freshness: "2024", UseCase: "reasoning"},
		{ID: "o1-mini", Name: "o1 Mini", Provider: "openai", ContextLen: 128000, Speed: "medium", Quality: "high", Cost: "medium", Freshness: "2024", UseCase: "code-reasoning"},
	}

	for _, m := range models {
		registry.models["openai/"+m.ID] = &m
	}
}

func registerAnthropic() {
	models := []ModelInfo{
		{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", Provider: "anthropic", ContextLen: 200000, Speed: "fast", Quality: "ultra", Cost: "high", Freshness: "2024", UseCase: "all-rounder"},
		{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Provider: "anthropic", ContextLen: 200000, Speed: "fast", Quality: "ultra", Cost: "high", Freshness: "2024", UseCase: "coding"},
		{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", Provider: "anthropic", ContextLen: 200000, Speed: "ultra", Quality: "high", Cost: "low", Freshness: "2024", UseCase: "fast"},
		{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", Provider: "anthropic", ContextLen: 200000, Speed: "medium", Quality: "ultra", Cost: "ultra", Freshness: "2024", UseCase: "complex-reasoning"},
	}

	for _, m := range models {
		registry.models["anthropic/"+m.ID] = &m
	}
}

func registerOther() {
	models := []ModelInfo{
		{ID: "claude-sonnet-4-20250514", Name: "Gemini 2.0 Flash", Provider: "google", ContextLen: 1000000, Speed: "ultra", Quality: "high", Cost: "low", Freshness: "2024", UseCase: "multimodal"},
		{ID: "command-r-plus", Name: "Command R+", Provider: "cohere", ContextLen: 128000, Speed: "medium", Quality: "high", Cost: "medium", Freshness: "2024", UseCase: "rag"},
		{ID: "meta-llama/Llama-3.3-70B-Instruct", Name: "Llama 3.3 70B HF", Provider: "together", ContextLen: 128000, Speed: "medium", Quality: "ultra", Cost: "medium", Freshness: "2024", UseCase: "open"},
		{ID: "deepseek-ai/DeepSeek-V3", Name: "DeepSeek V3", Provider: "together", ContextLen: 64000, Speed: "medium", Quality: "ultra", Cost: "low", Freshness: "2024", UseCase: "reasoning"},
	}

	for _, m := range models {
		registry.models["other/"+m.ID] = &m
	}
}

func (r *ModelRegistry) Get(id string) *ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if info, ok := r.models[id]; ok {
		return info
	}

	if alias, ok := r.aliases[id]; ok {
		return r.models[alias]
	}

	for key, info := range r.models {
		if info.ID == id || info.Name == id {
			return info
		}
	}

	return nil
}

func (r *ModelRegistry) List(provider string) []*ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ModelInfo
	for _, m := range r.models {
		if provider == "" || m.Provider == provider {
			info := m
			result = append(result, info)
		}
	}
	return result
}

func (r *ModelRegistry) Add(id string, info *ModelInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[id] = info
}

func (r *ModelRegistry) AddAlias(alias, target string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.aliases[alias] = target
}

func (r *ModelRegistry) Search(query string) []*ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = toLower(query)
	var result []*ModelInfo

	for _, m := range r.models {
		if contains(m.ID, query) || contains(m.Name, query) ||
		   contains(m.Provider, query) || contains(m.UseCase, query) {
			info := m
			result = append(result, info)
		}
	}

	return result
}

func (r *ModelRegistry) GetBest(speed, quality, cost string) *ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var best *ModelInfo
	bestScore := -1

	for _, m := range r.models {
		score := 0
		if speed == "any" || m.Speed == speed {
			score += 2
		}
		if quality == "any" || m.Quality == quality {
			score += 2
		}
		if cost == "any" || m.Cost == cost {
			score++
		}

		if score > bestScore {
			bestScore = score
			best = m
		}
	}

	return best
}

func (r *ModelRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.models)
}

func GetRegistry() *ModelRegistry {
	return registry
}

func ListAllModels() string {
	r := GetRegistry()
	models := r.List("")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Available Models (%d total):\n\n", len(models)))

	byProvider := make(map[string][]*ModelInfo)
	for _, m := range models {
		byProvider[m.Provider] = append(byProvider[m.Provider], m)
	}

	for provider, ms := range byProvider {
		sb.WriteString(fmt.Sprintf("## %s (%d models)\n", strings.ToUpper(provider), len(ms)))
		for _, m := range ms {
			sb.WriteString(fmt.Sprintf("  • %s (%s)\n", m.ID, m.UseCase))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), substr)
}

func toLower(s string) string {
	return strings.ToLower(s)
}

import "strings"
