package engine

import (
	"fmt"
	"sync"
	"time"
)

type DashboardView struct {
	engine *Engine
	mu     sync.RWMutex
}

func NewDashboardView(e *Engine) *DashboardView {
	return &DashboardView{engine: e}
}

func (dv *DashboardView) Render() string {
	dashboard := dv.engine.GetDashboard()
	return dashboard.Format()
}

func (dv *DashboardView) RenderCompact() string {
	var sb string

	status := dv.engine.status
	agents := dv.engine.GetAgentStatus()

	statusIcon := "○"
	if status == StatusRunning {
		statusIcon = "◉"
	} else if status == StatusError {
		statusIcon = "✗"
	}

	agentStatus := ""
	for id, s := range agents {
		icon := "○"
		if s == AgentWorking {
			icon = "◐"
		}
		agentStatus += fmt.Sprintf("%s%s ", icon, id[:3])
	}

	return fmt.Sprintf("[%s Siby] [%s]", statusIcon, agentStatus)
}

func (dv *DashboardView) RenderFull() string {
	var sb strings.Builder

	dashboard := dv.engine.GetDashboard()

	sb.WriteString("\n")
	sb.WriteString("  ┌─────────────────────────────────────────────────────────┐\n")
	sb.WriteString("  │          SIBY-AGENTIQ REAL-TIME DASHBOARD              │\n")
	sb.WriteString("  ├─────────────────────────────────────────────────────────┤\n")

	statusColor := "\033[92m"
	if dashboard.Status == "error" {
		statusColor = "\033[91m"
	} else if dashboard.Status == "running" {
		statusColor = "\033[93m"
	}

	sb.WriteString(fmt.Sprintf("  │ %s●%s Status: %-10s │ Uptime: %-12s    │\n",
		statusColor, "\033[0m", dashboard.Status, dashboard.Uptime.Round(time.Second)))
	sb.WriteString("  ├─────────────────────────────────────────────────────────┤\n")
	sb.WriteString("  │                    SUB-AGENTS STATUS                      │\n")

	for _, agent := range dashboard.Agents {
		statusIcon := "\033[90m○\033[0m"
		if agent.Status == "working" {
			statusIcon = "\033[93m◐\033[0m"
		} else if agent.Status == "done" {
			statusIcon = "\033[92m●\033[0m"
		} else if agent.Status == "error" {
			statusIcon = "\033[91m✗\033[0m"
		}

		progress := int(agent.Progress * 30)
		progressBar := ""
		for i := 0; i < 30; i++ {
			if i < progress {
				progressBar += "\033[92m█\033[0m"
			} else {
				progressBar += "\033[90m░\033[0m"
			}
		}

		sb.WriteString(fmt.Sprintf("  │ %s %-8s │ %-8s │ [%s] %3.0f%% │\n",
			statusIcon, agent.Name, agent.Role, progressBar, agent.Progress*100))
	}

	sb.WriteString("  ├─────────────────────────────────────────────────────────┤\n")
	sb.WriteString(fmt.Sprintf("  │ Active Tasks: %-3d │ Total: %-3d │ Version: %-10s │\n",
		dashboard.ActiveTasks, dashboard.TotalTasks, dashboard.Version))
	sb.WriteString("  └─────────────────────────────────────────────────────────┘\n")

	return sb.String()
}

type ProgressBar struct {
	Width   int
	Current float32
	Total   float32
	Label   string
}

func NewProgressBar(width int, label string) *ProgressBar {
	return &ProgressBar{
		Width: width,
		Label: label,
	}
}

func (pb *ProgressBar) Set(current, total float32) {
	pb.Current = current
	pb.Total = total
}

func (pb *ProgressBar) Render() string {
	if pb.Total == 0 {
		return fmt.Sprintf("%s: [", pb.Label) + strings.Repeat("░", pb.Width) + "] 0%"
	}

	percent := pb.Current / pb.Total
	filled := int(percent * float32(pb.Width))
	empty := pb.Width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	return fmt.Sprintf("%s: [%s] %3.0f%%", pb.Label, bar, percent*100)
}

import "strings"
