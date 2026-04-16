package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const minWidth = 40

type Model struct {
	messages []Message
	input    string
	status   StatusInfo
	width    int
	height   int
}

type Message struct {
	Role       string
	Content    string
	Timestamp  time.Time
	TokensUsed int
}

type StatusInfo struct {
	Provider  string
	Model     string
	Latency   string
	TokensIn  int
	TokensOut int
	Waiting   bool
	Ready     bool
}

func NewModel() *Model {
	return &Model{
		messages: []Message{},
		status: StatusInfo{
			Provider: "ollama",
			Waiting:  false,
			Ready:    true,
		},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.input != "" {
				m.messages = append(m.messages, Message{
					Role:      "user",
					Content:   m.input,
					Timestamp: time.Now(),
				})
				m.input = ""
				m.status.Waiting = true
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n  Provider: %s | Ready: %v\n", m.status.Provider, m.status.Ready))
	sb.WriteString(strings.Repeat("─", m.width) + "\n")

	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			sb.WriteString(fmt.Sprintf("You (%s):\n%s\n\n", msg.Timestamp.Format("15:04"), msg.Content))
		case "assistant":
			sb.WriteString(fmt.Sprintf("Siby (%s):\n%s\n\n", msg.Timestamp.Format("15:04"), msg.Content))
		}
	}

	if m.status.Waiting {
		sb.WriteString("Siby is thinking...\n")
	}

	sb.WriteString("\n> " + m.input)
	return sb.String()
}

func (m *Model) AddMessage(role, content string) {
	m.messages = append(m.messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}

var NordStyles = struct {
	Background lipgloss.Style
	Text       lipgloss.Style
	Accent     lipgloss.Style
	Success    lipgloss.Style
	Error      lipgloss.Style
	Warning    lipgloss.Style
}{
	Background: lipgloss.NewStyle().Background(lipgloss.Color("#2E3440")),
	Text:       lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")),
	Accent:     lipgloss.NewStyle().Foreground(lipgloss.Color("#88C0D0")),
	Success:    lipgloss.NewStyle().Foreground(lipgloss.Color("#A3BE8C")),
	Error:      lipgloss.NewStyle().Foreground(lipgloss.Color("#BF616A")),
	Warning:    lipgloss.NewStyle().Foreground(lipgloss.Color("#EBCB8B")),
}
