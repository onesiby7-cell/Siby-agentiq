package synthesis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/orchestrator"
	"github.com/siby-agentiq/siby-agentiq/internal/scorpion"
)

const (
	SynthesisGold   = "\033[93m"
	SynthesisCyan   = "\033[96m"
	SynthesisRed    = "\033[91m"
	SynthesisGreen  = "\033[92m"
	SynthesisPurple = "\033[95m"
	SynthesisBold   = "\033[1m"
	SynthesisReset  = "\033[0m"
)

type Synthesizer struct {
	scorpion *scorpion.Scorpion
	orch     *orchestrator.Orchestrator
}

type UnifiedResponse struct {
	Query          string
	LocalResult    *orchestrator.ExecutionResult
	WebResult      *scorpion.ScorpionSynthesis
	FinalSynthesis string
	Timestamp      time.Time
	SignedBy       string
}

func NewSynthesizer(s *scorpion.Scorpion, o *orchestrator.Orchestrator) *Synthesizer {
	return &Synthesizer{
		scorpion: s,
		orch:     o,
	}
}

func (s *Synthesizer) ProcessQuery(ctx context.Context, query string) *UnifiedResponse {
	response := &UnifiedResponse{
		Query:     query,
		Timestamp: time.Now(),
		SignedBy:  "Ibrahim Siby",
	}

	hasLocalSolution := s.checkLocalKnowledge(query)
	if !hasLocalSolution {
		webResult, err := s.scorpion.DeepSearch(ctx, query)
		if err == nil {
			response.WebResult = webResult
		}
	}

	localResult, _ := s.orch.Execute(ctx, query)
	if localResult != nil {
		response.LocalResult = localResult
	}

	response.FinalSynthesis = s.generateFinalSynthesis(response)

	return response
}

func (s *Synthesizer) checkLocalKnowledge(query string) bool {
	keywords := []string{"file", "code", "function", "class", "struct", "project", "directory"}
	queryLower := strings.ToLower(query)

	count := 0
	for _, kw := range keywords {
		if strings.Contains(queryLower, kw) {
			count++
		}
	}

	return count >= 2
}

func (s *Synthesizer) generateFinalSynthesis(response *UnifiedResponse) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s%s\n", SynthesisGold, strings.Repeat("═", 80)))
	sb.WriteString(fmt.Sprintf("%s  🦂 SIBY-AGENTIQ UNIFIED INTELLIGENCE SYNTHESIS 🦂%s\n", SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("%s  ✨ Powered by Ibrahim Siby • Vision 2026+ ✨%s\n", SynthesisPurple, SynthesisReset))
	sb.WriteString(fmt.Sprintf("%s%s\n\n", SynthesisGold, strings.Repeat("═", 80)))

	sb.WriteString(fmt.Sprintf("  %s📋 Query:%s %s\n", SynthesisCyan, SynthesisReset, response.Query))
	sb.WriteString(fmt.Sprintf("  %s🕐 Time:%s %s\n\n", SynthesisCyan, SynthesisReset, response.Timestamp.Format("2006-01-02 15:04:05")))

	if response.LocalResult != nil && response.LocalResult.Success {
		sb.WriteString(fmt.Sprintf("  %s╔════════════════════════════════════════════════════════════╗%s\n", SynthesisGreen, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s║%s  🤖 LOCAL INTELLIGENCE (45 AGENTS)                          %s║%s\n", SynthesisGreen, SynthesisBold, SynthesisReset, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s║  ✓ Solution locale trouvee en %v                              %s║%s\n", SynthesisGreen, response.LocalResult.Duration, SynthesisReset, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s╚════════════════════════════════════════════════════════════╝%s\n\n", SynthesisGreen, SynthesisReset))
	}

	if response.WebResult != nil {
		sb.WriteString(fmt.Sprintf("  %s╔════════════════════════════════════════════════════════════╗%s\n", SynthesisCyan, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s║%s  🦂 SCORPION DEEP SEARCH                                      %s║%s\n", SynthesisCyan, SynthesisBold, SynthesisReset, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s║  ✓ %d sources interrogees                                     %s║%s\n", SynthesisCyan, len(response.WebResult.Results), SynthesisReset, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s╚════════════════════════════════════════════════════════════╝%s\n\n", SynthesisCyan, SynthesisReset))
	}

	sb.WriteString(fmt.Sprintf("%s%s\n", SynthesisGold, strings.Repeat("═", 80)))
	sb.WriteString(fmt.Sprintf("%s  🦂 RÉSUMÉ EXÉCUTIF                                             🦂%s\n", SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("%s%s\n\n", SynthesisGold, strings.Repeat("─", 80)))

	if response.LocalResult != nil && response.LocalResult.Success {
		sb.WriteString(fmt.Sprintf("  %s✅ RESSOURCES LOCALES mobilisees avec succes%s\n", SynthesisGreen, SynthesisReset))
	}
	if response.WebResult != nil {
		sb.WriteString(fmt.Sprintf("  %s✅ RECHERCHE DEEP WEB effectuee via SCORPION%s\n", SynthesisCyan, SynthesisReset))
	}

	sb.WriteString(fmt.Sprintf("\n  %s🌟 Unifiee par la vision d'Ibrahim Siby 🌟%s\n", SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("\n%s%s\n", SynthesisGold, strings.Repeat("═", 80)))

	return sb.String()
}

func (s *Synthesizer) GenerateSignature() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s\n", strings.Repeat("█", 80)))
	sb.WriteString(fmt.Sprintf("%s  🦂 SIBY-AGENTIQ INTELLIGENCE ARTIFICIELLE SOUVERAINE 🦂%s\n", SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("%s  Creee par Ibrahim Siby • Republique de Guinee 🇬🇳%s\n", SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("%s\n", strings.Repeat("█", 80)))

	sb.WriteString(fmt.Sprintf("\n  %s🛡️ CARACTERISTIQUES SOUVERAINES:%s\n\n", SynthesisGold, SynthesisReset))

	features := []string{
		"45 Sous-Agents specialises en coordination",
		"SCORPION: Recherche Deep Web multi-sources",
		"GOD-IA: Vision Omnisciente du Systeme",
		"Auto-healing: Auto-correction intelligente",
		"Memoire Profonde: Apprentissage contextuel",
		"Chain-of-Thought: Raisonnement avance",
		"Multi-Provider: Ollama, Groq, Anthropic, OpenAI",
		"Souverainete: Commandes secretes (leader-siby)",
	}

	for _, feature := range features {
		sb.WriteString(fmt.Sprintf("  %s✓%s %s\n", SynthesisGreen, SynthesisReset, feature))
	}

	sb.WriteString(fmt.Sprintf("\n  %s📜 PHILOSOPHIE:%s\n\n", SynthesisPurple, SynthesisReset))
	sb.WriteString(fmt.Sprintf("  \"%sL'excellence engineering au service de l'innovation guineenne.%s\"\n", SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("  %s— Ibrahim Siby%s\n", SynthesisGold, SynthesisReset))

	sb.WriteString(fmt.Sprintf("\n%s\n", strings.Repeat("█", 80)))
	sb.WriteString(fmt.Sprintf("  🦂 %sVersion 2.0 SOVEREIGN • Tous droits reserves a Ibrahim Siby 🦂%s\n", SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("%s\n", strings.Repeat("█", 80)))

	return sb.String()
}
