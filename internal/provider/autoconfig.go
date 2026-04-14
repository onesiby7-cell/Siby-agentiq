package provider

import (
	"os"
	"sync"

	"github.com/siby-agentiq/siby-agentiq/internal/config"
)

type AutoConfig struct {
	mu sync.RWMutex
	detectedEnvVars map[string]string
	recommendedProvider string
	apiKeysFound []string
}

var autoConfig *AutoConfig

func InitAutoConfig() *AutoConfig {
	autoConfig = &AutoConfig{
		detectedEnvVars: make(map[string]string),
		recommendedProvider: "ollama",
	}
	autoConfig.detectEnvironment()
	return autoConfig
}

func (ac *AutoConfig) detectEnvironment() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	apiKeyVars := []string{
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"GOOGLE_API_KEY",
		"AZURE_OPENAI_KEY",
		"OLLAMA_HOST",
	}

	for _, varName := range apiKeyVars {
		if value := os.Getenv(varName); value != "" {
			ac.detectedEnvVars[varName] = value
			ac.apiKeysFound = append(ac.apiKeysFound, varName)
		}
	}

	if ollamaURL := os.Getenv("OLLAMA_HOST"); ollamaURL == "" {
		ac.detectedEnvVars["OLLAMA_HOST"] = "http://localhost:11434"
	}

	ac.determineRecommendation()
}

func (ac *AutoConfig) determineRecommendation() {
	if _, hasOllama := ac.detectedEnvVars["OLLAMA_HOST"]; hasOllama {
		ac.recommendedProvider = "ollama"
	} else if _, hasAnthropic := ac.detectedEnvVars["ANTHROPIC_API_KEY"]; hasAnthropic {
		ac.recommendedProvider = "anthropic"
	} else if _, hasOpenAI := ac.detectedEnvVars["OPENAI_API_KEY"]; hasOpenAI {
		ac.recommendedProvider = "openai"
	}
}

func (ac *AutoConfig) GetRecommendedProvider() string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.recommendedProvider
}

func (ac *AutoConfig) GetDetectedVariables() map[string]string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range ac.detectedEnvVars {
		result[k] = v
	}
	return result
}

func (ac *AutoConfig) HasAPIKey(provider string) bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	
	switch provider {
	case "anthropic":
		_, ok := ac.detectedEnvVars["ANTHROPIC_API_KEY"]
		return ok
	case "openai":
		_, ok := ac.detectedEnvVars["OPENAI_API_KEY"]
		return ok
	case "ollama":
		_, ok := ac.detectedEnvVars["OLLAMA_HOST"]
		return ok
	}
	return false
}

func (ac *AutoConfig) GenerateAutoConfig() config.ProviderConfig {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	cfg := config.ProviderConfig{
		Active: ac.recommendedProvider,
	}

	if host, ok := ac.detectedEnvVars["OLLAMA_HOST"]; ok {
		cfg.Ollama = config.OllamaConfig{
			Enabled:    true,
			BaseURL:    host,
			Model:      "llama3.2:latest",
			Timeout:    120,
			Stream:     true,
			KeepAlive:  "5m",
		}
	}

	if key, ok := ac.detectedEnvVars["ANTHROPIC_API_KEY"]; ok {
		cfg.Anthropic = config.AnthropicConfig{
			Enabled:     true,
			APIKey:     key,
			Model:      "claude-sonnet-4-20250514",
			MaxTokens:  8192,
			Temperature: 0.7,
		}
	}

	if key, ok := ac.detectedEnvVars["OPENAI_API_KEY"]; ok {
		cfg.OpenAI = config.OpenAIConfig{
			Enabled:    true,
			APIKey:    key,
			BaseURL:   "https://api.openai.com/v1",
			Model:     "gpt-4o",
			Temperature: 0.7,
		}
	}

	return cfg
}

func (ac *AutoConfig) GetWelcomeMessage() string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	var msg string
	
	if len(ac.apiKeysFound) > 0 {
		msg = "Detected API keys:\n"
		for _, key := range ac.apiKeysFound {
			msg += "  • " + key + "\n"
		}
		msg += "\n"
	} else {
		msg = "No cloud API keys detected.\n"
	}

	msg += "Recommended provider: " + ac.recommendedProvider + "\n"
	
	if ac.recommendedProvider == "ollama" {
		msg += "\nTip: Install Ollama for 100% local, free AI.\n"
		msg += "  Linux/Mac: curl -fsSL https://ollama.ai/install.sh | sh\n"
		msg += "  Then: ollama pull llama3.2\n"
	}

	return msg
}
