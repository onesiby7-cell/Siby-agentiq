package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type HeaderModel struct {
	AgentName  string
	Creator    string
	Country    string
	Status     string
	AgentCount int
	ModelName  string
	Provider   string
}

func NewHeader(agentName, creator, country string, agentCount int) *HeaderModel {
	return &HeaderModel{
		AgentName:  agentName,
		Creator:    creator,
		Country:    country,
		Status:     "ONLINE",
		AgentCount: agentCount,
	}
}

func (h *HeaderModel) UpdateStatus(status string) {
	h.Status = status
}

func (h *HeaderModel) UpdateProvider(provider, model string) {
	h.Provider = provider
	h.ModelName = model
}

func (h *HeaderModel) Render() string {
	return fmt.Sprintf(`
================================================================================
  🦂 SIBY-AGENTIQ v2.0.0 SOVEREIGN
  Created by: %s
  Country: %s
  Status: %s | Agents: %d
================================================================================
`, h.AgentName, h.Creator, h.Country, h.Status, h.AgentCount)
}

func RenderSplashScreen() string {
	return fmt.Sprintf(`
╔══════════════════════════════════════════════════════════════════════════════╗
║                                                                              ║
║    ██████╗  ██╗   ██╗  ███████╗                                              ║
║    ██╔══██╗ ██║   ██║  ██╔════╝                                              ║
║    ██████╔╝ ██║   ██║  █████╗                                                ║
║    ██╔══██╗ ██║   ██║  ██╔══╝                                                ║
║    ██║  ██║ ╚██████╔╝  ███████╗                                               ║
║    ╚═╝  ╚═╝  ╚═════╝   ╚══════╝                                               ║
║                                                                              ║
║                      AGENTIQ v2.0.0 - SOVEREIGN MODE                         ║
║                                                                              ║
║                        Created by Ibrahim Siby 🇬🇳                            ║
║                                                                              ║
║                      45 Sub-Agents | Multi-Provider | TUI                     ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝
`)
}

func RenderWelcome() string {
	return fmt.Sprintf(`%s
╔═══════════════════════════════════════════════════════════════════╗
║                                                                   ║
║   🦂 SIBY-AGENTIQ v2.0.0 - SOVEREIGN                           ║
║                                                                   ║
║   Bienvenue, Ibrahim. Tes 45 agents sont prets.                    ║
║                                                                   ║
║   Commandes:                                                       ║
║     /help   - Afficher l'aide                                     ║
║     /scan   - Scanner le projet actuel                            ║
║     /god    - Activer le mode GOD-IA                             ║
║     /scorpion - Recherche web deep                                ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
%s`, lipgloss.Color("#88C0D0"), lipgloss.Color("#D8DEE9"))
}
