package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/siby-agentiq/siby-agentiq/internal/config"
	"github.com/siby-agentiq/siby-agentiq/internal/provider"
	"github.com/siby-agentiq/siby-agentiq/internal/scanner"
)

type tickMsg time.Time
type streamChunkMsg string
type streamDoneMsg struct{ usage provider.Usage }

const minWidth = 40

var Nord = struct {
	PolarNight1  = lipgloss.Color("#2E3440")
	PolarNight2  = lipgloss.Color("#3B4252")
	PolarNight3  = lipgloss.Color("#434C5E")
	PolarNight4  = lipgloss.Color("#4C566A")
	SnowStorm1   = lipgloss.Color("#D8DEE9")
	SnowStorm2   = lipgloss.Color("#E5E9F0")
	SnowStorm3   = lipgloss.Color("#ECEFF4")
	Frost1      = lipgloss.Color("#8FBCBB")
	Frost2      = lipgloss.Color("#88C0D0")
	Frost3      = lipgloss.Color("#81A1C1")
	Frost4      = lipgloss.Color("#5E81AC")
	AuroraRed   = lipgloss.Color("#BF616A")
	AuroraOrange = lipgloss.Color("#D08770")
	AuroraYellow = lipgloss.Color("#EBCB8B")
	AuroraGreen  = lipgloss.Color("#A3BE8C")
	AuroraPurple = lipgloss.Color("#B48EAD")
}{
	PolarNight1:  lipgloss.Color("#2E3440"),
	PolarNight2:  lipgloss.Color("#3B4252"),
	PolarNight3:  lipgloss.Color("#434C5E"),
	PolarNight4:  lipgloss.Color("#4C566A"),
	SnowStorm1:   lipgloss.Color("#D8DEE9"),
	SnowStorm2:   lipgloss.Color("#E5E9F0"),
	SnowStorm3:   lipgloss.Color("#ECEFF4"),
	Frost1:       lipgloss.Color("#8FBCBB"),
	Frost2:       lipgloss.Color("#88C0D0"),
	Frost3:       lipgloss.Color("#81A1C1"),
	Frost4:       lipgloss.Color("#5E81AC"),
	AuroraRed:    lipgloss.Color("#BF616A"),
	AuroraOrange:  lipgloss.Color("#D08770"),
	AuroraYellow:  lipgloss.Color("#EBCB8B"),
	AuroraGreen:   lipgloss.Color("#A3BE8C"),
	AuroraPurple:  lipgloss.Color("#B48EAD"),
}

type NordStyles struct {
	Container      lipgloss.Style
	Header         lipgloss.Style
	StatusBar      lipgloss.Style
	InputField     lipgloss.Style
	UserBubble     lipgloss.Style
	AssistantBubble lipgloss.Style
	CodeBlock      lipgloss.Style
	CommandEcho    lipgloss.Style
	ErrorMessage   lipgloss.Style
	Thinking       lipgloss.Style
	ProviderActive lipgloss.Style
	ProviderInactive lipgloss.Style
}

var NordTheme = NordStyles{
	Container: lipgloss.NewStyle().
		Foreground(Nord.SnowStorm1).
		Background(Nord.PolarNight1).
		Margin(0).
		Padding(0),
	Header: lipgloss.NewStyle().
		Foreground(Nord.Frost2).
		Bold(true).
		Padding(0, 1),
	StatusBar: lipgloss.NewStyle().
		Foreground(Nord.SnowStorm1).
		Background(Nord.PolarNight2).
		Padding(0, 2),
	InputField: lipgloss.NewStyle().
		Foreground(Nord.Frost2).
		Background(Nord.PolarNight2).
		Bold(false),
	UserBubble: lipgloss.NewStyle().
		Foreground(Nord.AuroraYellow).
		Padding(0, 1),
	AssistantBubble: lipgloss.NewStyle().
		Foreground(Nord.AuroraGreen).
		Padding(0, 1),
	CodeBlock: lipgloss.NewStyle().
		Foreground(Nord.SnowStorm1).
		Background(Nord.PolarNight2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Nord.Frost4).
		Padding(1, 2).
		Margin(1, 0),
	CommandEcho: lipgloss.NewStyle().
		Foreground(Nord.Frost3).
		Italic(true).
		Padding(0, 1),
	ErrorMessage: lipgloss.NewStyle().
		Foreground(Nord.AuroraRed).
		Bold(true),
	Thinking: lipgloss.NewStyle().
		Foreground(Nord.PolarNight4).
		Italic(true),
	ProviderActive: lipgloss.NewStyle().
		Foreground(Nord.AuroraGreen),
	ProviderInactive: lipgloss.NewStyle().
		Foreground(Nord.PolarNight4),
}

type Message struct {
	Role       string
	Content    string
	Timestamp  time.Time
	Streaming  bool
	TokensUsed int
}

type ChatModel struct {
	cfg          *config.Config
	pm           *provider.ProviderManager
	scanner      *scanner.ProjectScanner
	projectCtx   *scanner.ProjectContext
	workDir      string

	messages     []Message
	input        textinput.Model
	viewport     viewport.Model
	spinner      spinner.Model

	width        int
	height       int
	status       StatusInfo
	err          error

	streaming     bool
	currentStream string
	streamUsage   provider.Usage

	clipboard     bool
	copyIndex     int
}

type StatusInfo struct {
	Provider    string
	Model       string
	Latency     string
	TokensIn    int
	TokensOut   int
	Waiting     bool
	Error       string
}

func NewChatModel(cfg *config.Config, pm *provider.ProviderManager, sc *scanner.ProjectScanner, projCtx *scanner.ProjectContext) *ChatModel {
	ti := textinput.New()
	ti.Placeholder = "Ask Siby-Agentiq... (/help for commands)"
	ti.Focus()
	ti.Prompt = "❯ "
	ti.TextStyle = NordTheme.InputField

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = NordTheme.Thinking

	wd, _ := os.Getwd()
	model := &ChatModel{
		cfg:         cfg,
		pm:          pm,
		scanner:     sc,
		projectCtx:  projCtx,
		workDir:     wd,
		messages:    make([]Message, 0),
		input:       ti,
		spinner:     sp,
		status: StatusInfo{
			Provider: pm.GetActiveName(),
			Model:    getModelName(pm),
		},
	}
	model.viewport = viewport.New(minWidth, 10)
	model.viewport.SetStyle(NordTheme.Container)
	return model
}

func getModelName(pm *provider.ProviderManager) string {
	switch pm.GetActiveName() {
	case "ollama":
		return "llama3.3"
	case "anthropic":
		return "claude-sonnet-4"
	case "openai":
		return "gpt-4o"
	default:
		return "unknown"
	}
}

func (m *ChatModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tickCmd(),
		m.scanProject(),
	)
}

func (m *ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport = viewport.New(msg.Width, msg.Height-5)
		m.viewport.SetStyle(NordTheme.Container)
		return m, nil

	case tickMsg:
		if m.streaming {
			cmds = append(cmds, tickCmd())
		}
		return m, tea.Batch(cmds...)

	case streamChunkMsg:
		m.currentStream += string(msg)
		m.updateViewport()
		cmds = append(cmds, tickCmd())
		return m, nil

	case streamDoneMsg:
		m.streaming = false
		m.streamUsage = msg.usage
		m.messages = append(m.messages, Message{
			Role:       "assistant",
			Content:    m.currentStream,
			Timestamp:  time.Now(),
			Streaming:  false,
			TokensUsed: msg.usage.OutputTokens,
		})
		m.currentStream = ""
		m.status.Waiting = false
		m.status.Latency = fmt.Sprintf("%dms", m.streamUsage.LatencyMS)
		m.updateViewport()
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c":
			if m.streaming {
				m.streaming = false
				m.messages = append(m.messages, Message{
					Role:      "assistant",
					Content:   m.currentStream + "\n[Generation interrupted]",
					Timestamp: time.Now(),
				})
				m.currentStream = ""
				m.updateViewport()
			}
			return m, nil

		case msg.String() == "ctrl+q":
			return m, tea.Quit

		case msg.String() == "ctrl+l":
			m.messages = nil
			m.updateViewport()
			return m, nil

		case msg.String() == "up" && m.input.Value() == "":
			if len(m.messages) > 0 {
				for i := len(m.messages) - 1; i >= 0 {
					if m.messages[i].Role == "user" {
						m.input.SetValue(m.messages[i].Content)
						break
					}
				}
			}
			return m, nil

		case msg.String() == "enter":
			input := strings.TrimSpace(m.input.Value())
			if input == "" {
				return m, nil
			}

			if strings.HasPrefix(input, "/") {
				return m, m.handleCommand(input)
			}

			m.messages = append(m.messages, Message{
				Role:      "user",
				Content:   input,
				Timestamp: time.Now(),
			})
			m.input.Reset()
			m.updateViewport()

			return m, m.sendToLLM(input)

		case msg.String() == "tab":
			m.clipboard = !m.clipboard
			return m, nil

		default:
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *ChatModel) handleCommand(cmd string) tea.Cmd {
	parts := strings.SplitN(cmd, " ", 2)
	command := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	switch command {
	case "/help", "/h":
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   formatHelp(),
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return nil

	case "/clear", "/c":
		m.messages = nil
		m.updateViewport()
		return nil

	case "/model", "/m":
		if args == "" {
			m.messages = append(m.messages, Message{
				Role:      "system",
				Content:   formatProviderStatus(m.pm),
				Timestamp: time.Now(),
			})
			m.updateViewport()
			return nil
		}
		if err := m.pm.SwitchProvider(args); err != nil {
			m.messages = append(m.messages, Message{
				Role:      "system",
				Content:   fmt.Sprintf("Error: %v", err),
				Timestamp: time.Now(),
			})
		} else {
			m.status.Provider = m.pm.GetActiveName()
			m.status.Model = getModelName(m.pm)
			m.messages = append(m.messages, Message{
				Role:      "system",
				Content:   fmt.Sprintf("Switched to %s", args),
				Timestamp: time.Now(),
			})
		}
		m.updateViewport()
		return nil

	case "/scan", "/s":
		return m.scanProject()

	case "/providers", "/p":
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   formatProviderStatus(m.pm),
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return nil

	case "/tokens", "/t":
		total := m.status.TokensIn + m.status.TokensOut
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   fmt.Sprintf("Tokens - In: %d | Out: %d | Total: %d", m.status.TokensIn, m.status.TokensOut, total),
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return nil

	case "/quit", "/q":
		return tea.Quit

	default:
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   fmt.Sprintf("Unknown command: %s. Type /help for available commands.", command),
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return nil
	}
}

func (m *ChatModel) scanProject() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		projCtx, err := m.scanner.Scan(ctx, m.workDir)
		if err != nil {
			return nil
		}
		m.projectCtx = projCtx
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   fmt.Sprintf("Project scanned: %d files, %d lines, %d dependencies",
				projCtx.Summary.TotalFiles, projCtx.Summary.TotalLines, len(projCtx.Summary.Dependencies)),
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return nil
	}
}

func (m *ChatModel) sendToLLM(userInput string) tea.Cmd {
	m.status.Waiting = true
	m.streaming = true
	m.currentStream = ""
	m.updateViewport()

	chainCfg := provider.ChainConfig{
		Enabled:        m.cfg.Agent.ChainOfThought.Enabled,
		ReasoningDepth: m.cfg.Agent.ChainOfThought.ReasoningDepth,
		ShowThinking:   m.cfg.TUI.ShowThinking,
	}

	var projectContext string
	if m.projectCtx != nil {
		projectContext = m.scanner.GetFormattedContext(m.projectCtx, m.cfg.Agent.Context.ContextMode)
	}

	chain := provider.NewChainBuilder(chainCfg, getSystemPrompt(m.cfg), projectContext)
	messages := chain.BuildInitialMessages(userInput)

	return m.streamResponse(messages)
}

func (m *ChatModel) streamResponse(msgs []provider.Message) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := m.pm.SmartStream(ctx, provider.SmartChatRequest{Messages: msgs})
		if err != nil {
			return streamDoneMsg{}
		}

		var totalTokens int
		for {
			chunk, ok := <-ch
			if !ok {
				break
			}
			if chunk.Done {
				return streamDoneMsg{usage: provider.Usage{OutputTokens: totalTokens}}
			}
			totalTokens += utf8.RuneCountInString(chunk.Content)
			tea.Println(streamChunkMsg(chunk.Content))
		}
		return streamDoneMsg{usage: provider.Usage{OutputTokens: totalTokens}}
	}
}

func (m *ChatModel) updateViewport() {
	m.viewport.SetContent(m.renderMessages())
}

func (m *ChatModel) renderMessages() string {
	var sb strings.Builder
	renderer, _ := glamour.NewRenderer(glamour.Configuration{
		Color:               false,
		HTML:                false,
		Mouse:               false,
		Math:                false,
		CodeBlock:           "┌─",
		CodeBlockBackground: string(Nord.PolarNight2),
		HeadingStyles:       true,
		Rule:                true,
		ItemStyles:          true,
		OrderedListStyle:    "numeric",
	})

	for i, msg := range m.messages {
		switch msg.Role {
		case "user":
			sb.WriteString(NordTheme.UserBubble.Render(fmt.Sprintf("You %s", time.Now().Format("15:04"))))
			sb.WriteString("\n")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")

		case "assistant":
			sb.WriteString(NordTheme.AssistantBubble.Render(fmt.Sprintf("Siby %s", time.Now().Format("15:04"))))
			sb.WriteString("\n")
			rendered, _ := renderer.Render(msg.Content)
			sb.WriteString(rendered)
			sb.WriteString("\n")
			if msg.TokensUsed > 0 {
				sb.WriteString(NordTheme.Thinking.Render(fmt.Sprintf("  └─ %d tokens", msg.TokensUsed)))
			}
			sb.WriteString("\n\n")

		case "system":
			sb.WriteString(NordTheme.CommandEcho.Render(msg.Content))
			sb.WriteString("\n\n")
		}
		_ = i
	}

	if m.streaming && m.currentStream != "" {
		sb.WriteString(NordTheme.AssistantBubble.Render("Siby thinking..."))
		sb.WriteString("\n")
		rendered, _ := renderer.Render(m.currentStream)
		sb.WriteString(rendered)
		sb.WriteString("\n")
		sb.WriteString(m.spinner.View())
		sb.WriteString("\n")
	}

	if m.status.Waiting && !m.streaming {
		sb.WriteString(m.spinner.View())
		sb.WriteString(" Waiting for response...\n")
	}

	return sb.String()
}

func (m *ChatModel) View() string {
	return fmt.Sprintf("%s\n%s\n%s",
		m.renderStatusBar(),
		m.viewport.View(),
		m.renderInput(),
	)
}

func (m *ChatModel) renderStatusBar() string {
	statusItems := []string{
		fmt.Sprintf(" %s", NordTheme.Header.Render("Siby-Agentiq")),
		fmt.Sprintf(" │ Provider: %s", NordTheme.ProviderActive.Render(m.status.Provider)),
		fmt.Sprintf(" │ Model: %s", m.status.Model),
	}

	if m.status.Latency != "" {
		statusItems = append(statusItems, fmt.Sprintf(" │ Latency: %s", m.status.Latency))
	}
	if m.status.TokensOut > 0 {
		statusItems = append(statusItems, fmt.Sprintf(" │ Tokens: %d", m.status.TokensOut))
	}

	if m.err != nil {
		statusItems = append(statusItems, NordTheme.ErrorMessage.Render(fmt.Sprintf(" │ Error: %v", m.err)))
	}

	help := NordTheme.Thinking.Render(" [Ctrl+C:Interrupt] [Ctrl+Q:Quit] [Tab:Clipboard]")
	statusItems = append(statusItems, help)

	return NordTheme.StatusBar.Render(strings.Join(statusItems, ""))
}

func (m *ChatModel) renderInput() string {
	input := m.input.View()
	return fmt.Sprintf("\n%s", input)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/10, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func getSystemPrompt(cfg *config.Config) string {
	return `You are Siby-Agentiq, an expert AI coding assistant.
Your role is to help developers with their coding tasks.
Always provide clean, well-documented, and idiomatic code.
Follow best practices for the language/framework being used.
When generating code, use proper formatting and include comments where necessary.
Be concise but thorough in your explanations.
Analyze the problem carefully before providing a solution.`
}

func formatHelp() string {
	return `Available Commands:
  /help, /h          - Show this help message
  /clear, /c         - Clear chat history
  /model [name], /m  - Switch provider (ollama, anthropic, openai)
  /scan, /s          - Rescan current project
  /providers, /p    - Show provider status
  /tokens, /t        - Show token usage
  /quit, /q          - Exit Siby-Agentiq

Keyboard Shortcuts:
  Ctrl+C    - Interrupt generation
  Ctrl+Q    - Quit
  Ctrl+L    - Clear screen
  Up        - Recall last input
  Tab       - Toggle clipboard mode`
}

func formatProviderStatus(pm *provider.ProviderManager) string {
	var sb strings.Builder
	availability := pm.CheckAllAvailability()
	active := pm.GetActiveName()

	sb.WriteString("Provider Status:\n")
	for _, name := range pm.ListProviders() {
		available := availability[name]
		status := NordTheme.ProviderInactive.Render("○")
		if available {
			status = NordTheme.ProviderActive.Render("●")
		}
		marker := "  "
		if name == active {
			marker = "►"
		}
		sb.WriteString(fmt.Sprintf("  %s %s %s\n", marker, status, name))
	}
	return sb.String()
}

type clipboardModel struct {
	show   bool
	text   string
	copied bool
}

var clipboardSupported = clipboard.Available()
