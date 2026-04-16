package scorpion

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

const (
	ScorpionYellow = "\033[93m"
	ScorpionBlack  = "\033[30m"
	ScorpionBG     = "\033[43m"
	ScorpionReset  = "\033[0m"
)

type Scorpion struct {
	providerManager *provider.ProviderManager
	results         map[string]*SearchResult
	mu              sync.RWMutex
	progressChan    chan float64
}

type SearchResult struct {
	Source     string
	Query      string
	Answer     string
	Links      []string
	Timestamp  time.Time
	Confidence float64
	Latency    time.Duration
}

type ScorpionConfig struct {
	MaxResults int
	Timeout    time.Duration
	Providers  []string
}

var defaultConfig = &ScorpionConfig{
	MaxResults: 5,
	Timeout:    30 * time.Second,
	Providers:  []string{"gemini", "gpt-4o", "perplexity"},
}

func NewScorpion(pm *provider.ProviderManager) *Scorpion {
	return &Scorpion{
		providerManager: pm,
		results:         make(map[string]*SearchResult),
		progressChan:    make(chan float64, 100),
	}
}

func (s *Scorpion) DeepSearch(ctx context.Context, query string) (*ScorpionSynthesis, error) {
	s.mu.Lock()
	s.results = make(map[string]*SearchResult)
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, defaultConfig.Timeout)
	defer cancel()

	go s.runProgressAnimation(ctx)

	var wg sync.WaitGroup
	resultChan := make(chan *SearchResult, len(defaultConfig.Providers))

	for _, prov := range defaultConfig.Providers {
		wg.Add(1)
		go func(provider string) {
			defer wg.Done()
			result := s.queryProvider(ctx, provider, query)
			if result != nil {
				resultChan <- result
			}
		}(prov)
	}

	go func() {
		wg.Wait()
		close(resultChan)
		close(s.progressChan)
	}()

	for result := range resultChan {
		s.mu.Lock()
		s.results[result.Source] = result
		s.mu.Unlock()
	}

	return s.synthesize(query), nil
}

func (s *Scorpion) queryProvider(ctx context.Context, provider, query string) *SearchResult {
	start := time.Now()

	result := &SearchResult{
		Source:    strings.ToUpper(provider),
		Query:     query,
		Timestamp: time.Now(),
	}

	switch provider {
	case "gemini":
		result.Answer = s.queryGemini(ctx, query)
	case "gpt-4o":
		result.Answer = s.queryGPT4(ctx, query)
	case "perplexity":
		result.Answer = s.queryPerplexity(ctx, query)
	default:
		result.Answer = "Provider not available"
	}

	result.Latency = time.Since(start)
	result.Confidence = s.calculateConfidence(result.Answer)

	return result
}

func (s *Scorpion) queryGemini(ctx context.Context, query string) string {
	if s.providerManager == nil {
		return s.fallbackSearch(query, "Gemini")
	}
	resp, err := s.providerManager.SmartChat(ctx, provider.SmartChatRequest{
		Messages: []provider.Message{{Role: "user", Content: fmt.Sprintf(
			"You are a research assistant. Provide a comprehensive answer to: %s. "+
				"Format: [GEMINI] Direct answer followed by relevant links in format: "+
				"Source: [url]", query)}},
	})
	if err != nil {
		return s.fallbackSearch(query, "Gemini")
	}
	if resp != nil {
		return resp.Message.Content
	}
	return s.fallbackSearch(query, "Gemini")
}

func (s *Scorpion) queryGPT4(ctx context.Context, query string) string {
	if s.providerManager == nil {
		return s.fallbackSearch(query, "GPT-4o")
	}
	resp, err := s.providerManager.SmartChat(ctx, provider.SmartChatRequest{
		Messages: []provider.Message{{Role: "user", Content: fmt.Sprintf(
			"You are a research assistant. Provide a comprehensive answer to: %s. "+
				"Format: [GPT-4o] Direct answer followed by relevant links in format: "+
				"Source: [url]", query)}},
	})
	if err != nil {
		return s.fallbackSearch(query, "GPT-4o")
	}
	if resp != nil {
		return resp.Message.Content
	}
	return s.fallbackSearch(query, "GPT-4o")
}

func (s *Scorpion) queryPerplexity(ctx context.Context, query string) string {
	if s.providerManager == nil {
		return s.fallbackSearch(query, "Perplexity")
	}
	resp, err := s.providerManager.SmartChat(ctx, provider.SmartChatRequest{
		Messages: []provider.Message{{Role: "user", Content: fmt.Sprintf(
			"You are a research assistant with real-time web access. Provide current information about: %s. "+
				"Format: [PERPLEXITY] Direct answer followed by relevant links in format: "+
				"Source: [url]", query)}},
	})
	if err != nil {
		return s.fallbackSearch(query, "Perplexity")
	}
	if resp != nil {
		return resp.Message.Content
	}
	return s.fallbackSearch(query, "Perplexity")
}

func (s *Scorpion) fallbackSearch(query, source string) string {
	links := s.extractLinks(query)
	linkStr := ""
	for _, link := range links {
		linkStr += fmt.Sprintf("Source: %s\n", link)
	}
	return fmt.Sprintf("[%s] Fallback search for: %s\n%s", source, query, linkStr)
}

func (s *Scorpion) extractLinks(query string) []string {
	linkPattern := regexp.MustCompile(`https?://[^\s]+`)
	return linkPattern.FindAllString(query, -1)
}

func (s *Scorpion) calculateConfidence(answer string) float64 {
	baseConfidence := 0.7
	if len(answer) > 100 {
		baseConfidence += 0.1
	}
	if strings.Contains(answer, "Source:") {
		baseConfidence += 0.1
	}
	if strings.Contains(answer, "https://") {
		baseConfidence += 0.1
	}
	return baseConfidence
}

func (s *Scorpion) synthesize(query string) *ScorpionSynthesis {
	s.mu.RLock()
	defer s.mu.RUnlock()

	synthesis := &ScorpionSynthesis{
		Query:       query,
		Results:     make([]*SearchResult, 0, len(s.results)),
		Timestamp:   time.Now(),
		FinalAnswer: "",
	}

	var totalConfidence float64
	for _, result := range s.results {
		synthesis.Results = append(synthesis.Results, result)
		totalConfidence += result.Confidence
	}

	if len(synthesis.Results) > 0 {
		synthesis.AverageConfidence = totalConfidence / float64(len(synthesis.Results))
	}

	synthesis.FinalAnswer = s.generateFinalAnswer(query, synthesis)

	return synthesis
}

func (s *Scorpion) generateFinalAnswer(query string, synthesis *ScorpionSynthesis) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s%s╔══════════════════════════════════════════════════════════════╗%s\n",
		ScorpionYellow, ScorpionBlack, ScorpionReset))
	sb.WriteString(fmt.Sprintf("%s%s║                    🦂 SCORPION DEEP SEARCH 🦂                ║%s\n",
		ScorpionYellow, ScorpionBlack, ScorpionReset))
	sb.WriteString(fmt.Sprintf("%s%s╚══════════════════════════════════════════════════════════════╝%s\n\n",
		ScorpionYellow, ScorpionBlack, ScorpionReset))

	sb.WriteString(fmt.Sprintf("  %s🔍 Requête:%s %s\n\n", ScorpionYellow, ScorpionReset, query))

	sb.WriteString(fmt.Sprintf("  %s📊 Résultats de %d sources:%s\n\n", ScorpionYellow, ScorpionReset, len(synthesis.Results)))

	for _, result := range synthesis.Results {
		sb.WriteString(fmt.Sprintf("  %s┌──────────────────────────────────────────────────────────────┐%s\n",
			ScorpionYellow, ScorpionReset))
		sb.WriteString(fmt.Sprintf("  %s│  %s%-10s %s │ Latence: %-10v | Confiance: %.0f%%%s │%s\n",
			ScorpionYellow, result.Source, formatDuration(result.Latency),
			fmt.Sprintf("%.0f%%", result.Confidence*100), ScorpionReset))
		sb.WriteString(fmt.Sprintf("  %s├──────────────────────────────────────────────────────────────┤%s\n",
			ScorpionYellow, ScorpionReset))

		lines := wrapText(result.Answer, 60)
		for _, line := range lines {
			sb.WriteString(fmt.Sprintf("  %s│  %s%s%s │%s\n", ScorpionYellow, ScorpionReset, padRight(line, 60), ScorpionYellow, ScorpionReset))
		}

		if len(result.Links) > 0 {
			sb.WriteString(fmt.Sprintf("  %s│  Liens: %s%s\n", ScorpionYellow, strings.Join(result.Links[:3], ", "), ScorpionReset))
		}
		sb.WriteString(fmt.Sprintf("  %s└──────────────────────────────────────────────────────────────┘%s\n\n",
			ScorpionYellow, ScorpionReset))
	}

	sb.WriteString(fmt.Sprintf("%s%s═══════════════════════════════════════════════════════════════%s\n",
		ScorpionYellow, ScorpionBlack, ScorpionReset))
	sb.WriteString(fmt.Sprintf("%s%s  🦂 Synthèse Finale - Powered by Ibrahim Siby 🦂%s\n",
		ScorpionYellow, ScorpionBlack, ScorpionReset))
	sb.WriteString(fmt.Sprintf("%s%s═══════════════════════════════════════════════════════════════%s\n",
		ScorpionYellow, ScorpionBlack, ScorpionReset))

	return sb.String()
}

type ScorpionSynthesis struct {
	Query             string
	Results           []*SearchResult
	Timestamp         time.Time
	FinalAnswer       string
	AverageConfidence float64
}

func (s *Scorpion) runProgressAnimation(ctx context.Context) {
	frames := []string{
		"█░░░░░░░░░░ 10%",
		"██░░░░░░░░░ 20%",
		"███░░░░░░░░ 30%",
		"████░░░░░░░ 40%",
		"█████░░░░░░ 50%",
		"██████░░░░░ 60%",
		"███████░░░░ 70%",
		"████████░░░ 80%",
		"█████████░░ 90%",
		"██████████ 100%",
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	idx := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if idx < len(frames) {
				fmt.Printf("\r  %s%s🦂 SCORPION SEARCH:%s %s%s%s",
					ScorpionYellow, ScorpionBlack, ScorpionReset,
					ScorpionYellow, frames[idx], ScorpionReset)
				idx++
			}
			if idx >= len(frames) {
				fmt.Printf("\r  %s%s🦂 SCORPION SEARCH:%s COMPLETE!          %s",
					ScorpionYellow, ScorpionBlack, ScorpionReset, ScorpionReset)
				return
			}
		}
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func wrapText(text string, width int) []string {
	var lines []string
	words := strings.Fields(text)
	currentLine := ""

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func (s *Scorpion) GetProgressChan() chan float64 {
	return s.progressChan
}

func (s *Scorpion) CompareResponses() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("\n🦂 COMPARISON ANALYSIS 🦂\n\n")

	if len(s.results) < 2 {
		return "Need at least 2 sources for comparison"
	}

	var bestProvider string
	var bestConfidence float64

	for source, result := range s.results {
		if result.Confidence > bestConfidence {
			bestConfidence = result.Confidence
			bestProvider = source
		}
	}

	sb.WriteString(fmt.Sprintf("  Winner: %s (%.0f%% confidence)\n\n", bestProvider, bestConfidence*100))

	return sb.String()
}
