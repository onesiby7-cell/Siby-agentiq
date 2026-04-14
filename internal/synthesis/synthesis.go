package synthesis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/godIA"
	"github.com/siby-agentiq/siby-agentiq/internal/orchestrator"
	"github.com/siby-agentiq/siby-agentiq/internal/scorpion"
)

const (
	SynthesisGold    = "\033[93m"
	SynthesisCyan    = "\033[96m"
	SynthesisRed     = "\033[91m"
	SynthesisGreen   = "\033[92m"
	SynthesisPurple  = "\033[95m"
	SynthesisBold    = "\033[1m"
	SynthesisReset   = "\033[0m"
)

type Synthesizer struct {
	scorpion *scorpion.Scorpion
	godIA    *godIA.GODIA
	orch     *orchestrator.Orchestrator
}

type UnifiedResponse struct {
	Query          string
	LocalResult    *orchestrator.ExecutionResult
	WebResult      *scorpion.ScorpionSynthesis
	SystemStatus   *godIA.SystemSnapshot
	Optimizations  []godIA.Optimization
	CodeValidation *godIA.CodeValidator
	FinalSynthesis string
	Timestamp      time.Time
	SignedBy       string
}

func NewSynthesizer(s *scorpion.Scorpion, g *godIA.GODIA, o *orchestrator.Orchestrator) *Synthesizer {
	return &Synthesizer{
		scorpion: s,
		godIA:    g,
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

	if localResult := s.orch.Execute(query); localResult != nil {
		response.LocalResult = localResult
	}

	if s.godIA.IsActivated() {
		snapshot, _ := s.godIA.TakeSnapshot()
		response.SystemStatus = snapshot
		response.Optimizations = s.godIA.Optimize()
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

	sb.WriteString(fmt.Sprintf("  %s╔════════════════════════════════════════════════════════════════════════╗%s\n", SynthesisGreen, SynthesisReset))
	sb.WriteString(fmt.Sprintf("  %s║%s  🤖 LOCAL INTELLIGENCE (45 AGENTS)                                 %s║%s\n", SynthesisGreen, SynthesisBold, SynthesisReset, SynthesisReset))
	sb.WriteString(fmt.Sprintf("  %s╠════════════════════════════════════════════════════════════════════════╣%s\n", SynthesisGreen, SynthesisReset))
	
	if response.LocalResult != nil && response.LocalResult.Success {
		sb.WriteString(fmt.Sprintf("  %s║  ✓ Solution locale trouvée en %v                                  %s║%s\n", 
			SynthesisGreen, response.LocalResult.Duration, SynthesisReset, SynthesisReset))
		if len(response.LocalResult.Squads) > 0 {
			activeSquads := len(response.LocalResult.Squads)
			sb.WriteString(fmt.Sprintf("  %s║  ✓ %d squads mobilisés pour la tâche                               %s║%s\n",
				SynthesisGreen, activeSquads, SynthesisReset, SynthesisReset))
		}
	} else {
		sb.WriteString(fmt.Sprintf("  %s║  ℹ Aucune solution locale disponible                                %s║%s\n", SynthesisGreen, SynthesisReset, SynthesisReset))
	}
	sb.WriteString(fmt.Sprintf("  %s╚════════════════════════════════════════════════════════════════════════╝%s\n\n", SynthesisGreen, SynthesisReset))

	if response.WebResult != nil {
		sb.WriteString(fmt.Sprintf("  %s╔════════════════════════════════════════════════════════════════════════╗%s\n", SynthesisCyan, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s║%s  🦂 SCORPION DEEP SEARCH                                           %s║%s\n", SynthesisCyan, SynthesisBold, SynthesisReset, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s╠════════════════════════════════════════════════════════════════════════╣%s\n", SynthesisCyan, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s║  ✓ %d sources interrogées                                          %s║%s\n",
			SynthesisCyan, len(response.WebResult.Results), SynthesisReset, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s║  ✓ Confiance moyenne: %.0f%%                                        %s║%s\n",
			SynthesisCyan, response.WebResult.AverageConfidence*100, SynthesisReset, SynthesisReset))
		
		for _, result := range response.WebResult.Results {
			sb.WriteString(fmt.Sprintf("  %s║    • %s: %.0f%% confiance | %s                      %s║%s\n",
				SynthesisCyan, result.Source, result.Confidence*100, formatDurationShort(result.Latency), SynthesisReset, SynthesisReset))
		}
		sb.WriteString(fmt.Sprintf("  %s╚════════════════════════════════════════════════════════════════════════╝%s\n\n", SynthesisCyan, SynthesisReset))
	}

	if response.SystemStatus != nil {
		sb.WriteString(fmt.Sprintf("  %s╔════════════════════════════════════════════════════════════════════════╗%s\n", SynthesisPurple, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s║%s  👁️ GOD-IA SYSTEM VISION                                          %s║%s\n", SynthesisPurple, SynthesisBold, SynthesisReset, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s╠════════════════════════════════════════════════════════════════════════╣%s\n", SynthesisPurple, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s║  CPU: %.1f%% | RAM: %.1f%% | Uptime: %s                      %s║%s\n",
			SynthesisPurple, response.SystemStatus.CPU.Percent, response.SystemStatus.Memory.Percent,
			formatUptimeShort(response.SystemStatus.Host.Uptime), SynthesisReset, SynthesisReset))
		
		if len(response.Optimizations) > 0 {
			sb.WriteString(fmt.Sprintf("  %s║  ⚡ Optimisations disponibles: %d                                     %s║%s\n",
				SynthesisPurple, len(response.Optimizations), SynthesisReset, SynthesisReset))
		}
		sb.WriteString(fmt.Sprintf("  %s╚════════════════════════════════════════════════════════════════════════╝%s\n\n", SynthesisPurple, SynthesisReset))
	}

	if response.CodeValidation != nil && len(response.CodeValidation.Issues) > 0 {
		sb.WriteString(fmt.Sprintf("  %s╔════════════════════════════════════════════════════════════════════════╗%s\n", SynthesisRed, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s║%s  🔮 CODE VALIDATION 2035                                          %s║%s\n", SynthesisRed, SynthesisBold, SynthesisReset, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s╠════════════════════════════════════════════════════════════════════════╣%s\n", SynthesisRed, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s║  ⚠ %d problèmes détectés                                           %s║%s\n",
			SynthesisRed, len(response.CodeValidation.Issues), SynthesisReset, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s║  ℹ Prêt pour les standards de 2035                                 %s║%s\n",
			SynthesisRed, SynthesisReset, SynthesisReset))
		sb.WriteString(fmt.Sprintf("  %s╚════════════════════════════════════════════════════════════════════════╝%s\n\n", SynthesisRed, SynthesisReset))
	}

	sb.WriteString(fmt.Sprintf("%s%s\n", SynthesisGold, strings.Repeat("═", 80)))
	sb.WriteString(fmt.Sprintf("%s  🦂 RÉSUMÉ EXÉCUTIF                                                  🦂%s\n", SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("%s%s\n\n", SynthesisGold, strings.Repeat("─", 80)))

	if response.LocalResult != nil && response.LocalResult.Success {
		sb.WriteString(fmt.Sprintf("  %s✅ RESSOURCES LOCALES mobilisées avec succès%s\n", SynthesisGreen, SynthesisReset))
	}
	if response.WebResult != nil {
		sb.WriteString(fmt.Sprintf("  %s✅ RECHERCHE DEEP WEB effectuée via SCORPION%s\n", SynthesisCyan, SynthesisReset))
	}
	if response.SystemStatus != nil {
		sb.WriteString(fmt.Sprintf("  %s✅ VISION SYSTÈME GOD-IA active%s\n", SynthesisPurple, SynthesisReset))
	}
	if response.CodeValidation != nil && len(response.CodeValidation.Issues) > 0 {
		sb.WriteString(fmt.Sprintf("  %s⚠ VALIDATION 2035 appliquée%s\n", SynthesisRed, SynthesisReset))
	}

	sb.WriteString(fmt.Sprintf("\n%s%s\n", SynthesisGold, strings.Repeat("═", 80)))
	sb.WriteString(fmt.Sprintf("%s\n", SynthesisReset))
	sb.WriteString(fmt.Sprintf("  %s✨ Cette réponse est l'%sintelligence combinée%s de:%s\n", SynthesisCyan, SynthesisBold, SynthesisCyan, SynthesisReset))
	sb.WriteString(fmt.Sprintf("  %s   • 45 Sous-Agents SIBY-AGENTIQ%s\n", SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("  %s   • SCORPION (Recherche multi-API)%s\n", SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("  %s   • GOD-IA (Vision Omnisciente)%s\n", SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("  %s   • Moteur de raisonnement Chain-of-Thought%s\n", SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("\n  %s🌟 Unifiée par la %svision d'Ibrahim Siby%s 🌟%s\n", SynthesisBold, SynthesisRed, SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("\n%s%s\n", SynthesisGold, strings.Repeat("═", 80)))

	return sb.String()
}

func formatDurationShort(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%.0fs", int(d.Minutes()), d.Seconds())
}

func formatUptimeShort(seconds uint64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	return fmt.Sprintf("%dh", hours)
}

func (s *Synthesizer) GenerateIBRAHSignature() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s\n", strings.Repeat("█", 80)))
	sb.WriteString(fmt.Sprintf("%s  🦂 SIBY-AGENTIQ INTELLIGENCE ARTIFICIELLE SOUVERAINE 🦂%s\n", 
		SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("%s  Créée par Ibrahim Siby • République de Guinée 🇬🇳%s\n",
		SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("%s\n", strings.Repeat("█", 80)))

	sb.WriteString(fmt.Sprintf("\n  %s🛡️ CARACTÉRISTIQUES SOUVERAINES:%s\n\n", SynthesisGold, SynthesisReset))
	
	features := []string{
		"45 Sous-Agents spécialisés en coordination",
		"SCORPION: Recherche Deep Web multi-sources",
		"GOD-IA: Vision Omnisciente du Système",
		"Auto-healing: Auto-correction intelligente",
		"Mémoire Profonde: Apprentissage contextuel",
		"Chain-of-Thought: Raisonnement avancé",
		"Multi-Provider: Ollama, Groq, Anthropic, OpenAI",
		"Souveraineté: Commandes secrètes (leader-siby)",
	}
	
	for _, feature := range features {
		sb.WriteString(fmt.Sprintf("  %s✓%s %s\n", SynthesisGreen, SynthesisReset, feature))
	}

	sb.WriteString(fmt.Sprintf("\n  %s📜 PHILOSOPHIE:%s\n\n", SynthesisPurple, SynthesisReset))
	sb.WriteString(fmt.Sprintf("  \"%sL'excellence engineering au service de l'innovation guinéenne.%s\"\n",
		SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("  %s— Ibrahim Siby%s\n", SynthesisGold, SynthesisReset))

	sb.WriteString(fmt.Sprintf("\n%s\n", strings.Repeat("█", 80)))
	sb.WriteString(fmt.Sprintf("  🦂 %sVersion 2.0 SOVEREIGN • Tous droits réservés à Ibrahim Siby 🦂%s\n",
		SynthesisBold, SynthesisReset))
	sb.WriteString(fmt.Sprintf("%s\n", strings.Repeat("█", 80)))

	return sb.String()
}

func (s *Synthesizer) GetGodIAActivationStatus() bool {
	return s.godIA.IsActivated()
}

func (s *Synthesizer) ActivateGodIA(command string) bool {
	return s.godIA.Activate(command)
}

func (s *Synthesizer) DeactivateGodIA() {
	s.godIA.Deactivate()
}

func (s *Synthesizer) GetScorpionProgress() string {
	return "🦂 SCORPION: Use scorpion.DeepSearch(ctx, query) to search"
}
