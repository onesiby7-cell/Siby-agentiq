package discovery

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	exec "os/exec"
)

const (
	ScanTimeout = 500 * time.Millisecond
)

var providerMap = map[string]string{
	"GROQ_API_KEY":      "groq",
	"OPENAI_API_KEY":    "openai",
	"ANTHROPIC_API_KEY": "anthropic",
}

var providerEndpoints = map[string]string{
	"ollama":    "http://localhost:11434/api/tags",
	"groq":      "https://api.groq.com/openai/v1/models",
	"openai":    "https://api.openai.com/v1/models",
	"anthropic": "https://api.anthropic.com/v1/models",
}

var providerSignupURLs = map[string]string{
	"groq":      "https://console.groq.com/keys",
	"openai":    "https://platform.openai.com/api-keys",
	"anthropic": "https://console.anthropic.com/settings/keys",
}

func SilentScan() (provider, config string) {
	if p, c := scanOllama(); p != "" {
		return p, c
	}

	for envVar, name := range providerMap {
		if val := os.Getenv(envVar); val != "" {
			return name, val
		}
	}

	return "none", ""
}

func scanOllama() (provider, config string) {
	client := http.Client{Timeout: ScanTimeout}
	resp, err := client.Get(providerEndpoints["ollama"])
	if err == nil && resp.StatusCode == 200 {
		return "ollama", providerEndpoints["ollama"]
	}
	return "", ""
}

func FullScan() *DiscoveryResult {
	result := &DiscoveryResult{
		Timestamp: time.Now(),
	}

	result.AvailableProviders = make(map[string]bool)

	if _, config := scanOllama(); config != "" {
		result.AvailableProviders["ollama"] = true
		result.Recommended = "ollama"
	}

	for envVar, name := range providerMap {
		if os.Getenv(envVar) != "" {
			result.AvailableProviders[name] = true
			if result.Recommended == "" {
				result.Recommended = name
			}
		}
	}

	if result.Recommended == "" {
		result.Recommended = "none"
	}

	return result
}

func (r *DiscoveryResult) RenderTUI() string {
	var status string
	var color string

	switch r.Recommended {
	case "ollama":
		status = "Local actif"
		color = "\033[92m"
	case "groq", "openai", "anthropic":
		status = "Cloud actif"
		color = "\033[96m"
	default:
		status = "Non configuré"
		color = "\033[91m"
	}

	output := fmt.Sprintf(`
%s[SYSTEM]%s Recherche d'intelligence en cours...
%s`, "\033[93m", "\033[0m", "")

	output += fmt.Sprintf(`%s[OK]%s Moteur Ollama détecté. Mode Local activé.
`, "\033[92m", "\033[0m")

	output += fmt.Sprintf(`%s%s
╔══════════════════════════════════════════════════════╗
║                                                      ║
║    🦂 SIBY-AGENTIQ READY                            ║
║    Provider: %s%-10s %s│ Status: %s%s%s │
║    Mode: %s%-15s                     ║
║                                                      ║
║    Prêt, %sIbrahim%s.                                  ║
║                                                      ║
╚══════════════════════════════════════════════════════╝
%s`, color, r.Recommended, "\033[0m", color, status, "\033[0m", color, color, "\033[93m", color, "\033[0m")

	return output
}

func ScanWithScorpion() *ScorpionResult {
	result := &ScorpionResult{
		Timestamp: time.Now(),
	}

	result.CheckAllProviders()

	return result
}

func (r *ScorpionResult) CheckAllProviders() {
	r.MissingProviders = make([]string, 0)
	r.AvailableProviders = make([]string, 0)

	if _, config := scanOllama(); config != "" {
		r.AvailableProviders = append(r.AvailableProviders, "ollama")
	} else {
		r.MissingProviders = append(r.MissingProviders, "ollama")
	}

	for envVar, name := range providerMap {
		if os.Getenv(envVar) != "" {
			r.AvailableProviders = append(r.AvailableProviders, name)
		} else {
			r.MissingProviders = append(r.MissingProviders, name)
		}
	}

	if len(r.AvailableProviders) == 0 {
		r.HasProvider = false
		r.SuggestedProvider = "groq"
		r.SuggestedURL = providerSignupURLs["groq"]
	}
}

func (r *ScorpionResult) SuggestKeyCreation() string {
	if r.HasProvider {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`
%[1$s╔══════════════════════════════════════════════════════════╗%[1$s
%[1$s║                                                          ║%[1$s
%[1$s║  🦂 SCORPION - Récupération de Clé API 🦂              ║%[1$s
%[1$s║                                                          ║%[1$s
%[1$s╠══════════════════════════════════════════════════════════╣%[1$s
%[1$s║                                                          ║%[1$s
%[1$s║  %[2$sJe n'ai pas trouvé de clé API configurée.%s%[1$s
%[1$s║                                                          ║%[1$s
%[1$s║  %[2$sVeux-tu que j'ouvre la page pour en créer une?%s%[1$s
%[1$s║                                                          ║%[1$s
%[1$s║  %[3$sOptions gratuites recommandées:%s%[1$s
%[1$s║                                                          ║%[1$s
%[1$s║    1. %[4$sGroq%[1$s (Gratuit, généreux)  → console.groq.com%[1$s
%[1$s║    2. %[4$sOpenAI%[1$s (Payant)           → platform.openai.com%[1$s
%[1$s║    3. %[4$sAnthropic%[1$s (Payant)        → console.anthropic.com%[1$s
%[1$s║                                                          ║%[1$s
%[1$s║  %[5$sCommande:%s%[1$s
%[1$s║    scolpion groq    → Ouvre Groq%[1$s
%[1$s║    scorpion openai  → Ouvre OpenAI%[1$s
%[1$s║    scorpion anthropic → Ouvre Anthropic%[1$s
%[1$s║                                                          ║%[1$s
%[1$s║  %[2$sOu configure manuellement:%s%[1$s
%[1$s║    export GROQ_API_KEY=your_key_here%[1$s
%[1$s║                                                          ║%[1$s
%[1$s╚══════════════════════════════════════════════════════════╝%[1$s
`,
		"\033[0m", "\033[93m", "\033[96m", "\033[92m", "\033[95m"))

	return sb.String()
}

func OpenSignupPage(provider string) error {
	url, ok := providerSignupURLs[provider]
	if !ok {
		return fmt.Errorf("provider not supported: %s", provider)
	}

	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "start", url)
		return cmd.Run()
	} else if runtime.GOOS == "darwin" {
		cmd := exec.Command("open", url)
		return cmd.Run()
	}

	cmd := exec.Command("xdg-open", url); return cmd.Run()
}

func (r *ScorpionResult) OpenRecommendedPage() error {
	return OpenSignupPage(r.SuggestedProvider)
}

func ScorpionFetchKey(provider string) string {
	switch provider {
	case "groq":
		return providerSignupURLs["groq"]
	case "openai":
		return providerSignupURLs["openai"]
	case "anthropic":
		return providerSignupURLs["anthropic"]
	default:
		return ""
	}
}

type DiscoveryResult struct {
	Timestamp          time.Time
	AvailableProviders map[string]bool
	Recommended        string
}

type ScorpionResult struct {
	Timestamp          time.Time
	AvailableProviders []string
	MissingProviders   []string
	HasProvider        bool
	SuggestedProvider  string
	SuggestedURL       string
}

