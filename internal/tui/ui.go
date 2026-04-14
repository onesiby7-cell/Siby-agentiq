package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/siby-agentiq/siby-agentiq/internal/config"
)

type Theme struct {
	Primary    lipgloss.Style
	Secondary  lipgloss.Style
	Success    lipgloss.Style
	Warning    lipgloss.Style
	Error      lipgloss.Style
	Background lipgloss.Style
	Text       lipgloss.Style
	Muted      lipgloss.Style
	Thought    lipgloss.Style
	PlanStep   lipgloss.Style
	Border     lipgloss.Style
	Highlight  lipgloss.Style
}

func NewTheme(cfg config.TUIConfig) Theme {
	return Theme{
		Primary: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Colors.Primary)).
			Bold(true),
		
		Secondary: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Colors.Secondary)),
		
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Colors.Success)),
		
		Warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Colors.Warning)),
		
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Colors.Error)),
		
		Background: lipgloss.NewStyle().
			Background(lipgloss.Color(cfg.Colors.Background)),
		
		Text: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Colors.Text)),
		
		Muted: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Colors.Muted)),
		
		Thought: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Colors.Muted)).
			Italic(true).
			Margin(1, 0, 0, 0),
		
		PlanStep: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Colors.Secondary)).
			MarginLeft(2),
		
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			BorderForeground(lipgloss.Color(cfg.Colors.Primary)),
		
		Highlight: lipgloss.NewStyle().
			Background(lipgloss.Color(cfg.Colors.Primary)).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true),
	}
}

type Model struct {
	Theme      Theme
	Width      int
	Height     int
	Input      string
	Messages   []Message
	Thinking   bool
	ShowHelp   bool
	Sidebar    bool
	Cursor     int
	History    []string
	HistoryIdx int
}

type Message struct {
	Role    string
	Content string
	Type    MessageType
}

type MessageType int

const (
	MsgTypeUser MessageType = iota
	MsgTypeAssistant
	MsgTypeThinking
	MsgTypePlan
	MsgTypeError
)

func NewModel(cfg config.TUIConfig) Model {
	return Model{
		Theme:      NewTheme(cfg),
		Width:      cfg.Width,
		Height:     cfg.Height,
		Input:      "",
		Messages:   make([]Message, 0),
		Thinking:   false,
		ShowHelp:   false,
		Sidebar:    false,
		Cursor:     0,
		History:    make([]string, 0),
		HistoryIdx: -1,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.ShowHelp {
				m.ShowHelp = false
			} else {
				return m, tea.Quit
			}
		case "enter":
			if m.Input != "" && !m.Thinking {
				m.History = append(m.History, m.Input)
				m.HistoryIdx = len(m.History)
				m.Messages = append(m.Messages, Message{
					Role:    "user",
					Content: m.Input,
					Type:    MsgTypeUser,
				})
				m.Input = ""
				m.Thinking = true
			}
		case "up":
			if m.HistoryIdx > 0 {
				m.HistoryIdx--
				m.Input = m.History[m.HistoryIdx]
			}
		case "down":
			if m.HistoryIdx < len(m.History)-1 {
				m.HistoryIdx++
				m.Input = m.History[m.HistoryIdx]
			} else {
				m.HistoryIdx = len(m.History)
				m.Input = ""
			}
		case "ctrl+b":
			m.Sidebar = !m.Sidebar
		case "ctrl+l":
			m.Messages = make([]Message, 0)
		case "f1":
			m.ShowHelp = !m.ShowHelp
		}
	}
	return m, nil
}

func (m Model) View() string {
	var s strings.Builder
	
	s.WriteString(m.Theme.Background.Render("\n"))
	
	s.WriteString(m.headerView())
	
	if m.ShowHelp {
		s.WriteString(m.helpView())
		return s.String()
	}
	
	s.WriteString(m.chatView())
	
	s.WriteString(m.inputView())
	
	return s.String()
}

func (m Model) headerView() string {
	title := m.Theme.Primary.Render("Siby-Agentiq")
	version := m.Theme.Muted.Render("v0.1.0")
	status := m.Theme.Success.Render("●")
	
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left, title, version),
		lipgloss.PlaceHorizontal(m.Width, lipgloss.Right, status),
	)
	
	return lipgloss.NewStyle().
		Width(m.Width).
		Border(lipgloss.BottomBorder()).
		Render(header) + "\n"
}

func (m Model) chatView() string {
	if len(m.Messages) == 0 {
		welcome := m.Theme.Muted.Render("Welcome to Siby-Agentiq! Type your request below.")
		return lipgloss.Place(m.Width, m.Height-5, lipgloss.Center, lipgloss.Center, welcome)
	}
	
	var chat strings.Builder
	for _, msg := range m.Messages {
		switch msg.Type {
		case MsgTypeUser:
			chat.WriteString(m.Theme.Secondary.Render("You: ") + msg.Content + "\n")
		case MsgTypeAssistant:
			chat.WriteString(m.Theme.Primary.Render("Siby: ") + msg.Content + "\n")
		case MsgTypeThinking:
			chat.WriteString(m.Theme.Thought.Render("Thinking: " + msg.Content) + "\n")
		case MsgTypePlan:
			chat.WriteString(m.Theme.PlanStep.Render(msg.Content) + "\n")
		case MsgTypeError:
			chat.WriteString(m.Theme.Error.Render("Error: " + msg.Content) + "\n")
		}
	}
	
	if m.Thinking {
		chat.WriteString(m.Theme.Muted.Render("Thinking..."))
	}
	
	return chat.String()
}

func (m Model) inputView() string {
	cursor := " "
	if m.Cursor%2 == 0 {
		cursor = m.Theme.Primary.Render("█")
	}
	
	prompt := m.Theme.Primary.Render("> ")
	input := m.Input + cursor
	
	return lipgloss.NewStyle().
		MarginTop(1).
		Width(m.Width).
		Render(prompt + input)
}

func (m Model) helpView() string {
	help := `
Siby-Agentiq Help
=================
Enter     - Send message
Up/Down   - Navigate history
Ctrl+B    - Toggle sidebar
Ctrl+L    - Clear chat
F1        - Toggle help
Esc/Ctrl+C - Quit
`
	return m.Theme.Border.Render(help)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.showHelp {
				m.showHelp = false
			} else {
				return m, tea.Quit
			}
		case "enter":
			if m.input != "" && !m.thinking {
				m.history = append(m.history, m.input)
				m.historyIdx = len(m.history)
				m.messages = append(m.messages, Message{
					Role:    "user",
					Content: m.input,
					Type:    MsgTypeUser,
				})
				m.input = ""
				m.thinking = true
			}
		case "up":
			if m.historyIdx > 0 {
				m.historyIdx--
				m.input = m.history[m.historyIdx]
			}
		case "down":
			if m.historyIdx < len(m.history)-1 {
				m.historyIdx++
				m.input = m.history[m.historyIdx]
			} else {
				m.historyIdx = len(m.history)
				m.input = ""
			}
		case "ctrl+b":
			m.sidebar = !m.sidebar
		case "ctrl+l":
			m.messages = make([]Message, 0)
		case "f1":
			m.showHelp = !m.showHelp
		}
	}
	return m, nil
}

func (m Model) View() string {
	var s strings.Builder
	
	s.WriteString(m.theme.Background.Render("\n"))
	
	s.WriteString(m.headerView())
	
	if m.showHelp {
		s.WriteString(m.helpView())
		return s.String()
	}
	
	s.WriteString(m.chatView())
	
	s.WriteString(m.inputView())
	
	return s.String()
}

func (m Model) headerView() string {
	title := m.theme.Primary.Render("Siby-Agentiq")
	version := m.theme.Muted.Render("v0.1.0")
	status := m.theme.Success.Render("●")
	
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left, title, version),
		lipgloss.PlaceHorizontal(m.width, lipgloss.Right, status),
	)
	
	return lipgloss.NewStyle().
		Width(m.width).
		Border(lipgloss.BottomBorder()).
		Render(header) + "\n"
}

func (m Model) chatView() string {
	if len(m.messages) == 0 {
		welcome := m.theme.Muted.Render("Welcome to Siby-Agentiq! Type your request below.")
		return lipgloss.Place(m.width, m.height-5, lipgloss.Center, lipgloss.Center, welcome)
	}
	
	var chat strings.Builder
	for _, msg := range m.messages {
		switch msg.Type {
		case MsgTypeUser:
			chat.WriteString(m.theme.Secondary.Render("You: ") + msg.Content + "\n")
		case MsgTypeAssistant:
			chat.WriteString(m.theme.Primary.Render("Siby: ") + msg.Content + "\n")
		case MsgTypeThinking:
			chat.WriteString(m.theme.Thought.Render("Thinking: " + msg.Content) + "\n")
		case MsgTypePlan:
			chat.WriteString(m.theme.PlanStep.Render(msg.Content) + "\n")
		case MsgTypeError:
			chat.WriteString(m.theme.Error.Render("Error: " + msg.Content) + "\n")
		}
	}
	
	if m.thinking {
		chat.WriteString(m.theme.Muted.Render("Thinking..."))
	}
	
	return chat.String()
}

func (m Model) inputView() string {
	cursor := " "
	if m.cursor%2 == 0 {
		cursor = m.theme.Primary.Render("█")
	}
	
	prompt := m.theme.Primary.Render("> ")
	input := m.input + cursor
	
	return lipgloss.NewStyle().
		MarginTop(1).
		Width(m.width).
		Render(prompt + input)
}

func (m Model) helpView() string {
	help := `
Siby-Agentiq Help
=================
Enter     - Send message
Up/Down   - Navigate history
Ctrl+B    - Toggle sidebar
Ctrl+L    - Clear chat
F1        - Toggle help
Esc/Ctrl+C - Quit
`
	return m.theme.Border.Render(help)
}

type BubbleTeaModel struct {
	Model
}

func (b *BubbleTeaModel) SetMessages(msgs []Message) {
	b.Model.messages = msgs
}

func (b *BubbleTeaModel) SetThinking(thinking bool) {
	b.Model.thinking = thinking
}
