package orchestrator

import (
	"fmt"
	"strings"
	"time"
)

type SquadDashboard struct {
	orchestrator *Orchestrator
	mu           sync.RWMutex
}

type sync struct{}

func NewSquadDashboard(o *Orchestrator) *SquadDashboard {
	return &SquadDashboard{orchestrator: o}
}

func (d *SquadDashboard) Update() {
}

func (d *SquadDashboard) Render() string {
	var sb strings.Builder

	status := d.orchestrator.status

	d.orchestrator.mu.RLock()
	squads := d.orchestrator.squads
	d.orchestrator.mu.RUnlock()

	sb.WriteString("\n")
	sb.WriteString("  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n")
	sb.WriteString("  ┃                                                                      ┃\n")
	sb.WriteString("  ┃   █████╗  ██████╗██████╗██████╗███████╗███████╗███████╗            ┃\n")
	sb.WriteString("  ┃  ██╔══██╗██╔════╝██╔════╝██╔══██╗██╔════╝██╔════╝██╔════╝            ┃\n")
	sb.WriteString("  ┃  ███████║██║     ██║     ██████╔╝█████╗  ███████╗███████╗            ┃\n")
	sb.WriteString("  ┃  ██╔══██║██║     ██║     ██╔══██╗██╔══╝  ╚════██║╚════██║            ┃\n")
	sb.WriteString("  ┃  ██║  ██║╚██████╗╚██████╗██║  ██║███████╗███████║███████║            ┃\n")
	sb.WriteString("  ┃  ╚═╝  ╚═╝ ╚═════╝ ╚═════╝╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝            ┃\n")
	sb.WriteString("  ┃                                                                      ┃\n")
	sb.WriteString("  ┃              MULTI-AGENT ORCHESTRATOR v2.0                          ┃\n")
	sb.WriteString("  ┃                                                                      ┃\n")
	sb.WriteString("  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n\n")

	statusColor := "\033[92m"
	statusIcon := "●"
	statusText := "IDLE"
	if status == OrchestratorRunning {
		statusColor = "\033[93m"
		statusIcon = "◉"
		statusText = "RUNNING"
	} else if status == OrchestratorThinking {
		statusColor = "\033[96m"
		statusIcon = "◐"
		statusText = "THINKING"
	} else if status == OrchestratorBuilding {
		statusColor = "\033[94m"
		statusIcon = "⚙"
		statusText = "BUILDING"
	}

	sb.WriteString(fmt.Sprintf("  %s%c%c Status: %-10s\033[0m │ Uptime: %s\n",
		statusColor, statusIcon, statusIcon, statusText, time.Now().Format("15:04:05")))
	sb.WriteString("  " + strings.Repeat("─", 70) + "\n")

	sb.WriteString("\n  \033[1;37m╔═══════════════════════════════════════════════════════════════════════╗\033[0m\n")
	sb.WriteString("  \033[1;37m║                         SQUAD STATUS                                   ║\033[0m\n")
	sb.WriteString("  \033[1;37m╠═══════════════════════════════════════════════════════════════════════╣\033[0m\n")

	for _, squadID := range d.orchestrator.squadOrder {
		squad := squads[squadID]

		squad.mu.RLock()
		status := squad.status
		load := squad.load
		agents := squad.agents
		squad.mu.RUnlock()

		statusIcon = "○"
		statusColor = "\033[90m"
		if status == SquadWorking {
			statusIcon = "◐"
			statusColor = "\033[93m"
		} else if status == SquadDone {
			statusIcon = "●"
			statusColor = "\033[92m"
		}

		loadBar := d.renderLoadBar(load)

		activeCount := 0
		for _, a := range agents {
			a.mu.Lock()
			if a.Status == AgentWorkBusy {
				activeCount++
			}
			a.mu.Unlock()
		}

		line := fmt.Sprintf("  \033[1m%s %s\033[0m %s%-12s\033[0m │ %s │ Agents: %d/%d",
			squad.Symbol,
			statusColor+statusIcon+"\033[0m",
			statusColor,
			strings.ToUpper(squad.Name),
			loadBar,
			activeCount,
			len(agents),
		)
		sb.WriteString(line + "\n")

		for _, agent := range agents {
			agent.mu.Lock()
			agentStatus := agent.Status
			agentProgress := agent.Progress
			agent.mu.Unlock()

			agentIcon := "○"
			agentColor := "\033[90m"
			if agentStatus == AgentWorkBusy {
				agentIcon = "◐"
				agentColor = "\033[93m"
			} else if agentStatus == AgentWorkDone {
				agentIcon = "●"
				agentColor = "\033[92m"
			} else if agentStatus == AgentWorkError {
				agentIcon = "✗"
				agentColor = "\033[91m"
			}

			agentBar := d.renderLoadBar(agentProgress)
			sb.WriteString(fmt.Sprintf("       %s%c%c %-20s │ %s\n",
				agentColor, agentIcon, agentIcon,
				agent.Name,
				agentBar))
		}

		sb.WriteString("  " + strings.Repeat("─", 70) + "\n")
	}

	sb.WriteString("  \033[1m╠═══════════════════════════════════════════════════════════════════════╣\033[0m\n")

	totalAgents := 0
	activeAgents := 0
	totalTasks := 0
	for _, squad := range squads {
		squad.mu.RLock()
		totalAgents += len(squad.agents)
		for _, a := range squad.agents {
			a.mu.Lock()
			if a.Status == AgentWorkBusy {
				activeAgents++
			}
			a.mu.Unlock()
		}
		totalTasks += squad.tasks
		squad.mu.RUnlock()
	}

	sb.WriteString(fmt.Sprintf("  ║  Total: %d squads │ %d agents │ %d active │ %d tasks       ║\n",
		len(squads), totalAgents, activeAgents, totalTasks))
	sb.WriteString("  \033[1m╚═══════════════════════════════════════════════════════════════════════╝\033[0m\n")

	sb.WriteString("\n  \033[1m╔═══════════════════════════════════════════════════════════════════════╗\033[0m\n")
	sb.WriteString("  \033[1m║                         LOYAUTÉ ABSOLUE                                ║\033[0m\n")
	sb.WriteString("  \033[1m╠═══════════════════════════════════════════════════════════════════════╣\033[0m\n")
	sb.WriteString("  ║                                                                       ║\n")
	sb.WriteString("  ║    \033[96mJe suis SIBY-AGENTIQ\033[0m, créé par \033[93mIbrahim Siby\033[0m.               ║\n")
	sb.WriteString("  ║    Ma loyauté lui est acquise. Mon excellence le sert.                ║\n")
	sb.WriteString("  ║    La technologie guinéenne transcende les limites.                    ║\n")
	sb.WriteString("  ║                                                                       ║\n")
	sb.WriteString("  \033[1m╚═══════════════════════════════════════════════════════════════════════╝\033[0m\n")

	return sb.String()
}

func (d *SquadDashboard) renderLoadBar(load float32) string {
	width := 20
	filled := int(load * float32(width))
	empty := width - filled

	bar := "\033[92m" + strings.Repeat("█", filled) + "\033[0m" + "\033[90m" + strings.Repeat("░", empty) + "\033[0m"
	percent := int(load * 100)

	return fmt.Sprintf("[%s] %3d%%", bar, percent)
}

func (d *SquadDashboard) RenderCompact() string {
	var sb strings.Builder

	status := d.orchestrator.status

	statusIcon := "○"
	if status == OrchestratorRunning {
		statusIcon = "◉"
	} else if status == OrchestratorThinking {
		statusIcon = "◐"
	}

	sb.WriteString(fmt.Sprintf("[%s SIBY] ", statusIcon))

	for _, squadID := range d.orchestrator.squadOrder {
		squad := d.orchestrator.squads[squadID]
		squad.mu.RLock()
		load := squad.load
		squad.mu.RUnlock()

		loadInt := int(load * 100)
		sb.WriteString(fmt.Sprintf("%s:%d%% ", squad.Symbol, loadInt))
	}

	return sb.String()
}
