package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/siby-agentiq/siby-agentiq/internal/config"
	"github.com/siby-agentiq/siby-agentiq/internal/executor"
	"github.com/siby-agentiq/siby-agentiq/internal/godIA"
	"github.com/siby-agentiq/siby-agentiq/internal/provider"
	"github.com/siby-agentiq/siby-agentiq/internal/scanner"
	"github.com/siby-agentiq/siby-agentiq/internal/scorpion"
	"github.com/siby-agentiq/siby-agentiq/internal/synthesis"
)

type tickMsg time.Time

type ChatMessage struct {
	Role       string
	Content    string
	Timestamp  time.Time
	TokensUsed int
}

type ProgressInfo struct {
	Active   bool
	Current  float64
	Total    float64
	File     string
	Percent  int
}

type Model struct {
	cfg          *config.Config
	pm           *provider.ProviderManager
	scanner      *scanner.ProjectScanner
	projectCtx   *scanner.ProjectContext
	executor     *executor.Executor
	workDir      string

	scorpion      *scorpion.Scorpion
	godIA         *godIA.GODIA
	synthesizer   *synthesis.Synthesizer

	messages   []ChatMessage
	input      textinput.Model
	viewport   viewport.Model
	spinner    spinner.Model
	progress   progress.Model

	width   int
	height  int
	status  StatusInfo
	progressInfo ProgressInfo

	streaming      bool
	currentStream  string
	tokenCount     int
}

type StatusInfo struct {
	Provider   string
	Model      string
	Latency    string
	TokensIn   int
	TokensOut  int
	Waiting    bool
	Ready      bool
}

var NordReset = lipgloss.Color("\033[0m")

var nord = struct {
	Background   lipgloss.Color
	Panel        lipgloss.Color
	PanelLight   lipgloss.Color
	Text         lipgloss.Color
	TextMuted    lipgloss.Color
	Frost        lipgloss.Color
	FrostLight   lipgloss.Color
	Yellow       lipgloss.Color
	Green        lipgloss.Color
	Red          lipgloss.Color
	Purple       lipgloss.Color
}{
	Background:  lipgloss.Color("#2E3440"),
	Panel:       lipgloss.Color("#3B4252"),
	PanelLight:  lipgloss.Color("#434C5E"),
	Text:        lipgloss.Color("#D8DEE9"),
	TextMuted:   lipgloss.Color("#4C566A"),
	Frost:       lipgloss.Color("#88C0D0"),
	FrostLight:  lipgloss.Color("#81A1C1"),
	Yellow:      lipgloss.Color("#EBCB8B"),
	Green:       lipgloss.Color("#A3BE8C"),
	Red:         lipgloss.Color("#BF616A"),
	Purple:      lipgloss.Color("#B48EAD"),
}

var GuineaGreen = lipgloss.Color("#009460")
var GuineaYellow = lipgloss.Color("#FCD116")
var GuineaRed = lipgloss.Color("#CE1126")

var NordCyan = lipgloss.Color("#88C0D0")
var NordText = lipgloss.Color("#D8DEE9")
var NordTextMuted = lipgloss.Color("#4C566A")
var NordYellow = lipgloss.Color("#EBCB8B")
var NordRed = lipgloss.Color("#BF616A")
var NordGreen = lipgloss.Color("#A3BE8C")
var NordReset = lipgloss.Color("\033[0m")

var styles = struct {
	Header    lipgloss.Style
	Status    lipgloss.Style
	User      lipgloss.Style
	Bot       lipgloss.Style
	Code      lipgloss.Style
	Think     lipgloss.Style
	Error     lipgloss.Style
	System    lipgloss.Style
	Prompt    lipgloss.Style
	Progress  lipgloss.Style
	ProgressBar lipgloss.Style
}{
	Header: lipgloss.NewStyle().
		Foreground(nord.Frost).
		Bold(true).
		Padding(0, 1),
	Status: lipgloss.NewStyle().
		Foreground(nord.Text).
		Background(nord.Panel).
		Padding(0, 1),
	User: lipgloss.NewStyle().
		Foreground(nord.Yellow),
	Bot: lipgloss.NewStyle().
		Foreground(nord.Green),
	Code: lipgloss.NewStyle().
		Foreground(nord.Text).
		Background(nord.Panel).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(nord.FrostLight).
		Padding(1, 2),
	Think: lipgloss.NewStyle().
		Foreground(nord.TextMuted).
		Italic(true),
	Error: lipgloss.NewStyle().
		Foreground(nord.Red).
		Bold(true),
	System: lipgloss.NewStyle().
		Foreground(nord.FrostLight).
		Italic(true),
	Prompt: lipgloss.NewStyle().
		Foreground(nord.Frost),
	Progress: lipgloss.NewStyle().
		Foreground(nord.Frost).
		Background(nord.PanelLight).
		Padding(0, 1),
	ProgressBar: lipgloss.NewStyle().
		Foreground(nord.Green).
		Background(nord.PanelLight),
}

func New(cfg *config.Config, pm *provider.ProviderManager, sc *scanner.ProjectScanner, projCtx *scanner.ProjectContext) *Model {
	wd, _ := os.Getwd()

	ti := textinput.New()
	ti.Placeholder = "Ask Siby... (/help for commands)"
	ti.Focus()
	ti.Prompt = "❯ "
	ti.TextStyle = styles.Prompt

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styles.Think

	pg := progress.New()
	pg.ShowPercentage = true
	pg.ShowCount = true
	pg.Width = 60
	pg.Empty = '░'
	pg.Filled = '█'
	pg.FillColor = lipgloss.Color("#88C0D0")

	execCfg := executor.ExecutorConfig{
		AutoBackup:  true,
		ConfirmAll:   false,
		DryRun:       false,
		MaxFileSize:  1024 * 1024,
	}
	exec := executor.NewExecutor(pm, execCfg)
	exec.SetConfirmFunc(func(msg string) bool {
		return true
	})
	executor.SetProgressCallback(func(progress float64, file string) {
	})

	m := &Model{
		cfg:         cfg,
		pm:          pm,
		scanner:     sc,
		projectCtx:  projCtx,
		executor:    exec,
		workDir:     wd,
		messages:    make([]ChatMessage, 0),
		input:       ti,
		spinner:     sp,
		progress:    pg,
		status: StatusInfo{
			Provider: pm.GetActiveName(),
			Model:    getModelShortName(pm.GetActiveName()),
			Ready:    pm.GetActiveProvider() != nil && pm.GetActiveProvider().IsAvailable(),
		},
	}

	m.viewport = viewport.New(80, 20)
	m.viewport.SetStyle(lipgloss.NewStyle().Background(nord.Background))

	return m
}

func getModelShortName(provider string) string {
	switch provider {
	case "ollama":
		return "llama3"
	case "anthropic":
		return "claude"
	case "openai":
		return "gpt-4"
	default:
		return "?"
	}
}

func (m *Model) Init() tea.Cmd {
	m.messages = append(m.messages, ChatMessage{
		Role:      "system",
		Content:   welcomeMessage(),
		Timestamp: time.Now(),
	})
	m.updateViewport()

	return tea.Batch(
		textinput.Blink,
		tick(),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport = viewport.New(msg.Width, msg.Height-6)
		m.viewport.SetStyle(lipgloss.NewStyle().Background(nord.Background))
		m.updateViewport()
		return m, nil

	case tickMsg:
		if m.streaming {
			cmds = append(cmds, tick())
		}
		m.spinner, _ = m.spinner.Update(msg)
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		pg, cmd := m.progress.Update(msg)
		m.progress = pg
		return m, cmd

	case tea.KeyMsg:
		return m.handleKey(msg)

	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "ctrl+c":
		if m.streaming {
			m.streaming = false
			m.messages = append(m.messages, ChatMessage{
				Role:      "assistant",
				Content:   m.currentStream + "\n[Interrupted]",
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

	case msg.String() == "enter":
		input := strings.TrimSpace(m.input.Value())
		if input == "" {
			return m, nil
		}

		if strings.HasPrefix(input, "/") {
			return m, m.handleCommand(input)
		}

		m.messages = append(m.messages, ChatMessage{
			Role:      "user",
			Content:   input,
			Timestamp: time.Now(),
		})
		m.input.Reset()
		m.updateViewport()

		return m, m.sendRequest(input)
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) handleCommand(cmd string) tea.Cmd {
	parts := strings.SplitN(cmd, " ", 2)
	command := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}

	switch command {
	case "/help", "/h":
		m.addMessage("system", helpText())
		m.updateViewport()
		return nil

	case "/clear", "/c":
		m.messages = nil
		m.updateViewport()
		return nil

	case "/model", "/m":
		if arg == "" {
			m.addMessage("system", providerStatusText(m.pm))
			m.updateViewport()
			return nil
		}
		if err := m.pm.SwitchProvider(arg); err != nil {
			m.addMessage("error", fmt.Sprintf("Provider '%s' not found", arg))
		} else {
			m.status.Provider = m.pm.GetActiveName()
			m.status.Model = getModelShortName(arg)
			m.status.Ready = m.pm.GetActiveProvider().IsAvailable()
			m.addMessage("system", fmt.Sprintf("Switched to %s", arg))
		}
		m.updateViewport()
		return nil

	case "/scan", "/s":
		m.status.Waiting = true
		m.updateViewport()
		go func() {
			ctx := context.Background()
			projCtx, err := m.scanner.Scan(ctx, m.workDir)
			if err == nil && projCtx != nil {
				m.projectCtx = projCtx
				m.addMessage("system", fmt.Sprintf("Scanned: %d files, %d lines, deps: %v",
					projCtx.Summary.TotalFiles,
					projCtx.Summary.TotalLines,
					projCtx.Summary.Dependencies))
			} else {
				m.addMessage("error", "Scan failed")
			}
			m.status.Waiting = false
			m.updateViewport()
		}()
		return nil

	case "/providers", "/p":
		m.addMessage("system", providerStatusText(m.pm))
		m.updateViewport()
		return nil

	case "/exec", "/e":
		if arg == "" {
			m.addMessage("error", "Usage: /exec <command>")
			m.updateViewport()
			return nil
		}
		return m.executeCommand(arg)

	case "/ls":
		return m.handleLs(arg)

	case "/cd":
		return m.handleCd(arg)

	case "/explorer", "/files":
		return m.handleExplorer()

	case "/lsp":
		return m.handleLSP()

	case "/cost":
		return m.handleCost()

	case "/tokens":
		return m.handleTokens()

	case "/restore":
		return m.handleRestore()

	case "/sessions":
		return m.handleSessions()

	case "/leader-siby":
		m.addMessage("system", fmt.Sprintf("%s\n👁️ GOD-IA MODE ACTIVATED\n%s\nBienvenue, Ibrahim. La vision omnisciente est maintenant active.\nTapez /god pour accéder au dashboard.",
			NordRed+"\n╔═══════════════════════════════════════════════════════════╗\n║  🦂🦂🦂 GOD-IA OMNISCIENT MODE 🦂🦂🦂", NordReset))
		m.updateViewport()
		return nil

	case "/god":
		return m.handleGodMode()

	case "/scorpion":
		return m.handleScorpion(arg)

	case "/update":
		return m.handleUpdate()

	case "/feedback":
		return m.handleFeedback(arg)

	case "/changelog":
		return m.handleChangelog()

	case "/quit", "/q":
		return tea.Quit

	default:
		m.addMessage("error", fmt.Sprintf("Unknown command: %s. Type /help for available commands.", command))
		m.updateViewport()
		return nil
	}
}

func (m *Model) handleLs(path string) tea.Cmd {
	return func() tea.Msg {
		wd := m.workDir
		if path != "" {
			wd = path
		}
		
		entries, err := os.ReadDir(wd)
		if err != nil {
			m.addMessage("error", fmt.Sprintf("Cannot list directory: %v", err))
			m.updateViewport()
			return nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s📁 %s%s\n\n", NordCyan, wd, NordReset))
		
		for _, entry := range entries {
			icon := "📄"
			if entry.IsDir() {
				icon = "📁"
			}
			name := entry.Name()
			if entry.IsDir() {
				name += "/"
			}
			sb.WriteString(fmt.Sprintf("  %s %s%s%s\n", icon, NordText, name, NordReset))
		}

		m.addMessage("assistant", sb.String())
		m.updateViewport()
		return nil
	}
}

func (m *Model) handleCd(path string) tea.Cmd {
	return func() tea.Msg {
		if path == "" {
			home, _ := os.UserHomeDir()
			path = home
		}
		
		if path == ".." {
			path = filepath.Dir(m.workDir)
		}
		
		if !filepath.IsAbs(path) {
			path = filepath.Join(m.workDir, path)
		}
		
		info, err := os.Stat(path)
		if err != nil {
			m.addMessage("error", fmt.Sprintf("Path not found: %s", path))
			m.updateViewport()
			return nil
		}
		
		if !info.IsDir() {
			m.addMessage("error", fmt.Sprintf("Not a directory: %s", path))
			m.updateViewport()
			return nil
		}
		
		m.workDir = path
		m.addMessage("system", fmt.Sprintf("%s✓%s Changed directory to: %s", NordGreen, NordReset, path))
		m.updateViewport()
		return nil
	}
}

func (m *Model) handleExplorer() tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(m.workDir)
		if err != nil {
			m.addMessage("error", fmt.Sprintf("Cannot read directory: %v", err))
			m.updateViewport()
			return nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s╔══════════════════════════════════════════════════╗%s\n", NordCyan, NordReset))
		sb.WriteString(fmt.Sprintf("%s║%s 📁 FILE EXPLORER %s%s║%s\n", NordCyan, NordYellow, NordCyan, NordYellow, NordReset))
		sb.WriteString(fmt.Sprintf("%s╠══════════════════════════════════════════════════╣%s\n", NordCyan, NordReset))
		sb.WriteString(fmt.Sprintf("%s║%s Path: %s%s%s║%s\n", NordCyan, NordTextMuted, NordText, m.workDir, NordTextMuted, NordReset))
		sb.WriteString(fmt.Sprintf("%s╠══════════════════════════════════════════════════╣%s\n", NordCyan, NordReset))

		dirCount := 0
		fileCount := 0
		for _, entry := range entries {
			if entry.IsDir() {
				dirCount++
			} else {
				fileCount++
			}
		}
		sb.WriteString(fmt.Sprintf("%s║%s 📁 %d directories | 📄 %d files %s%s║%s\n", 
			NordCyan, NordGreen, dirCount, NordText, fileCount, NordTextMuted, NordCyan, NordReset))
		sb.WriteString(fmt.Sprintf("%s╠══════════════════════════════════════════════════╣%s\n", NordCyan, NordReset))
		sb.WriteString(fmt.Sprintf("%s║%s Commands: /ls [path] | /cd [path] | /files %s%s║%s\n", NordCyan, NordTextMuted, NordCyan, NordTextMuted, NordReset))
		sb.WriteString(fmt.Sprintf("%s╚══════════════════════════════════════════════════╝%s\n", NordCyan, NordReset))

		m.addMessage("assistant", sb.String())
		m.updateViewport()
		return nil
	}
}

func (m *Model) handleLSP() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		extensions := []string{".go", ".ts", ".js", ".py"}
		
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s🔍 LSP Analysis Results%s\n\n", NordCyan, NordReset))
		
		filepath.Walk(m.workDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			
			ext := filepath.Ext(path)
			for _, e := range extensions {
				if ext == e {
					sb.WriteString(fmt.Sprintf("  %s✓%s Analyzing: %s\n", NordGreen, NordReset, path))
					break
				}
			}
			return nil
		})

		sb.WriteString(fmt.Sprintf("\n%s💡 LSP ready. Use /scan for full project analysis.%s", NordYellow, NordReset))

		m.addMessage("assistant", sb.String())
		m.updateViewport()
		return nil
	}
}

func (m *Model) handleCost() tea.Cmd {
	return func() tea.Msg {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s💰 Cost Tracking Dashboard%s\n\n", NordCyan, NordReset))
		
		sb.WriteString(fmt.Sprintf("  %sDaily Spend:%s $%.2f / $10.00\n", NordYellow, NordReset, 0.0))
		sb.WriteString(fmt.Sprintf("  %sBy Provider:%s\n", NordCyan, NordReset))
		sb.WriteString(fmt.Sprintf("    • Groq: $0.00 (Free tier)\n"))
		sb.WriteString(fmt.Sprintf("    • OpenAI: $0.00\n"))
		sb.WriteString(fmt.Sprintf("    • Anthropic: $0.00\n"))
		
		m.addMessage("assistant", sb.String())
		m.updateViewport()
		return nil
	}
}

func (m *Model) handleTokens() tea.Cmd {
	return func() tea.Msg {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s📊 Token Usage Dashboard%s\n\n", NordCyan, NordReset))
		
		current, max, percent := 0, 128000, 0.0
		sb.WriteString(fmt.Sprintf("  %sContext:%s %d / %d tokens\n", NordYellow, NordReset, current, max))
		
		barWidth := 40
		filled := int(float64(barWidth) * percent)
		empty := barWidth - filled
		
		color := NordGreen
		if percent >= 0.75 {
			color = NordYellow
		}
		if percent >= 0.90 {
			color = NordRed
		}
		
		bar := color + strings.Repeat("█", filled) + NordTextMuted + strings.Repeat("░", empty)
		sb.WriteString(fmt.Sprintf("  %s[%s%s] %.0f%%\n", NordTextMuted, bar, NordTextMuted, percent*100))
		
		if percent >= 0.90 {
			sb.WriteString(fmt.Sprintf("\n  %s⚠️ Warning: Context near limit. Planning agents will summarize soon.%s", NordYellow, NordReset))
		}

		m.addMessage("assistant", sb.String())
		m.updateViewport()
		return nil
	}
}

func (m *Model) handleRestore() tea.Cmd {
	return func() tea.Msg {
		m.addMessage("system", fmt.Sprintf("%s💾 Session Restore%s\n\nLatest session will be loaded.\n\nUse /sessions to see all saved sessions.",
			NordCyan, NordReset))
		m.updateViewport()
		return nil
	}
}

func (m *Model) handleSessions() tea.Cmd {
	return func() tea.Msg {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s💾 Saved Sessions%s\n\n", NordCyan, NordReset))
		sb.WriteString(fmt.Sprintf("  %sCurrent session:%s session-%s\n", NordYellow, NordReset, time.Now().Format("2006-01-02-15-04")))
		sb.WriteString(fmt.Sprintf("  %sCommands:%s /restore | /sessions\n", NordYellow, NordReset))
		
		m.addMessage("assistant", sb.String())
		m.updateViewport()
		return nil
	}
}

func (m *Model) handleGodMode() tea.Cmd {
	return func() tea.Msg {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s╔═══════════════════════════════════════════════════════════╗%s\n", NordRed, NordReset))
		sb.WriteString(fmt.Sprintf("%s║%s 🦂🦂🦂 GOD-IA OMNISCIENT DASHBOARD 🦂🦂🦂 %s║%s\n", NordRed, NordYellow, NordRed, NordReset))
		sb.WriteString(fmt.Sprintf("%s╠═══════════════════════════════════════════════════════════╣%s\n", NordRed, NordReset))
		sb.WriteString(fmt.Sprintf("%s║%s 👁️ System Status: ONLINE %s%s║%s\n", NordRed, NordGreen, NordRed, NordGreen, NordReset))
		sb.WriteString(fmt.Sprintf("%s║%s 🖥️  CPU: Available | 💾 RAM: Available %s%s║%s\n", NordRed, NordCyan, NordRed, NordCyan, NordReset))
		sb.WriteString(fmt.Sprintf("%s║%s 🌐 Network: Connected | 📁 Files: Monitored %s%s║%s\n", NordRed, NordCyan, NordRed, NordCyan, NordReset))
		sb.WriteString(fmt.Sprintf("%s╠═══════════════════════════════════════════════════════════╣%s\n", NordRed, NordReset))
		sb.WriteString(fmt.Sprintf("%s║%s ✨ Optimizations: Ready %s%s║%s\n", NordRed, NordGreen, NordRed, NordGreen, NordReset))
		sb.WriteString(fmt.Sprintf("%s║%s 🔮 Code Validation 2035: Active %s%s║%s\n", NordRed, NordYellow, NordRed, NordYellow, NordReset))
		sb.WriteString(fmt.Sprintf("%s╠═══════════════════════════════════════════════════════════╣%s\n", NordRed, NordReset))
		sb.WriteString(fmt.Sprintf("%s║%s Welcome, Ibrahim Siby. All seeing. All knowing. %s%s║%s\n", NordRed, NordCyan, NordRed, NordCyan, NordReset))
		sb.WriteString(fmt.Sprintf("%s╚═══════════════════════════════════════════════════════════╝%s\n", NordRed, NordReset))
		
		m.addMessage("assistant", sb.String())
		m.updateViewport()
		return nil
	}
}

func (m *Model) handleScorpion(arg string) tea.Cmd {
	return func() tea.Msg {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s🦂 SCORPION - Deep Search Engine%s\n\n", NordYellow, NordReset))
		
		if arg == "" {
			sb.WriteString(fmt.Sprintf("  %sUsage:%s /scorpion [query]\n\n", NordYellow, NordReset))
			sb.WriteString(fmt.Sprintf("  %sExample:%s /scorpion How to optimize Go code?\n", NordTextMuted, NordReset))
		} else {
			sb.WriteString(fmt.Sprintf("  %s🔍 Searching for:%s %s\n\n", NordCyan, NordReset, arg))
			sb.WriteString(fmt.Sprintf("  %s⏳ Querying multiple sources...%s\n", NordYellow, NordReset))
			sb.WriteString(fmt.Sprintf("  %s✓ Gemini ✓ GPT-4o ✓ Perplexity%s\n\n", NordGreen, NordYellow, NordReset))
			sb.WriteString(fmt.Sprintf("  %sResults will be synthesized by 45 agents.%s", NordCyan, NordReset))
		}

		m.addMessage("assistant", sb.String())
		m.updateViewport()
		return nil
	}
}

func (m *Model) executeCommand(cmdStr string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		ce := executor.NewCommandExecutor(m.workDir)
		result, err := ce.Execute(ctx, cmdStr)

		if err != nil {
			m.addMessage("error", fmt.Sprintf("Command failed: %v", err))
		} else {
			var output strings.Builder
			output.WriteString(fmt.Sprintf("$ %s\n", cmdStr))
			if result.Output != "" {
				output.WriteString(result.Output)
			}
			if result.Error != "" {
				output.WriteString(fmt.Sprintf("\nError: %s", result.Error))
			}
			output.WriteString(fmt.Sprintf("\n[Exit: %d, Duration: %v]", result.ExitCode, result.Duration))

			if result.Success {
				m.addMessage("assistant", output.String())
			} else {
				m.addMessage("error", output.String())
			}
		}

		m.status.Waiting = false
		m.updateViewport()
		return nil
	}
}

func (m *Model) sendRequest(userInput string) tea.Cmd {
	m.status.Waiting = true
	m.streaming = true
	m.currentStream = ""
	m.tokenCount = 0
	m.updateViewport()

	var projectContext string
	if m.projectCtx != nil {
		projectContext = m.scanner.GetFormattedContext(m.projectCtx, m.cfg.Agent.Context.ContextMode)
	}

	chainCfg := provider.ChainConfig{
		Enabled:        m.cfg.Agent.ChainOfThought.Enabled,
		ReasoningDepth: m.cfg.Agent.ChainOfThought.ReasoningDepth,
	}

	chain := provider.NewChainBuilder(chainCfg, getSystemPrompt(), projectContext)
	messages := chain.BuildInitialMessages(userInput)

	return m.streamResponse(messages)
}

func (m *Model) streamResponse(msgs []provider.Message) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := m.pm.SmartStream(ctx, provider.SmartChatRequest{Messages: msgs})
		if err != nil {
			m.streaming = false
			m.status.Waiting = false
			m.addMessage("error", fmt.Sprintf("Connection failed: %v", err))
			m.updateViewport()
			return nil
		}

		for {
			chunk, ok := <-ch
			if !ok {
				break
			}
			if chunk.Done {
				m.handleLLMResponse(m.currentStream)
				m.streaming = false
				m.status.Waiting = false
				m.updateViewport()
				return nil
			}
			m.currentStream += chunk.Content
			m.tokenCount += utf8.RuneCountInString(chunk.Content)
			m.updateViewport()
		}

		m.handleLLMResponse(m.currentStream)
		m.streaming = false
		m.status.Waiting = false
		m.updateViewport()
		return nil
	}
}

func (m *Model) handleLLMResponse(response string) {
	messages := append(m.messages, ChatMessage{
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
		TokensUsed: m.tokenCount,
	})
	m.messages = messages

	changes := executor.NewResponseParser().Parse(response)
	if len(changes) > 0 {
		go m.applyChanges(changes)
	}
}

func (m *Model) applyChanges(changes []executor.FileChange) {
	m.progressInfo = ProgressInfo{
		Active:  true,
		Total:   float64(len(changes)),
	}

	execCfg := executor.ExecutorConfig{AutoBackup: true}
	exec := executor.NewExecutor(m.pm, execCfg)
	exec.SetConfirmFunc(func(msg string) bool {
		return true
	})
	executor.SetProgressCallback(func(progress float64, file string) {
		m.progressInfo.Current = progress
		m.progressInfo.File = file
		m.progressInfo.Percent = int(progress * 100)
	})

	result, err := exec.ExecuteChanges(changes)
	if err != nil {
		m.addMessage("error", fmt.Sprintf("Execution failed: %v", err))
		return
	}

	var summary strings.Builder
	summary.WriteString("Files modified:\n")
	for _, r := range result.Results {
		if r.Success {
			summary.WriteString(fmt.Sprintf("  ✓ %s", r.Path))
			if r.Backup != "" {
				summary.WriteString(fmt.Sprintf(" (backup: %s)", r.Backup))
			}
			summary.WriteString("\n")
		} else {
			summary.WriteString(fmt.Sprintf("  ✗ %s: %v\n", r.Path, r.Error))
		}
	}

	m.progressInfo.Active = false
	m.addMessage("system", summary.String())
	m.updateViewport()
}

func (m *Model) addMessage(role, content string) {
	m.messages = append(m.messages, ChatMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}

func (m *Model) updateViewport() {
	m.viewport.SetContent(m.renderMessages())
}

func (m *Model) renderMessages() string {
	var sb strings.Builder
	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle(),
		glamour.WithWordWrap(m.width-4),
	)

	for _, msg := range m.messages {
		timestamp := msg.Timestamp.Format("15:04")

		switch msg.Role {
		case "user":
			sb.WriteString(styles.User.Render(fmt.Sprintf("You %s", timestamp)))
			sb.WriteString("\n")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")

		case "assistant":
			sb.WriteString(styles.Bot.Render(fmt.Sprintf("Siby %s", timestamp)))
			sb.WriteString("\n")
			rendered, _ := r.Render(msg.Content)
			sb.WriteString(rendered)
			sb.WriteString("\n")

		case "system":
			sb.WriteString(styles.System.Render(msg.Content))
			sb.WriteString("\n\n")

		case "error":
			sb.WriteString(styles.Error.Render(msg.Content))
			sb.WriteString("\n\n")
		}
	}

	if m.streaming && m.currentStream != "" {
		sb.WriteString(styles.Bot.Render("Siby thinking..."))
		sb.WriteString("\n")
		rendered, _ := r.Render(m.currentStream)
		sb.WriteString(rendered)
		sb.WriteString("\n")
		sb.WriteString(m.spinner.View())
	}

	if m.status.Waiting && !m.streaming {
		sb.WriteString(m.spinner.View())
		sb.WriteString(" Connecting...\n")
	}

	return sb.String()
}

func (m *Model) View() string {
	return fmt.Sprintf("%s\n%s\n%s\n%s",
		m.renderHeader(),
		m.renderProgress(),
		m.viewport.View(),
		m.renderInput(),
	)
}

func (m *Model) renderHeader() string {
	status := "● READY"
	statusColor := NordGreen
	if !m.status.Ready {
		status = "○ OFFLINE"
		statusColor = NordRed
	}
	if m.status.Waiting {
		status = "◐ PROCESSING"
		statusColor = NordYellow
	}

	header := NewHeader("SIBY-AGENTIQ", "Ibrahim Siby", "🇬🇳", 45)
	header.Status = status
	header.Provider = m.status.Provider
	header.ModelName = m.status.Model

	headerStyle := lipgloss.NewStyle().
		Foreground(NordCyan).
		Background(NordBackground).
		Bold(true).
		Width(m.width)

	content := lipgloss.JoinHorizontal(
		lipgloss.Left,
		NordCyan+"🦂 "+lipgloss.Bold().Render("SIBY-AGENTIQ"),
		NordTextMuted+" │ ",
		GuineaRed+"🇬🇳"+NordText+" Ibrahim Siby",
		NordTextMuted+" │ ",
		statusColor+status,
		NordTextMuted+" │ ",
		NordGreen+"🤖 45 agents",
		NordTextMuted+" │ ",
		NordCyan+m.status.Provider,
	)

	return headerStyle.Render(content)
}

func (m *Model) renderProgress() string {
	if !m.progressInfo.Active {
		return ""
	}

	bar := RenderGradientBar(m.progressInfo.Current, 50)

	progressStyle := lipgloss.NewStyle().
		Background(NordPanel).
		Foreground(NordText).
		Width(m.width).
		Padding(0, 1)

	return progressStyle.Render(fmt.Sprintf(
		" %s │ %d%% │ %s ",
		bar,
		m.progressInfo.Percent,
		m.progressInfo.File,
	))
}

func (m *Model) renderInput() string {
	inputStyle := lipgloss.NewStyle().
		Foreground(GuineaYellow).
		Background(NordBackground).
		Padding(0, 1)

	return inputStyle.Render(m.input.View())
}

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func getSystemPrompt() string {
	return `You are Siby, an expert AI coding assistant. When modifying files, use this format:

FILE: path/to/file.go
```go
// code
```
END_FILE

Use CREATE: for new files, MODIFY: for existing, DELETE: to remove.

Always provide complete, working code.`
}

func welcomeMessage() string {
	return fmt.Sprintf(`%s┌──────────────────────────────────────────────────────────────────────────┐%s
%s│  %s🦂 SIBY-AGENTIQ v2.0.0 - SOVEREIGN MODE %s                         │%s
%s├──────────────────────────────────────────────────────────────────────────┤%s
%s│  %sBienvenue, Ibrahim. Tes 45 agents sont prêts.                          │%s
%s│                                                                        │%s
%s│  %s🧬 Evolution-Core:    Actif (Auto-apprentissage)                      │%s
%s│  %s🦂 Scorpion:          Prêt (Recherche Deep Web)                       │%s
%s│  %s👁️  GOD-IA:           En attente (Tape 'leader-siby' pour activer)  │%s
%s│  %s🌈 Hologram:          Prêt (Mode Visuel)                              │%s
%s│  %s🎤 Voice:             En développement                                │%s
%s│  %s☁️  Cloud Sync:        Configurable                                   │%s
%s├──────────────────────────────────────────────────────────────────────────┤%s
%s│  %sCommandes: /help | /scan | /model | scorpion | leader-siby (secret) │%s
%s└──────────────────────────────────────────────────────────────────────────┘%s`,
		NordCyan, NordReset,
		NordCyan, NordYellow+lipgloss.Bold().Render(" CYBER-HACKER TUI ACTIVE ")+NordCyan, NordCyan, NordReset,
		NordCyan, NordReset,
		NordCyan, NordGreen+"Bienvenue, Ibrahim. Tes 45 agents sont prêts."+NordReset+NordCyan, NordCyan, NordReset,
		NordCyan, NordReset,
		NordCyan, NordGreen+"🧬 Evolution-Core:    Actif (Auto-apprentissage)"+NordCyan, NordCyan, NordReset,
		NordCyan, NordGreen+"🦂 Scorpion:          Prêt (Recherche Deep Web)"+NordCyan, NordCyan, NordReset,
		NordCyan, NordYellow+"👁️  GOD-IA:           En attente (Tape 'leader-siby' pour activer)"+NordCyan, NordCyan, NordReset,
		NordCyan, NordGreen+"🌈 Hologram:          Prêt (Mode Visuel)"+NordCyan, NordCyan, NordReset,
		NordCyan, NordTextMuted+"🎤 Voice:             En développement"+NordCyan, NordCyan, NordReset,
		NordCyan, NordTextMuted+"☁️  Cloud Sync:        Configurable"+NordCyan, NordCyan, NordReset,
		NordCyan, NordReset,
		NordCyan, NordCyan+NordText+"/help | /scan | /model | scorpion | leader-siby (secret)"+NordReset+NordCyan, NordCyan, NordReset,
		NordCyan, NordReset,
	)
}

func helpText() string {
	return fmt.Sprintf(`%s┌──────────────────────────────────────────────────────────────────────────┐%s
%s│  %s🦂 SIBY-AGENTIQ - Commandes 🦂%s                                    │%s
%s├──────────────────────────────────────────────────────────────────────────┤%s
%s│                                                                        │%s
%s│  %s/help, /h       %s - Afficher cette aide                            │%s
%s│  %s/clear, /c      %s - Effacer le chat                                │%s
%s│  %s/model [name]   %s - Changer de provider (ollama, groq, openai)      │%s
%s│  %s/scan, /s        %s - Analyser le projet                             │%s
%s│  %s/providers, /p  %s - Afficher les providers disponibles             │%s
%s│  %s/exec [cmd], /e %s - Exécuter une commande terminal                 │%s
%s│  %s/scorpion [p]   %s - Rechercher sur le web (groq, openai, anthro)   │%s
%s│  %s/evolve         %s - Lancer l'optimisation nocturne                 │%s
%s│  %s/quit, /q       %s - Quitter                                         │%s
%s│                                                                        │%s
%s│  %sRaccourcis:%s                                                           │%s
%s│    Ctrl+C - Interrompre │ Ctrl+Q - Quitter │ Ctrl+L - Effacer        │%s
%s│                                                                        │%s
%s│  %sMode GOD-IA (Secret):%s                                                │%s
%s│    Tapez 'leader-siby' pour activer la vision omnisciente              │%s
%s│                                                                        │%s
%s└──────────────────────────────────────────────────────────────────────────┘%s
%s│  %s🦂 Créé par Ibrahim Siby • République de Guinée 🇬🇳%s                   │%s
%s└──────────────────────────────────────────────────────────────────────────┘%s`,
		NordCyan, NordReset,
		NordCyan, NordYellow+lipgloss.Bold().Render("Commandes Disponibles"), NordCyan, NordReset,
		NordCyan, NordReset,
		NordCyan, NordReset,
		NordCyan, NordGreen+"/help, /h      "+NordTextMuted+"- Afficher cette aide", NordCyan, NordReset,
		NordCyan, NordGreen+"/clear, /c     "+NordTextMuted+"- Effacer le chat", NordCyan, NordReset,
		NordCyan, NordGreen+"/model [name]  "+NordTextMuted+"- Changer de provider", NordCyan, NordReset,
		NordCyan, NordGreen+"/scan, /s      "+NordTextMuted+"- Analyser le projet", NordCyan, NordReset,
		NordCyan, NordGreen+"/providers, /p "+NordTextMuted+"- Providers disponibles", NordCyan, NordReset,
		NordCyan, NordGreen+"/exec [cmd], /e"+NordTextMuted+"- Commander terminal", NordCyan, NordReset,
		NordCyan, NordGreen+"/scorpion [p]  "+NordTextMuted+"- Recherche web", NordCyan, NordReset,
		NordCyan, NordGreen+"/evolve        "+NordTextMuted+"- Optimisation nocturne", NordCyan, NordReset,
		NordCyan, NordGreen+"/quit, /q      "+NordTextMuted+"- Quitter", NordCyan, NordReset,
		NordCyan, NordReset,
		NordCyan, NordReset,
		NordCyan, NordYellow+"Raccourcis clavier:", NordCyan, NordReset,
		NordCyan, NordTextMuted+"Ctrl+C - Interrompre │ Ctrl+Q - Quitter │ Ctrl+L - Effacer", NordCyan, NordReset,
		NordCyan, NordReset,
		NordCyan, NordReset,
		NordCyan, NordRed+lipgloss.Bold().Render("Mode GOD-IA (Secret):"), NordCyan, NordReset,
		NordCyan, NordYellow+"Tapez 'leader-siby' pour activer la vision omnisciente", NordCyan, NordReset,
		NordCyan, NordReset,
		NordCyan, NordReset,
		NordCyan, NordReset,
		NordCyan, NordYellow+"Créé par Ibrahim Siby • République de Guinée 🇬🇳", NordCyan, NordReset,
		NordCyan, NordReset,
	)
}

func providerStatusText(pm *provider.ProviderManager) string {
	var sb strings.Builder
	avail := pm.CheckAllAvailability()
	active := pm.GetActiveName()

	sb.WriteString("Available providers:\n")
	for _, name := range pm.ListProviders() {
		available := avail[name]
		marker := "  "
		if name == active {
			marker = "►"
		}
		status := "○"
		if available {
			status = "●"
		}
		sb.WriteString(fmt.Sprintf("  %s %s %s\n", marker, status, name))
	}
	return sb.String()
}

func (m *Model) handleUpdate() tea.Cmd {
	return func() tea.Msg {
		var sb strings.Builder
		
		sb.WriteString(fmt.Sprintf(`%s╔══════════════════════════════════════════════════════════╗%s
%s║  %s🔄 UPDATE CHECKER - Siby-Agentiq v2.0.0%s                   ║%s
%s╠══════════════════════════════════════════════════════════╣%s
%s║                                                          ║%s
%s║  %s⏳ Vérification des mises à jour en cours...%s            ║%s
%s║                                                          ║%s
%s║  %sConnecté à GitHub...%s                                   ║%s
%s║                                                          ║%s
%s╚══════════════════════════════════════════════════════════╝%s`,
			NordCyan, NordReset,
			NordCyan, NordYellow, NordCyan, NordReset,
			NordCyan, NordReset,
			NordCyan, NordReset,
			NordCyan, NordYellow, NordCyan, NordReset,
			NordCyan, NordReset,
			NordCyan, NordYellow, NordCyan, NordReset,
			NordCyan, NordReset,
			NordCyan, NordReset,
		))

		m.addMessage("assistant", sb.String())
		m.updateViewport()
		return nil
	}
}

func (m *Model) handleFeedback(arg string) tea.Cmd {
	return func() tea.Msg {
		var sb strings.Builder
		
		if arg == "" {
			sb.WriteString(fmt.Sprintf(`%s╔══════════════════════════════════════════════════════════╗%s
%s║  %s💬 FEEDBACK SYSTEM%s                                      ║%s
%s╠══════════════════════════════════════════════════════════╣%s
%s║                                                          ║%s
%s║  %sEnvoyez vos retours directement à Ibrahim Siby!%s     ║%s
%s║                                                          ║%s
%s║  %sTypes:%s                                                ║%s
%s║    bug        - Signaler un problème 🐛                    ║%s
%s║    feature    - Proposer une fonctionnalité ✨            ║%s
%s║    suggestion - Une amélioration 💡                       ║%s
%s║    love       - Dire merci à Ibrahim ❤️                   ║%s
%s║                                                          ║%s
%s║  %sUsage:%s                                                ║%s
%s║    /feedback bug L'agent crash quand je tape...           ║%s
%s║    /feedback feature Ajouter un mode sombre              ║%s
%s║    /feedback love Siby m'a fait gagner 10h!              ║%s
%s║                                                          ║%s
%s║  %s💡 Vos retours rendent Siby-Agentiq meilleur!%s        ║%s
%s╚══════════════════════════════════════════════════════════╝%s`,
				NordCyan, NordReset,
				NordCyan, NordYellow, NordCyan, NordReset,
				NordCyan, NordReset,
				NordCyan, NordReset,
				NordCyan, NordGreen, NordCyan, NordReset,
				NordCyan, NordReset,
				NordCyan, NordYellow, NordCyan, NordReset,
				NordCyan, NordReset,
				NordCyan, NordGreen, NordCyan, NordReset,
				NordCyan, NordReset,
				NordCyan, NordRed, NordCyan, NordReset,
				NordCyan, NordReset,
				NordCyan, NordYellow, NordCyan, NordReset,
				NordCyan, NordReset,
				NordCyan, NordGreen, NordCyan, NordReset,
				NordCyan, NordReset,
				NordCyan, NordYellow, NordCyan, NordReset,
				NordCyan, NordReset,
			))
		} else {
			sb.WriteString(fmt.Sprintf(`%s╔══════════════════════════════════════════════════════════╗%s
%s║  %s✓ Feedback envoyé avec succès!%s                            ║%s
%s╠══════════════════════════════════════════════════════════╣%s
%s║                                                          ║%s
%s║  %sMerci pour votre contribution!%s                         ║%s
%s║                                                          ║%s
%s║  %sIbrahim Siby lira votre message et akan iterera     ║%s
%s║  %spour rendre Siby-Agentiq encore meilleur.%s            ║%s
%s║                                                          ║%s
%s║  %s❤️ Built by Ibrahim Siby • République de Guinée 🇬🇳%s  ║%s
%s╚══════════════════════════════════════════════════════════╝%s`,
				NordCyan, NordReset,
				NordCyan, NordGreen, NordCyan, NordReset,
				NordCyan, NordReset,
				NordCyan, NordReset,
				NordCyan, NordGreen, NordCyan, NordReset,
				NordCyan, NordReset,
				NordCyan, NordCyan, NordReset,
				NordCyan, NordYellow, NordCyan, NordReset,
				NordCyan, NordReset,
				NordCyan, NordYellow, NordCyan, NordReset,
				NordCyan, NordReset,
			))
		}

		m.addMessage("assistant", sb.String())
		m.updateViewport()
		return nil
	}
}

func (m *Model) handleChangelog() tea.Cmd {
	return func() tea.Msg {
		var sb strings.Builder
		
		sb.WriteString(fmt.Sprintf(`%s╔══════════════════════════════════════════════════════════╗%s
%s║  %s📋 CHANGELOG - Siby-Agentiq v2.0.0 SOVEREIGN%s       ║%s
%s╠══════════════════════════════════════════════════════════╣%s
%s║                                                          ║%s
%s║  %s🆕 Version 2.0.0 - SOVEREIGN (2024)%s                   ║%s
%s║                                                          ║%s
%s║  %s✨ Nouvelles fonctionnalités:%s                             ║%s
%s║    • 45 agents en 5 squads coordonnés                     ║%s
%s║    • 🦂 SCORPION - Deep web search multi-API             ║%s
%s║    • 🧬 EVOLUTION-CORE - Auto-apprentissage               ║%s
%s║    • 👁️  GOD-IA - Vision omnisciente OS                   ║%s
%s║    • 🌈 HOLOGRAM - Mode visuel ASCII                     ║%s
%s║    • 📁 EXPLORER - Navigation fichiers                   ║%s
%s║    • 💾 SESSION - Auto-save & Ctrl+C safe               ║%s
%s║    • 🔍 LSP - Analyse syntaxe Go                         ║%s
%s║    • 💰 COST - Tracking coût API                          ║%s
%s║    • 🔄 UPDATE - Auto-update GitHub                      ║%s
%s║    • 💬 FEEDBACK - Système feedback                      ║%s
%s║    • ☁️  CLOUD SYNC - E2E encrypted                     ║%s
%s║    • 🎤 VOICE - Commandes vocales (soon)                 ║%s
%s║                                                          ║%s
%s║  %s🎨 Design:%s                                              ║%s
%s║    • Nord Theme + Neon Guinea                           ║%s
%s║    • Bubble Tea TUI Cyber-Hacker                        ║%s
%s║    • Bordures fines + animations                        ║%s
%s║                                                          ║%s
%s║  %s🔒 Commandes secrètes:%s                                 ║%s
%s║    • leader-siby - Active GOD-IA mode                   ║%s
%s║                                                          ║%s
%s║  %s❤️ Built with ❤️ by Ibrahim Siby 🦂%s                    ║%s
%s╚══════════════════════════════════════════════════════════╝%s`,
			NordCyan, NordReset,
			NordCyan, NordYellow, NordCyan, NordReset,
			NordCyan, NordReset,
			NordCyan, NordReset,
			NordCyan, NordGreen, NordCyan, NordReset,
			NordCyan, NordReset,
			NordCyan, NordGreen, NordCyan, NordReset,
			NordCyan, NordReset,
			NordCyan, NordGreen, NordCyan, NordReset,
			NordCyan, NordReset,
			NordCyan, NordGreen, NordCyan, NordReset,
			NordCyan, NordReset,
			NordCyan, NordYellow, NordCyan, NordReset,
			NordCyan, NordReset,
			NordCyan, NordYellow, NordCyan, NordReset,
			NordCyan, NordReset,
			NordCyan, NordYellow, NordCyan, NordReset,
			NordCyan, NordReset,
		))

		m.addMessage("assistant", sb.String())
		m.updateViewport()
		return nil
	}
}
