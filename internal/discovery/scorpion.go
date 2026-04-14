package discovery

import (
	"fmt"
	"os"
	"strings"
)

const (
	ScorpionYellow = "\033[93m"
	ScorpionBlack  = "\033[30m"
	ScorpionBG     = "\033[43m"
	ScorpionReset  = "\033[0m"
	ScorpionGreen  = "\033[92m"
	ScorpionCyan   = "\033[96m"
)

type ScorpionCommand struct {
	Provider string
	Action   string
}

func NewScorpionCommand(args []string) *ScorpionCommand {
	if len(args) < 2 {
		return &ScorpionCommand{Action: "help"}
	}

	return &ScorpionCommand{
		Provider: args[0],
		Action:   args[1],
	}
}

func (s *ScorpionCommand) Execute() error {
	switch s.Action {
	case "fetch", "get", "create":
		return s.FetchKey()
	case "check":
		return s.CheckStatus()
	case "help":
		return s.ShowHelp()
	default:
		return s.FetchKey()
	}
}

func (s *ScorpionCommand) FetchKey() error {
	url := ScorpionFetchKey(s.Provider)
	if url == "" {
		return fmt.Errorf("provider non supporté: %s", s.Provider)
	}

	fmt.Printf(`
%[1$s🦂 SCORPION KEY FETCHER 🦂%[1$s

%[2$sOuverture de la page:%s%[1$s
%[3$s%[4$s%[1$s

%[2$sInstructions:%[1$s
1. Crée un compte gratuit sur le site
2. Génère une nouvelle clé API
3. Copie la clé (commence par 'gsk_' pour Groq)
4. Configure dans ton terminal:

   %[5$sexport GROQ_API_KEY=gsk_xxx...%[1$s

%[2$sAprès configuration, relance Siby-Agentiq!%[1$s

`,
		ScorpionReset, ScorpionCyan, ScorpionYellow, url, ScorpionGreen)

	return OpenSignupPage(s.Provider)
}

func (s *ScorpionCommand) CheckStatus() error {
	result := ScanWithScorpion()

	fmt.Printf(`
%[1$s🦂 SCORPION STATUS 🦂%[1$s

%[2$sProviders disponibles:%[1$s
`, ScorpionReset, ScorpionCyan)

	for _, p := range result.AvailableProviders {
		fmt.Printf("  %[1$s✓%[1$s %s\n", ScorpionGreen, p)
	}

	fmt.Printf(`
%[2$sProviders manquants:%[1$s
`, ScorpionCyan)

	for _, p := range result.MissingProviders {
		fmt.Printf("  %[1$s✗%[1$s %s\n", ScorpionYellow, p)
	}

	if !result.HasProvider {
		fmt.Printf(`
%[1$s╔══════════════════════════════════════════════════════════╗
║                                                          ║
║  %[2$s⚠ Aucune clé API trouvée!%[1$s                            ║
║                                                          ║
║  %[3$sLancez 'siby scorpion groq fetch'%[1$s pour ouvrir     ║
║  %[3$sla page de création de clé Groq (gratuit).%[1$s        ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝%[1$s
`, ScorpionReset, ScorpionYellow, ScorpionCyan)
	}

	return nil
}

func (s *ScorpionCommand) ShowHelp() error {
	fmt.Printf(`
%[1$s🦂 SCORPION - Aide 🦂%[1$s

%[2$sCommandes disponibles:%[1$s

  %[3$ssiby scorpion groq fetch%[1$s
      Ouvre la page Groq pour créer une clé API gratuite

  %[3$ssiby scorpion openai fetch%[1$s
      Ouvre la page OpenAI pour créer une clé API

  %[3$ssiby scorpion anthropic fetch%[1$s
      Ouvre la page Anthropic pour créer une clé API

  %[3$ssiby scorpion check%[1$s
      Vérifie l'état des providers configurés

%[2$sConfiguration manuelle:%[1$s

  %[4$sexport GROQ_API_KEY=gsk_xxx...%[1$s
  %[4$sexport OPENAI_API_KEY=sk-xxx...%[1$s
  %[4$sexport ANTHROPIC_API_KEY=sk-ant-xxx...%[1$s

%[2$sFichiers de configuration:%[1$s

  %[4$s~/.siby/config.json%[1$s

`, ScorpionReset, ScorpionCyan, ScorpionGreen, ScorpionYellow)

	return nil
}

func ScorpionBanner() string {
	return fmt.Sprintf(`
%[1$s%[3$s
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║  %[2$s🦂🦂🦂 SCORPION - Deep Intelligence Fetcher 🦂🦂🦂%[2$s       ║
║                                                           ║
║  %[4$sLe module qui trouve et configure les clés API%[4$s        ║
║  %[4$spour que Siby-Agentiq fonctionne immédiatement.%[4$s       ║
║                                                           ║
║  %[5$s✓ Local-First%[5$s     → Ollama priorité              ║
║  %[5$s✓ Auto-Discovery%[5$s  → Scan silencieux au démarrage  ║
║  %[5$s✓ Key Fetcher%[5$s     → Ouverture page API gratuite  ║
║  %[5$s✓ Zero Config%[5$s     → Fonctionne out-of-the-box   ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝%[1$s
`,
		ScorpionReset, ScorpionYellow, ScorpionBlack,
		ScorpionCyan, ScorpionGreen)
}

func ParseScorpionCommand(input string) bool {
	parts := strings.Fields(strings.ToLower(input))
	if len(parts) < 2 || parts[0] != "scorpion" {
		return false
	}

	cmd := NewScorpionCommand(parts[1:])
	cmd.Execute()
	return true
}

func SetEnvKey(provider, key string) error {
	envVar := ""
	switch provider {
	case "groq":
		envVar = "GROQ_API_KEY"
	case "openai":
		envVar = "OPENAI_API_KEY"
	case "anthropic":
		envVar = "ANTHROPIC_API_KEY"
	default:
		return fmt.Errorf("provider non supporté: %s", provider)
	}

	return os.Setenv(envVar, key)
}

func GetEnvKey(provider string) string {
	envVar := ""
	switch provider {
	case "groq":
		envVar = "GROQ_API_KEY"
	case "openai":
		envVar = "OPENAI_API_KEY"
	case "anthropic":
		envVar = "ANTHROPIC_API_KEY"
	}
	return os.Getenv(envVar)
}

func HasAPIKey(provider string) bool {
	return GetEnvKey(provider) != ""
}

func GetAllConfiguredProviders() []string {
	providers := []string{}
	for _, p := range []string{"ollama", "groq", "openai", "anthropic"} {
		if p == "ollama" {
			if _, config := scanOllama(); config != "" {
				providers = append(providers, p)
			}
		} else if HasAPIKey(p) {
			providers = append(providers, p)
		}
	}
	return providers
}
