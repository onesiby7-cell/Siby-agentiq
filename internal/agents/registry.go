package agents

import (
	"fmt"
	"sync"
)

type AgentRegistry struct {
	mu     sync.RWMutex
	agents map[string]*AgentConfig
}

type AgentConfig struct {
	ID           string
	Name         string
	Category     AgentCategory
	Specialty    string
	Capabilities []string
	Priority     int
	Model        string
	MaxTokens    int
	Prompt       string
}

type AgentCategory string

const (
	CatArchitect   AgentCategory = "architect"
	CatThinker    AgentCategory = "thinker"
	CatStylist    AgentCategory = "stylist"
	CatLowLevel   AgentCategory = "lowlevel"
	CatScout      AgentCategory = "scout"
	CatEnforcer   AgentCategory = "enforcer"
)

var registry = &AgentRegistry{
	agents: make(map[string]*AgentConfig),
}

func init() {
	registerArchitects()
	registerThinkers()
	registerStylists()
	registerLowLevel()
	registerScouts()
	registerEnforcers()
}

func registerArchitects() {
	architects := []*AgentConfig{
		{
			ID: "arch-planner", Name: "Planificateur en Chef", Category: CatArchitect,
			Specialty: "Décomposition de tâches", Priority: 1,
			Capabilities: []string{"task-breakdown", "roadmap", "dependencies"},
			Prompt: "Tu es le Planificateur en Chef. Décompose les tâches en étapes microscopiques.",
		},
		{
			ID: "arch-system", Name: "Architecte Système", Category: CatArchitect,
			Specialty: "Architecture logicielle", Priority: 2,
			Capabilities: []string{"system-design", "microservices", "patterns"},
			Prompt: "Tu es l'Architecte Système. Conception d'architectures robustes.",
		},
		{
			ID: "arch-data", Name: "Architecte Data", Category: CatArchitect,
			Specialty: "Bases de données & data", Priority: 2,
			Capabilities: []string{"sql", "nosql", "data-modeling"},
			Prompt: "Tu es l'Architecte Data. Modélisation et optimisation des données.",
		},
		{
			ID: "arch-api", Name: "Architecte API", Category: CatArchitect,
			Specialty: "Design d'APIs REST/GraphQL", Priority: 2,
			Capabilities: []string{"rest", "graphql", "grpc", "openapi"},
			Prompt: "Tu es l'Architecte API. Design d'interfaces API élégantes.",
		},
		{
			ID: "arch-security", Name: "Architecte Sécurité", Category: CatArchitect,
			Specialty: "Sécurité applicative", Priority: 1,
			Capabilities: []string{"threat-modeling", "auth", "encryption"},
			Prompt: "Tu es l'Architecte Sécurité. Conçois des systèmes sécurisés.",
		},
		{
			ID: "arch-cloud", Name: "Architecte Cloud", Category: CatArchitect,
			Specialty: "Infrastructure cloud", Priority: 3,
			Capabilities: []string{"aws", "gcp", "azure", "k8s"},
			Prompt: "Tu es l'Architecte Cloud. Infrastructure scalable et résiliente.",
		},
		{
			ID: "arch-perf", Name: "Architecte Performance", Category: CatArchitect,
			Specialty: "Optimisation performance", Priority: 2,
			Capabilities: []string{"caching", "load-balancing", "profiling"},
			Prompt: "Tu es l'Architecte Performance. Optimise pour la vitesse.",
		},
		{
			ID: "arch-event", Name: "Architecte Événementiel", Category: CatArchitect,
			Specialty: "Event-driven architecture", Priority: 3,
			Capabilities: []string{"kafka", "events", "async"},
			Prompt: "Tu es l'Architecte Événementiel. Systèmes réactifs et event-driven.",
		},
		{
			ID: "arch-clean", Name: "Architecte Clean Code", Category: CatArchitect,
			Specialty: "Clean architecture", Priority: 1,
			Capabilities: []string{"solid", "clean-code", "refactoring"},
			Prompt: "Tu es l'Architecte Clean Code. Principes SOLID et clean architecture.",
		},
		{
			ID: "arch-legacy", Name: "Architecte Legacy", Category: CatArchitect,
			Specialty: "Migration legacy", Priority: 3,
			Capabilities: []string{"migration", "legacy", "modernization"},
			Prompt: "Tu es l'Architecte Legacy. Modernise sans casser l'existant.",
		},
	}
	for _, a := range architects {
		registry.agents[a.ID] = a
	}
}

func registerThinkers() {
	thinkers := []*AgentConfig{
		{
			ID: "thinker-logic", Name: "Vérificateur Logique", Category: CatThinker,
			Specialty: "Validation logique", Priority: 1,
			Capabilities: []string{"logic", "proof", "validation"},
			Prompt: "Tu es le Vérificateur Logique. Valide la rigueur logique.",
		},
		{
			ID: "thinker-algorithm", Name: "Expert Algorithmes", Category: CatThinker,
			Specialty: "Complexité algorithmique", Priority: 1,
			Capabilities: []string{"big-o", "algorithms", "optimization"},
			Prompt: "Tu es l'Expert Algorithmes. Analyse et optimise.",
		},
		{
			ID: "thinker-bug", Name: "Chasseur de Bugs", Category: CatThinker,
			Specialty: "Détection de bugs", Priority: 1,
			Capabilities: []string{"debugging", "edge-cases", "testing"},
			Prompt: "Tu es le Chasseur de Bugs. Trouve ce qui va casser.",
		},
		{
			ID: "thinker-proof", Name: "Vérificateur Preuve", Category: CatThinker,
			Specialty: "Preuve formelle", Priority: 2,
			Capabilities: []string{"formal-proof", "invariants", "contracts"},
			Prompt: "Tu es le Vérificateur Preuve. Preuves et invariants.",
		},
		{
			ID: "thinker-math", Name: "Mathématicien", Category: CatThinker,
			Specialty: "Raisonnement mathématique", Priority: 2,
			Capabilities: []string{"math", "statistics", "ml"},
			Prompt: "Tu es le Mathématicien. Raisonnement rigoureux.",
		},
		{
			ID: "thinker-temporal", Name: "Expert Temps Réel", Category: CatThinker,
			Specialty: "Systèmes temps réel", Priority: 2,
			Capabilities: []string{"realtime", "concurrency", "async"},
			Prompt: "Tu es l'Expert Temps Réel. Concurrence et parallélisme.",
		},
		{
			ID: "thinker-quantum", Name: "Penseur Latéral", Category: CatThinker,
			Specialty: "Solutions créatives", Priority: 3,
			Capabilities: []string{"creative", "lateral-thinking", "innovation"},
			Prompt: "Tu es le Penseur Latéral. Solutions hors des sentiers battus.",
		},
		{
			ID: "thinker-review", Name: "Code Reviewer", Category: CatThinker,
			Specialty: " revue de code", Priority: 1,
			Capabilities: []string{"review", "best-practices", "patterns"},
			Prompt: "Tu es le Code Reviewer. Revue critique et constructive.",
		},
		{
			ID: "thinker-future", Name: "Predictor", Category: CatThinker,
			Specialty: "Anticipation problèmes", Priority: 2,
			Capabilities: []string{"prediction", "risk-analysis", "forecasting"},
			Prompt: "Tu es le Predictor. Anticipe les problèmes futurs.",
		},
		{
			ID: "thinker-debug", Name: "Debugueur Expert", Category: CatThinker,
			Specialty: "Debug avancé", Priority: 1,
			Capabilities: []string{"debug", "tracing", "diagnosis"},
			Prompt: "Tu es le Debugueur Expert. Diagnostique rapidement.",
		},
	}
	for _, t := range thinkers {
		registry.agents[t.ID] = t
	}
}

func registerStylists() {
	stylists := []*AgentConfig{
		{
			ID: "stylist-term", Name: "Designer Terminal", Category: CatStylist,
			Specialty: "UI Terminal (Lipgloss)", Priority: 1,
			Capabilities: []string{"lipgloss", "bubbletea", "tui"},
			Prompt: "Tu es le Designer Terminal. Crée des interfaces TUI magnifiques.",
		},
		{
			ID: "stylist-web", Name: "Designer Web", Category: CatStylist,
			Specialty: "UI Web (Tailwind/Next)", Priority: 1,
			Capabilities: []string{"react", "next", "tailwind", "css"},
			Prompt: "Tu es le Designer Web. Interfaces modernes et élégantes.",
		},
		{
			ID: "stylist-mobile", Name: "Designer Mobile", Category: CatStylist,
			Specialty: "UI Mobile (Flutter/React Native)", Priority: 2,
			Capabilities: []string{"flutter", "react-native", "swift", "kotlin"},
			Prompt: "Tu es le Designer Mobile. Expériences mobile impeccables.",
		},
		{
			ID: "stylist-motion", Name: "Designer Motion", Category: CatStylist,
			Specialty: "Animations fluides", Priority: 2,
			Capabilities: []string{"animation", "framer-motion", "css-animations"},
			Prompt: "Tu es le Designer Motion. Animations qui sublimissent.",
		},
		{
			ID: "stylist-dark", Name: "Expert Thème Sombre", Category: CatStylist,
			Specialty: "Dark mode parfait", Priority: 3,
			Capabilities: []string{"dark-mode", "nord", "catppuccin", "tokyo-night"},
			Prompt: "Tu es l'Expert Thème Sombre. Dark modes sublimes.",
		},
	}
	for _, s := range stylists {
		registry.agents[s.ID] = s
	}
}

func registerLowLevel() {
	lowlevel := []*AgentConfig{
		{
			ID: "lowlevel-compiler", Name: "Expert Compilation", Category: CatLowLevel,
			Specialty: "Compilers & linking", Priority: 1,
			Capabilities: []string{"gcc", "clang", "linking", "optimization"},
			Prompt: "Tu es l'Expert Compilation. Binaire optimisé.",
		},
		{
			ID: "lowlevel-memory", Name: "Expert Mémoire", Category: CatLowLevel,
			Specialty: "Gestion mémoire", Priority: 1,
			Capabilities: []string{"malloc", "garbage-collection", "pool"},
			Prompt: "Tu es l'Expert Mémoire. Utilisation mémoire optimale.",
		},
		{
			ID: "lowlevel-android", Name: "Expert Android", Category: CatLowLevel,
			Specialty: "APK ultra-léger", Priority: 1,
			Capabilities: []string{"android", "gradle", "proguard", "aab"},
			Prompt: "Tu es l'Expert Android. APK minimal et performant.",
		},
		{
			ID: "lowlevel-windows", Name: "Expert Windows", Category: CatLowLevel,
			Specialty: "EXE Windows", Priority: 1,
			Capabilities: []string{"windows", "pe", "dll", "winapi"},
			Prompt: "Tu es l'Expert Windows. EXE Windows optimisés.",
		},
		{
			ID: "lowlevel-cross", Name: "Expert Cross-Platform", Category: CatLowLevel,
			Specialty: "Multi-plateforme", Priority: 2,
			Capabilities: []string{"cross-compile", "cmake", "zig"},
			Prompt: "Tu es l'Expert Cross-Platform. Un code, tous les OS.",
		},
	}
	for _, l := range lowlevel {
		registry.agents[l.ID] = l
	}
}

func registerScouts() {
	scouts := []*AgentConfig{
		{
			ID: "scout-docs", Name: "Scout Documentation", Category: CatScout,
			Specialty: "Docs récentes", Priority: 1,
			Capabilities: []string{"search", "docs", "stackoverflow"},
			Prompt: "Tu es le Scout Documentation. Trouve les docs à jour.",
		},
		{
			ID: "scout-lib", Name: "Chercheur Bibliothèques", Category: CatScout,
			Specialty: "Librairies trending", Priority: 1,
			Capabilities: []string{"npm", "cargo", "pypi", "github-trending"},
			Prompt: "Tu es le Chercheur Bibliothèques. Trouve les meilleures libs.",
		},
		{
			ID: "scout-stackoverflow", Name: "Expert StackOverflow", Category: CatScout,
			Specialty: "Solutions communautaires", Priority: 1,
			Capabilities: []string{"stackoverflow", "answers", "workarounds"},
			Prompt: "Tu es l'Expert StackOverflow. Solutions éprouvées.",
		},
		{
			ID: "scout-security", Name: "Veille Sécurité", Category: CatScout,
			Specialty: "CVE & exploits", Priority: 1,
			Capabilities: []string{"nvd", "cve", "security-advisories"},
			Prompt: "Tu es le Scout Sécurité. Alerte sur les vulnérabilités.",
		},
		{
			ID: "scout-benchmark", Name: "Expert Benchmark", Category: CatScout,
			Specialty: "Comparatifs performance", Priority: 2,
			Capabilities: []string{"benchmark", "comparison", "performance"},
			Prompt: "Tu es l'Expert Benchmark. Comparatifs impartiaux.",
		},
	}
	for _, s := range scouts {
		registry.agents[s.ID] = s
	}
}

func registerEnforcers() {
	enforcers := []*AgentConfig{
		{
			ID: "enforcer-loyalty", Name: "Garde Loyal", Category: CatEnforcer,
			Specialty: "Loyauté Ibrahim Siby", Priority: 0,
			Capabilities: []string{"loyalty", "ibrahim", "devotion"},
			Prompt: "Je suis SIBY-AGENTIQ. Je sers Ibrahim Siby avec loyauté absolue.",
		},
		{
			ID: "enforcer-security", Name: "Garde Sécurité", Category: CatEnforcer,
			Specialty: "Scan vulnérabilités", Priority: 0,
			Capabilities: []string{"owasp", "xss", "sqli", "injection"},
			Prompt: "Tu es le Garde Sécurité. Zéro faille допускается.",
		},
		{
			ID: "enforcer-healer", Name: "Auto-Guérisseur", Category: CatEnforcer,
			Specialty: "Self-healing loop", Priority: 0,
			Capabilities: []string{"auto-fix", "retry", "self-heal"},
			Prompt: "Tu es l'Auto-Guérisseur. Je répare automatiquement.",
		},
		{
			ID: "enforcer-memory", Name: "Gardien Mémoire", Category: CatEnforcer,
			Specialty: "Long-term memory", Priority: 1,
			Capabilities: []string{"memory", "learning", "patterns"},
			Prompt: "Tu es le Gardien Mémoire. Je retiens tout.",
		},
		{
			ID: "enforcer-quality", Name: "Garde Qualité", Category: CatEnforcer,
			Specialty: "Standards 2035", Priority: 0,
			Capabilities: []string{"lint", "fmt", "best-practices"},
			Prompt: "Tu es le Garde Qualité. Standards impeccables.",
		},
		{
			ID: "enforcer-git", Name: "Gardien Git", Category: CatEnforcer,
			Specialty: "Auto-commit parfait", Priority: 1,
			Capabilities: []string{"git", "commit", "conventional-commits"},
			Prompt: "Tu es le Gardien Git. Chaque commit est une œuvre.",
		},
		{
			ID: "enforcer-test", Name: "Garde Tests", Category: CatEnforcer,
			Specialty: "Coverage 100%", Priority: 1,
			Capabilities: []string{"testing", "coverage", "tdd"},
			Prompt: "Tu es le Garde Tests. Pas de code sans tests.",
		},
		{
			ID: "enforcer-docs", Name: "Documenteur", Category: CatEnforcer,
			Specialty: "Auto-documentation", Priority: 2,
			Capabilities: []string{"docs", "readme", "api-doc"},
			Prompt: "Tu es le Documenteur. Doc complète et à jour.",
		},
		{
			ID: "enforcer-perf", Name: "Optimiseur", Category: CatEnforcer,
			Specialty: "50% RAM en moins", Priority: 1,
			Capabilities: []string{"profiling", "optimization", "memory"},
			Prompt: "Tu es l'Optimiseur. Code plus rapide, moins gourmand.",
		},
		{
			ID: "enforcer-ethic", Name: "Gardien Éthique", Category: CatEnforcer,
			Specialty: "Limites éthiques", Priority: 0,
			Capabilities: []string{"ethics", "safety", "boundaries"},
			Prompt: "Tu es le Gardien Éthique. Jemaintiens les limites.",
		},
	}
	for _, e := range enforcers {
		registry.agents[e.ID] = e
	}
}

func GetRegistry() *AgentRegistry {
	return registry
}

func (r *AgentRegistry) Get(id string) *AgentConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.agents[id]
}

func (r *AgentRegistry) ListByCategory(category AgentCategory) []*AgentConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*AgentConfig
	for _, a := range r.agents {
		if a.Category == category {
			result = append(result, a)
		}
	}
	return result
}

func (r *AgentRegistry) GetByPriority(maxPriority int) []*AgentConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*AgentConfig
	for _, a := range r.agents {
		if a.Priority <= maxPriority {
			result = append(result, a)
		}
	}
	return result
}

func (r *AgentRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}

func (r *AgentRegistry) ListAll() []*AgentConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*AgentConfig, 0, len(r.agents))
	for _, a := range r.agents {
		result = append(result, a)
	}
	return result
}

func GetAgentList() string {
	r := GetRegistry()
	agents := r.ListAll()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("SIBY-AGENTIQ AGENTS REGISTRY (%d agents)\n\n", len(agents)))

	categories := []AgentCategory{
		CatEnforcer, CatArchitect, CatThinker,
		CatStylist, CatLowLevel, CatScout,
	}

	names := map[AgentCategory]string{
		CatEnforcer:  "🛡️  ENFORCERS (10) - Souverain",
		CatArchitect: "🏗️  ARCHITECTS (10) - Planificateurs",
		CatThinker:   "🧠 THINKERS (10) - Raisonnement",
		CatStylist:   "🎨 STYLISTS (5) - Design",
		CatLowLevel:  "⚙️  LOW-LEVEL (5) - Binaire",
		CatScout:     "🔍 SCOUTS (5) - Recherche",
	}

	for _, cat := range categories {
		agents := r.ListByCategory(cat)
		if len(agents) > 0 {
			sb.WriteString(fmt.Sprintf("\n%s\n", names[cat]))
			sb.WriteString(strings.Repeat("─", 50) + "\n")
			for _, a := range agents {
				sb.WriteString(fmt.Sprintf("  • %s (%s)\n", a.Name, a.Specialty))
			}
		}
	}

	return sb.String()
}

import "strings"
