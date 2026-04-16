package ui

import "github.com/charmbracelet/lipgloss"

var (
	NordBackground = lipgloss.Color("#2E3440")
	NordPanel      = lipgloss.Color("#3B4252")
	NordPanelLight = lipgloss.Color("#434C5E")
	NordOrange     = lipgloss.Color("#D08770")

	NeonCyan   = lipgloss.Color("#00FFFF")
	NeonPink   = lipgloss.Color("#FF00FF")
	NeonGreen  = lipgloss.Color("#00FF00")
	NeonYellow = lipgloss.Color("#FFFF00")
)

var (
	HeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#88C0D0")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#009460")).
			Padding(0, 1).
			Bold(true)

	SibyBubble = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color("#88C0D0")).
			PaddingLeft(2).
			MarginBottom(1)

	UserBubble = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color("#FCD116")).
			PaddingRight(2).
			MarginBottom(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A3BE8C")).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BF616A")).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EBCB8B"))

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#88C0D0"))

	GuineaFlag = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#009460"))
)
