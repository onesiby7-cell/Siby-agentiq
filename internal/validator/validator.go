package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

type APIValidator struct {
	mu          sync.RWMutex
	providers   map[string]*ProviderStatus
	httpClient  *http.Client
	fallbackOrder []string
}

type ProviderStatus struct {
	Name       string
	APIKey     string
	Endpoint   string
	Available  bool
	Valid     bool
	QuotaUsed float64
	QuotaMax  float64
	Latency   time.Duration
	LastCheck time.Time
	Error     string
}

type ValidationResult struct {
	Provider   string
	Valid      bool
	HasQuota   bool
	Remaining  float64
	Latency    time.Duration
	Error      string
	Fallback   string
}

var providerConfigs = map[string]struct {
	Endpoint   string
	QuotaLimit float64
	Headers    map[string]string
}{
	"ollama": {
		Endpoint:   "http://localhost:11434/api/tags",
		QuotaLimit: 0,
	},
	"groq": {
		Endpoint: "https://api.groq.com/openai/v1/models",
		QuotaLimit: 10000,
		Headers: map[string]string{
			"Authorization": "Bearer ",
		},
	},
	"openai": {
		Endpoint: "https://api.openai.com/v1/models",
		QuotaLimit: 100000,
		Headers: map[string]string{
			"Authorization": "Bearer ",
		},
	},
	"anthropic": {
		Endpoint: "https://api.anthropic.com/v1/models",
		QuotaLimit: 100000,
		Headers: map[string]string{
			"x-api-key": "",
		},
	},
}

func NewAPIValidator() *APIValidator {
	v := &APIValidator{
		providers: make(map[string]*ProviderStatus),
		httpClient: &http.Client{Timeout: 5 * time.Second},
		fallbackOrder: []string{"ollama", "groq", "openai", "anthropic"},
	}

	v.initProviders()
	return v
}

func (v *APIValidator) initProviders() {
	v.providers["ollama"] = &ProviderStatus{Name: "ollama"}
	v.providers["groq"] = &ProviderStatus{Name: "groq", Endpoint: providerConfigs["groq"].Endpoint}
	v.providers["openai"] = &ProviderStatus{Name: "openai", Endpoint: providerConfigs["openai"].Endpoint}
	v.providers["anthropic"] = &ProviderStatus{Name: "anthropic", Endpoint: providerConfigs["anthropic"].Endpoint}
}

func (v *APIValidator) LoadAPIKeys() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.providers["groq"].APIKey = os.Getenv("GROQ_API_KEY")
	v.providers["openai"].APIKey = os.Getenv("OPENAI_API_KEY")
	v.providers["anthropic"].APIKey = os.Getenv("ANTHROPIC_API_KEY")
}

func (v *APIValidator) ValidateAll(ctx context.Context) map[string]*ValidationResult {
	v.LoadAPIKeys()

	results := make(map[string]*ValidationResult)
	
	v.validateOllama(ctx, results)
	v.validateGroq(ctx, results)
	v.validateOpenAI(ctx, results)
	v.validateAnthropic(ctx, results)

	return results
}

func (v *APIValidator) validateOllama(ctx context.Context, results map[string]*ValidationResult) {
	start := time.Now()
	
	resp, err := v.httpClient.Get(providerConfigs["ollama"].Endpoint)
	latency := time.Since(start)

	status := v.providers["ollama"]
	status.LastCheck = time.Now()
	status.Latency = latency

	if err == nil && resp.StatusCode == 200 {
		status.Available = true
		status.Valid = true
		results["ollama"] = &ValidationResult{
			Provider:  "ollama",
			Valid:     true,
			HasQuota:  false,
			Remaining: 0,
			Latency:   latency,
		}
	} else {
		status.Available = false
		status.Valid = false
		status.Error = err.Error()
		results["ollama"] = &ValidationResult{
			Provider: "ollama",
			Valid:   false,
			Error:   err.Error(),
		}
	}
}

func (v *APIValidator) validateGroq(ctx context.Context, results map[string]*ValidationResult) {
	start := time.Now()
	status := v.providers["groq"]
	
	if status.APIKey == "" {
		results["groq"] = &ValidationResult{Provider: "groq", Valid: false, Error: "No API key"}
		return
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", status.Endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+status.APIKey)

	resp, err := v.httpClient.Do(req)
	latency := time.Since(start)

	status.LastCheck = time.Now()
	status.Latency = latency

	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			status.Available = true
			status.Valid = true
			results["groq"] = &ValidationResult{
				Provider:  "groq",
				Valid:     true,
				HasQuota:  true,
				Remaining: providerConfigs["groq"].QuotaLimit,
				Latency:   latency,
				Fallback:  v.getFallback("groq"),
			}
		} else {
			status.Valid = false
			results["groq"] = &ValidationResult{
				Provider: "groq",
				Valid:   false,
				Error:   fmt.Sprintf("HTTP %d", resp.StatusCode),
			}
		}
	} else {
		status.Valid = false
		status.Error = err.Error()
		results["groq"] = &ValidationResult{
			Provider: "groq",
			Valid:   false,
			Error:   err.Error(),
		}
	}
}

func (v *APIValidator) validateOpenAI(ctx context.Context, results map[string]*ValidationResult) {
	start := time.Now()
	status := v.providers["openai"]
	
	if status.APIKey == "" {
		results["openai"] = &ValidationResult{Provider: "openai", Valid: false, Error: "No API key"}
		return
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", status.Endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+status.APIKey)

	resp, err := v.httpClient.Do(req)
	latency := time.Since(start)

	status.LastCheck = time.Now()
	status.Latency = latency

	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			status.Available = true
			status.Valid = true
			results["openai"] = &ValidationResult{
				Provider:  "openai",
				Valid:     true,
				HasQuota:  true,
				Remaining: providerConfigs["openai"].QuotaLimit,
				Latency:   latency,
				Fallback:  v.getFallback("openai"),
			}
		} else {
			status.Valid = false
			results["openai"] = &ValidationResult{Provider: "openai", Valid: false, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
		}
	} else {
		status.Valid = false
		status.Error = err.Error()
		results["openai"] = &ValidationResult{Provider: "openai", Valid: false, Error: err.Error()}
	}
}

func (v *APIValidator) validateAnthropic(ctx context.Context, results map[string]*ValidationResult) {
	start := time.Now()
	status := v.providers["anthropic"]
	
	if status.APIKey == "" {
		results["anthropic"] = &ValidationResult{Provider: "anthropic", Valid: false, Error: "No API key"}
		return
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", status.Endpoint, nil)
	req.Header.Set("x-api-key", status.APIKey)

	resp, err := v.httpClient.Do(req)
	latency := time.Since(start)

	status.LastCheck = time.Now()
	status.Latency = latency

	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			status.Available = true
			status.Valid = true
			results["anthropic"] = &ValidationResult{
				Provider:  "anthropic",
				Valid:     true,
				HasQuota:  true,
				Remaining: providerConfigs["anthropic"].QuotaLimit,
				Latency:   latency,
				Fallback:  v.getFallback("anthropic"),
			}
		} else {
			status.Valid = false
			results["anthropic"] = &ValidationResult{Provider: "anthropic", Valid: false, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
		}
	} else {
		status.Valid = false
		status.Error = err.Error()
		results["anthropic"] = &ValidationResult{Provider: "anthropic", Valid: false, Error: err.Error()}
	}
}

func (v *APIValidator) getFallback(provider string) string {
	for i, p := range v.fallbackOrder {
		if p == provider && i < len(v.fallbackOrder)-1 {
			return v.fallbackOrder[i+1]
		}
	}
	return "ollama"
}

func (v *APIValidator) GetBestProvider(ctx context.Context) string {
	results := v.ValidateAll(ctx)
	
	for _, p := range v.fallbackOrder {
		if r, ok := results[p]; ok && r.Valid {
			return p
		}
	}
	return "none"
}

func (v *APIValidator) GetStatus() map[string]*ProviderStatus {
	v.mu.RLock()
	defer v.mu.RUnlock()

	status := make(map[string]*ProviderStatus)
	for k, p := range v.providers {
		status[k] = p
	}
	return status
}

type CostTracker struct {
	mu           sync.RWMutex
	usage        map[string]*CostEntry
	dailyLimit   float64
	dailySpent   float64
	lastReset    time.Time
}

type CostEntry struct {
	Provider    string
	Model       string
	InputTokens int
	OutputTokens int
	CostPer1K   float64
	TotalCost   float64
	Timestamp   time.Time
}

var pricing = map[string]map[string]float64{
	"openai": {
		"gpt-4":          0.03,
		"gpt-4-turbo":    0.01,
		"gpt-3.5-turbo":  0.002,
	},
	"anthropic": {
		"claude-3-opus":  0.015,
		"claude-3-sonnet": 0.003,
		"claude-3-haiku":  0.00025,
	},
	"groq": {
		"llama-3-70b":    0.0007,
		"mixtral-8x7b":   0.00024,
	},
}

func NewCostTracker() *CostTracker {
	return &CostTracker{
		usage:      make(map[string]*CostEntry),
		dailyLimit: 10.0,
		lastReset:  time.Now(),
	}
}

func (ct *CostTracker) Record(provider, model string, inputTokens, outputTokens int) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	costPer1K := pricing[provider][model]
	if costPer1K == 0 {
		costPer1K = 0.001
	}

	totalCost := (float64(inputTokens) + float64(outputTokens)) / 1000.0 * costPer1K

	entry := &CostEntry{
		Provider:     provider,
		Model:        model,
		InputTokens:   inputTokens,
		OutputTokens: outputTokens,
		CostPer1K:    costPer1K,
		TotalCost:    totalCost,
		Timestamp:    time.Now(),
	}

	key := fmt.Sprintf("%s-%s-%d", provider, model, time.Now().Unix())
	ct.usage[key] = entry
	ct.dailySpent += totalCost

	if time.Since(ct.lastReset) > 24*time.Hour {
		ct.dailySpent = 0
		ct.lastReset = time.Now()
	}
}

func (ct *CostTracker) GetDailySpent() float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.dailySpent
}

func (ct *CostTracker) GetDailyLimit() float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.dailyLimit
}

func (ct *CostTracker) SetDailyLimit(limit float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.dailyLimit = limit
}

func (ct *CostTracker) IsOverLimit() bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.dailySpent >= ct.dailyLimit
}

func (ct *CostTracker) GetUsageByProvider() map[string]float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	byProvider := make(map[string]float64)
	for _, entry := range ct.usage {
		byProvider[entry.Provider] += entry.TotalCost
	}
	return byProvider
}

func (ct *CostTracker) ExportJSON() string {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	data := map[string]interface{}{
		"daily_spent":  ct.dailySpent,
		"daily_limit":  ct.dailyLimit,
		"last_reset":   ct.lastReset,
		"usage":        ct.usage,
		"by_provider":  ct.GetUsageByProvider(),
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonData)
}

type TokenCounter struct {
	mu            sync.RWMutex
	maxTokens     int
	currentTokens int
	usagePercent  float64
	warnAtPercent float64
	summarizeAtPercent float64
}

func NewTokenCounter(maxTokens int) *TokenCounter {
	return &TokenCounter{
		maxTokens:           maxTokens,
		warnAtPercent:       0.75,
		summarizeAtPercent:  0.90,
	}
}

func (tc *TokenCounter) AddTokens(count int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.currentTokens += count
	tc.usagePercent = float64(tc.currentTokens) / float64(tc.maxTokens)
}

func (tc *TokenCounter) SetTokens(count int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.currentTokens = count
	tc.usagePercent = float64(tc.currentTokens) / float64(tc.maxTokens)
}

func (tc *TokenCounter) GetUsage() (current, max int, percent float64) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.currentTokens, tc.maxTokens, tc.usagePercent
}

func (tc *TokenCounter) NeedsSummarization() bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.usagePercent >= tc.summarizeAtPercent
}

func (tc *TokenCounter) NeedsWarning() bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.usagePercent >= tc.warnAtPercent
}

func (tc *TokenCounter) Reset() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.currentTokens = 0
	tc.usagePercent = 0
}

func (tc *TokenCounter) GetStatusColor() string {
	_, _, percent := tc.GetUsage()
	
	switch {
	case percent >= 0.90:
		return "\033[91m"
	case percent >= 0.75:
		return "\033[93m"
	default:
		return "\033[92m"
	}
}

func (tc *TokenCounter) Render() string {
	current, max, percent := tc.GetUsage()
	color := tc.GetStatusColor()

	barWidth := 20
	filled := int(float64(barWidth) * percent)
	empty := barWidth - filled

	bar := color + "█" + "\033[92m" + strings.Repeat("█", max(filled-1, 0)) + "\033[0m" + "\033[2m" + strings.Repeat("░", empty) + "\033[0m"

	return fmt.Sprintf("%sTokens: %d/%d [%s] %.0f%%", color, current, max, bar, percent*100)
}

import "strings"

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
