package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type HeaderModel struct {
	AgentName   string
	Creator     string
	Country     string
	Status      string
	AgentCount  int
	ModelName   string
	Provider    string
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
	statusColor := NordGreen
	if h.Status != "ONLINE" {
		statusColor = NordRed
	}

	statusIcon := "●"
	if h.Status == "OFFLINE" {
		statusIcon = "○"
	} else if h.Status == "PROCESSING" {
		statusIcon = "◐"
	}

	headerWidth := 78

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundBorder()).
		BorderForeground(NordCyan).
		BorderBackground(NordBackground).
		Padding(0, 1)

	content := lipgloss.JoinHorizontal(
		lipgloss.Left,
		NordCyan+lipgloss.Bold().Render(" 🦂 ")+h.AgentName+lipgloss.ResetStyle().String(),
		NordTextMuted+" │ ",
		GuineaRed+GuineaYellow+GuineaGreen+" "+h.Country+" "+lipgloss.ResetStyle().String(),
		NordTextMuted+" │ ",
		statusColor+lipgloss.Bold().Render(statusIcon)+" "+h.Status+lipgloss.ResetStyle().String(),
		NordTextMuted+" │ ",
		NordGreen+lipgloss.Bold().Render("🤖")+NordText+" "+fmt.Sprintf("%d", h.AgentCount)+" agents"+lipgloss.ResetStyle().String(),
	)

	if h.Provider != "" {
		content += NordTextMuted + " │ " + NordCyan + h.Provider + lipgloss.ResetStyle().String()
	}

	padded := lipgloss.NewStyle().
		Width(headerWidth).
		Align(lipgloss.Left).
		Foreground(NordText).
		Background(NordBackground).
		Render(content)

	return padded
}

func RenderSplashScreen() string {
	splash := fmt.Sprintf(`
%s%s
%s╔══════════════════════════════════════════════════════════════════════════════╗
║                                                                              ║
║   %s██████╗%s  ██╗%s  ██████╗%s  ██╗  ██╗       %s██████╗%s  ███████╗%s  ██╗  ██╗%s  ██████╗%s  ████████╗%s  ██╗%s  ██████╗   ██╗       ██╗%s   ║
║   %s██╔══██╗%s  ██║%s  ██╔══██╗%s  ██║  ██║       %s██╔═══██╗%s  ██╔════╝%s  ██║  ██║%s  ██╔══██╗%s  ██╔════╝%s  ██║%s  ██╔══██╗   ██║       ██║%s   ║
║   %s██████╔╝%s  ██║%s  ██████╔╝%s  ███████║       %s██║   ██║%s  █████╗  %s  ███████║%s  ██████╔╝%s  █████╗  %s  ██║%s  ██████╔╝   ██║       ██║%s   ║
║   %s██╔══██╗%s  ██║%s  ██╔══██╗%s  ██╔══██║       %s██║   ██║%s  ██╔══╝  %s  ██╔══██║%s  ██╔══██╗%s  ██╔══╝  %s  ██║%s  ██╔══██╗   ██║       ██║%s   ║
║   %s██║  ██║%s  ██║%s  ██║  ██║%s  ██║  ██║       %s╚██████╔╝%s  ███████╗%s  ██║  ██║%s  ██║  ██║%s  ███████╗%s  ██║%s  ██║  ██║   ███████╗%s   ║
║   %s╚═╝  ╚═╝%s  ╚═╝%s  ╚═╝  ╚═╝%s  ╚═╝  ╚═╝       %s ╚═════╝ %s  ╚══════╝%s  ╚═╝  ╚═╝%s  ╚═╝  ╚═╝%s  ╚══════╝%s  ╚═╝%s  ╚═╝  ╚═╝   ╚══════╝%s   ║
║                                                                              ║
╠══════════════════════════════════════════════════════════════════════════════╣
║                                                                              ║
║   %s┌─────────────────────────────────────────────────────────────────────┐   ║
║   %s│     %sPROUDLY BUILT IN GUINEA BY IBRAHIM SIBY 🇬🇳%s                        │   ║
║   %s│     %sMOTEUR : 45 AGENTS ACTIFS | MODE : GOD-IA%s                          │   ║
║   %s│     %sThe Last Agent You Will Ever Need%s                                   │   ║
║   %s└─────────────────────────────────────────────────────────────────────┘   ║
║                                                                              ║
║   %s⌘ Commands: /help | /scan | /model | leader-siby (secret)%s                 ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝
%s`,
		lipgloss.ResetStyle(),
		NordCyan,
		NordCyan, NordYellow, NordCyan, NordGreen, NordCyan, NordRed,
		NordCyan, NordGreen, NordCyan, NordYellow, NordCyan, NordGreen, NordCyan,
		NordCyan, NordRed, NordCyan, NordGreen, NordCyan, NordYellow, NordCyan,
		NordCyan, NordGreen, NordCyan, NordRed, NordCyan, NordYellow, NordCyan,
		NordCyan,
		NordCyan, NordYellow, NordCyan, NordGreen, NordCyan, NordRed,
		NordCyan, NordGreen, NordCyan, NordYellow, NordCyan, NordGreen, NordCyan,
		NordCyan, NordRed, NordCyan, NordGreen, NordCyan, NordYellow, NordCyan,
		NordCyan, NordGreen, NordCyan, NordRed, NordCyan, NordYellow, NordCyan,
		NordCyan,
		NordCyan,
		NordYellow,
		NordGreen,
		NordCyan,
		NordTextMuted,
		NordCyan,
		NordYellow,
		lipgloss.ResetStyle(),
	)

	return splash
}

func RenderFooter(tokens, latency string, model string) string {
	footerWidth := 78

	bar := NordTextMuted + "│ " + lipgloss.ResetStyle().String()
	bar += NordYellow + "/ask " + NordTextMuted + "│ " + lipgloss.ResetStyle().String()
	bar += NordCyan + "Tokens: " + NordText + tokens + NordTextMuted + " │ " + lipgloss.ResetStyle().String()
	bar += NordGreen + "Model: " + NordText + model + NordTextMuted + " │ " + lipgloss.ResetStyle().String()
	bar += NordPurple + "Latence: " + NordText + latency + NordTextMuted + " │ " + lipgloss.ResetStyle().String()
	bar += NordYellow + "🦂" + lipgloss.ResetStyle().String()

	padded := lipgloss.NewStyle().
		Width(footerWidth).
		Foreground(NordTextMuted).
		Background(NordPanel).
		Render(bar)

	return padded
}

func RenderLoadingAnimation(prefix string, progress float64) string {
	bar := RenderGradientBar(progress, 40)
	time.Sleep(10 * time.Millisecond)
	return fmt.Sprintf("%s %s %d%%", prefix, bar, int(progress*100))
}

func RenderScorpionLoading(query string) string {
	frames := []string{
		"🦂 QUERY: " + query + " [□□□□□□□□□□] 0%",
		"🦂 QUERY: " + query + " [■□□□□□□□□□] 10%",
		"🦂 QUERY: " + query + " [■■■□□□□□□□□] 30%",
		"🦂 QUERY: " + query + " [■■■■■□□□□□□] 50%",
		"🦂 QUERY: " + query + " [■■■■■■■□□□□] 70%",
		"🦂 QUERY: " + query + " [■■■■■■■■□□□] 90%",
		"🦂 QUERY: " + query + " [■■■■■■■■■■■] 100%",
	}

	var result strings.Builder
	for i, frame := range frames {
		time.Sleep(100 * time.Millisecond)
		result.WriteString("\r" + GuineaYellow + frame + lipgloss.ResetStyle().String())
		if i == len(frames)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func RenderGradientBar(progress float64, width int) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	filled := int(float64(width) * progress)
	empty := width - filled

	gradient := ""
	for i := 0; i < filled; i++ {
		pos := float64(i) / float64(width)
		if pos < 0.33 {
			gradient += NordRed
		} else if pos < 0.66 {
			gradient += NordYellow
		} else {
			gradient += NordGreen
		}
		gradient += "█"
	}

	gradient += NordTextMuted
	for i := 0; i < empty; i++ {
		gradient += "░"
	}

	return gradient + lipgloss.ResetStyle("").String()
}

func RenderScorpionBar(progress float64, width int) string {
	filled := int(float64(width) * progress)
	empty := width - filled

	bar := GuineaYellow + "█" + GuineaGreen + "█"
	if filled > 2 {
		bar += strings.Repeat("█", filled-2)
	}
	bar += NordTextMuted + strings.Repeat("░", empty)

	return bar + lipgloss.ResetStyle("").String()
}

func RenderPulseIndicator() string {
	frames := []string{"●", "◐", "○", "◑"}
	idx := (int(time.Now().UnixNano()) / 100000000) % len(frames)
	return NordGreen + frames[idx] + lipgloss.ResetStyle("").String()
}

func RenderNeonText(text string, color lipgloss.Color) string {
	return color + lipgloss.Bold().Render(text) + lipgloss.ResetStyle("").String()
}

func RenderGuineaFlag() string {
	return GuineaRed + "▓" + GuineaYellow + "▓" + GuineaGreen + "▓" + lipgloss.ResetStyle("").String()
}
