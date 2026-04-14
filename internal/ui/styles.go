package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (

	NordBackground = lipgloss.Color("#2E3440")
	NordPanel = lipgloss.Color("#3B4252")
	NordPanelLight = lipgloss.Color("#434C5E")
	NordText = lipgloss.Color("#D8DEE9")
	NordTextMuted = lipgloss.Color("#4C566A")
	NordCyan = lipgloss.Color("#88C0D0")
	NordPurple = lipgloss.Color("#B48EAD")
	NordGreen = lipgloss.Color("#A3BE8C")
	NordRed = lipgloss.Color("#BF616A")
	NordYellow = lipgloss.Color("#EBCB8B")
	NordOrange = lipgloss.Color("#D08770")

	GuineaGreen = lipgloss.Color("#009460")
	GuineaYellow = lipgloss.Color("#FCD116")
	GuineaRed = lipgloss.Color("#CE1126")

	NeonCyan = lipgloss.Color("#00FFFF")
	NeonPink = lipgloss.Color("#FF00FF")
	NeonGreen = lipgloss.Color("#00FF00")
	NeonYellow = lipgloss.Color("#FFFF00")
)

var (
	HeaderStyle = lipgloss.NewStyle().
			Foreground(NordCyan).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(GuineaGreen).
			Padding(0, 1).
			Bold(true)

	SibyBubble = lipgloss.NewStyle().
			Border(lipgloss.LeftEdgeBorder()).
			BorderForeground(NordCyan).
			PaddingLeft(2).
			MarginBottom(1)

	UserBubble = lipgloss.NewStyle().
			Border(lipgloss.RightEdgeBorder()).
			BorderForeground(GuineaYellow).
			PaddingRight(2).
			MarginBottom(1)

	InputStyle = lipgloss.NewStyle().
			Foreground(GuineaYellow).
			Italic(true)

	AgentBadge = lipgloss.NewStyle().
			Background(NordBackground).
			Foreground(NordGreen).
			Padding(0, 1).
			Bold(true)

	StatusOnline = lipgloss.NewStyle().
			Foreground(NeonGreen).
			Bold(true)

	StatusOffline = lipgloss.NewStyle().
			Foreground(NordRed).
			Bold(true)

	CodeBlock = lipgloss.NewStyle().
			Background(NordPanel).
			Foreground(NordGreen).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(NordCyan).
			Padding(1, 2)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(NordRed).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(NordGreen).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(NordYellow).
			Bold(true)

	ScorpionProgress = lipgloss.NewStyle().
				Foreground(GuineaYellow).
				Background(NordPanel)

	GodIAMode = lipgloss.NewStyle().
			Foreground(GuineaRed).
			Bold(true)

	FooterStyle = lipgloss.NewStyle().
			Foreground(NordTextMuted).
			Background(NordPanel)

	PanelBorder = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(NordPanelLight)

	TitleStyle = lipgloss.NewStyle().
			Foreground(NordCyan).
			Bold(true).
			Underline(true)
)

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
