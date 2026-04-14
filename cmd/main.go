package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbletea"
	"github.com/siby-agentiq/siby-agentiq/internal/agent"
	"github.com/siby-agentiq/siby-agentiq/internal/config"
	"github.com/siby-agentiq/siby-agentiq/internal/provider"
	tui "github.com/siby-agentiq/siby-agentiq/internal/tui"
)

type appState struct {
	agent     *agent.Agent
	providerMgr *provider.ProviderManager
	model     tui.Model
	cfg       *config.Config
}

func main() {
	cfg, err := loadOrCreateConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	providerMgr := provider.NewProviderManager(cfg.Provider)
	
	activeProvider := providerMgr.GetActiveProvider()
	if activeProvider == nil {
		log.Fatal("No provider available")
	}

	availability := providerMgr.CheckAllAvailability()
	for name, available := range availability {
		status := "✓"
		if !available {
			status = "✗"
		}
		fmt.Printf("[%s] %s: %s\n", status, name, map[bool]string{true: "available", false: "unavailable"}[available])
	}

	if !availability[providerMgr.GetActiveName()] {
		fmt.Printf("Warning: Active provider '%s' is not available\n", providerMgr.GetActiveName())
	}

	agentInstance := agent.NewAgent(
		activeProvider,
		cfg.Agent,
		cfg.Reasoning,
		cfg.Agent.Context,
	)

	app := &appState{
		agent:       agentInstance,
		providerMgr: providerMgr,
		model:       tui.NewModel(cfg.TUI),
		cfg:         cfg,
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := p.Start(); err != nil {
			log.Printf("TUI error: %v", err)
		}
		cancel()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	cancel()
	fmt.Println("\nGoodbye!")
}

func loadOrCreateConfig() (*config.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(home, ".siby-agentiq")
	configPath := filepath.Join(configDir, "config.yaml")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultCfg := config.GetDefaultConfig()
		if err := defaultCfg.Save(configPath); err != nil {
			return nil, err
		}
		fmt.Printf("Created default config at %s\n", configPath)
	}

	return config.LoadConfig(configPath)
}

type appState struct {
	agent       *agent.Agent
	providerMgr *provider.ProviderManager
	model       tui.Model
	cfg         *config.Config
}

func (a *appState) Init() tea.Cmd {
	return nil
}

func (a *appState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if a.model.Input != "" && !a.model.Thinking {
				userInput := a.model.Input
				a.model.Input = ""
				a.model.Thinking = true
				
				return a.model, func() tea.Msg {
					return ProcessMessage{Input: userInput}
				}
			}
		case "up":
			if a.model.HistoryIdx > 0 {
				a.model.HistoryIdx--
				a.model.Input = a.model.History[a.model.HistoryIdx]
			}
		case "down":
			if a.model.HistoryIdx < len(a.model.History)-1 {
				a.model.HistoryIdx++
				a.model.Input = a.model.History[a.model.HistoryIdx]
			} else {
				a.model.HistoryIdx = len(a.model.History)
				a.model.Input = ""
			}
		case "ctrl+c", "esc":
			return a.model, tea.Quit
		case "ctrl+l":
			a.model.Messages = nil
		}

	case ProcessMessage:
		response, err := a.agent.ProcessRequest(context.Background(), msg.Input, getWorkingDir())
		if err != nil {
			return a.model, func() tea.Msg {
				return MessageReceived{Error: err.Error()}
			}
		}
		return a.model, func() tea.Msg {
			return MessageReceived{Response: response}
		}

	case MessageReceived:
		a.model.Thinking = false
		if msg.Error != "" {
			a.model.Messages = append(a.model.Messages, tui.Message{
				Role:    "system",
				Content: msg.Error,
				Type:    tui.MsgTypeError,
			})
		} else if msg.Response != nil {
			a.model.Messages = append(a.model.Messages, tui.Message{
				Role:    "user",
				Content: msg.Response.Output,
				Type:    tui.MsgTypeAssistant,
			})
		}
	}

	return a.model, nil
}

func (a *appState) View() string {
	var sb strings.Builder
	
	sb.WriteString("\033[1;36m╔════════════════════════════════════════════════════════════════╗\033[0m\n")
	sb.WriteString(fmt.Sprintf("\033[1;36m║\033[0m \033[1;35mSiby-Agentiq\033[0m v0.1.0                                    \033[1;36m║\033[0m\n"))
	sb.WriteString(fmt.Sprintf("\033[1;36m║\033[0m Provider: \033[32m%s\033[0m                                            \033[1;36m║\033[0m\n", a.providerMgr.GetActiveName()))
	sb.WriteString("\033[1;36m╚════════════════════════════════════════════════════════════════╝\033[0m\n\n")
	
	if len(a.model.Messages) == 0 {
		sb.WriteString("\033[2mWelcome to Siby-Agentiq! Ask me anything about your project.\033[0m\n\n")
	}
	
	for _, msg := range a.model.Messages {
		switch msg.Type {
		case tui.MsgTypeUser:
			sb.WriteString(fmt.Sprintf("\033[1;34mYou:\033[0m %s\n\n", msg.Content))
		case tui.MsgTypeAssistant:
			sb.WriteString(fmt.Sprintf("\033[1;35mSiby:\033[0m %s\n\n", msg.Content))
		case tui.MsgTypeThinking:
			sb.WriteString(fmt.Sprintf("\033[90m🤔 %s\033[0m\n\n", msg.Content))
		case tui.MsgTypePlan:
			sb.WriteString(fmt.Sprintf("\033[36m  → %s\033[0m\n", msg.Content))
		case tui.MsgTypeError:
			sb.WriteString(fmt.Sprintf("\033[31mError: %s\033[0m\n\n", msg.Content))
		}
	}
	
	if a.model.Thinking {
		sb.WriteString("\033[90mThinking...\033[0m ")
	}
	
	sb.WriteString(fmt.Sprintf("\n\033[1;35m>\033[0m %s\033[7m \033[0m", a.model.Input))
	
	return sb.String()
}

type ProcessMessage struct {
	Input string
}

type MessageReceived struct {
	Response *agent.AgentResponse
	Error    string
}

func getWorkingDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}
