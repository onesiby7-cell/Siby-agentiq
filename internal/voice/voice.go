package voice

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	VoiceReset   = "\033[0m"
	VoiceCyan    = "\033[96m"
	VoiceGreen   = "\033[92m"
	VoiceYellow  = "\033[93m"
	VoiceRed     = "\033[91m"
)

type VoiceCommand struct {
	Command    string
	Action     string
	Parameters map[string]string
}

type VoiceEngine struct {
	enabled    bool
	wakeWord   string
	listening  bool
	mu         sync.RWMutex
	commands   map[string]*VoiceCommand
	onCommand  func(*VoiceCommand) error
}

type VoiceConfig struct {
	WakeWord      string
	Language      string
	Confidence    float64
	Mode          string
}

var defaultConfig = &VoiceConfig{
	WakeWord:   "hey siby",
	Language:   "en-US",
	Confidence: 0.7,
	Mode:       "cli",
}

func NewVoiceEngine() *VoiceEngine {
	ve := &VoiceEngine{
		enabled:  false,
		wakeWord: defaultConfig.WakeWord,
		commands: make(map[string]*VoiceCommand),
		listening: false,
	}
	ve.initCommands()
	return ve
}

func (ve *VoiceEngine) initCommands() {
	commands := []struct {
		phrases []string
		action  string
	}{
		{
			phrases: []string{"build the project", "build project", "compile", "make"},
			action:  "build",
		},
		{
			phrases: []string{"run tests", "test", "run test"},
			action:  "test",
		},
		{
			phrases: []string{"deploy", "push to production", "release"},
			action:  "deploy",
		},
		{
			phrases: []string{"scan project", "analyze", "check code"},
			action:  "scan",
		},
		{
			phrases: []string{"help", "commands", "what can you do"},
			action:  "help",
		},
		{
			phrases: []string{"search for", "find information", "lookup"},
			action:  "search",
		},
		{
			phrases: []string{"create file", "new file", "add file"},
			action:  "create",
		},
		{
			phrases: []string{"fix bug", "debug", "error"},
			action:  "fix",
		},
		{
			phrases: []string{"god mode", "activate god", "leader siby"},
			action:  "godmode",
		},
		{
			phrases: []string{"exit", "quit", "bye"},
			action:  "exit",
		},
	}

	for _, cmd := range commands {
		for _, phrase := range cmd.phrases {
			ve.commands[strings.ToLower(phrase)] = &VoiceCommand{
				Command: phrase,
				Action:  cmd.action,
			}
		}
	}
}

func (ve *VoiceEngine) Enable() error {
	ve.mu.Lock()
	defer ve.mu.Unlock()

	if ve.enabled {
		return nil
	}

	ve.enabled = true
	return nil
}

func (ve *VoiceEngine) Disable() {
	ve.mu.Lock()
	defer ve.mu.Unlock()
	ve.enabled = false
	ve.listening = false
}

func (ve *VoiceEngine) IsEnabled() bool {
	ve.mu.RLock()
	defer ve.mu.RUnlock()
	return ve.enabled
}

func (ve *VoiceEngine) SetWakeWord(word string) {
	ve.mu.Lock()
	defer ve.mu.Unlock()
	ve.wakeWord = strings.ToLower(word)
}

func (ve *VoiceEngine) StartListening(ctx context.Context) error {
	ve.mu.Lock()
	if !ve.enabled {
		ve.mu.Unlock()
		return fmt.Errorf("voice engine not enabled")
	}
	ve.listening = true
	ve.mu.Unlock()

	go ve.listenLoop(ctx)

	return nil
}

func (ve *VoiceEngine) listenLoop(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ve.mu.Lock()
			ve.listening = false
			ve.mu.Unlock()
			return
		default:
			<-ticker.C
		}
	}
}

func (ve *VoiceEngine) IsListening() bool {
	ve.mu.RLock()
	defer ve.mu.RUnlock()
	return ve.listening
}

func (ve *VoiceEngine) SetCommandHandler(handler func(*VoiceCommand) error) {
	ve.mu.Lock()
	defer ve.mu.Unlock()
	ve.onCommand = handler
}

func (ve *VoiceEngine) ParseCommand(input string) (*VoiceCommand, bool) {
	input = strings.ToLower(strings.TrimSpace(input))

	if !strings.Contains(input, ve.wakeWord) && !strings.HasPrefix(input, ve.wakeWord) {
		return nil, false
	}

	cleaned := strings.TrimPrefix(input, ve.wakeWord)
	cleaned = strings.TrimPrefix(cleaned, ",")
	cleaned = strings.TrimPrefix(cleaned, " ")
	cleaned = strings.TrimPrefix(cleaned, ":")

	if cmd, ok := ve.commands[cleaned]; ok {
		return cmd, true
	}

	for phrase, cmd := range ve.commands {
		if strings.Contains(cleaned, phrase) {
			params := ve.extractParameters(cleaned, phrase)
			cmdCopy := *cmd
			cmdCopy.Parameters = params
			return &cmdCopy, true
		}
	}

	return &VoiceCommand{
		Command:    input,
		Action:    "unknown",
		Parameters: map[string]string{"raw": input},
	}, false
}

func (ve *VoiceEngine) extractParameters(input, command string) map[string]string {
	params := make(map[string]string)

	after := strings.TrimPrefix(input, command)
	after = strings.TrimSpace(after)

	if after != "" {
		params["target"] = after
	}

	return params
}

func (ve *VoiceEngine) SimulateVoiceCommand(text string) (*VoiceCommand, error) {
	ve.mu.RLock()
	handler := ve.onCommand
	ve.mu.RUnlock()

	cmd, found := ve.ParseCommand(text)

	if !found && cmd.Action == "unknown" {
		fmt.Printf("%s🎤 [VOICE] Command not recognized: %s%s\n", VoiceYellow, text, VoiceReset)
		fmt.Printf("%s🎤 [VOICE] Try saying: "hey siby, help"%s\n", VoiceYellow, VoiceReset)
		return cmd, fmt.Errorf("command not recognized")
	}

	if handler != nil {
		if err := handler(cmd); err != nil {
			return cmd, err
		}
	}

	return cmd, nil
}

func (ve *VoiceEngine) RenderVoiceStatus() string {
	var sb strings.Builder

	status := VoiceRed + "INACTIVE"
	if ve.enabled {
		status = VoiceGreen + "ACTIVE"
	}

	listening := VoiceRed + "NO"
	if ve.listening {
		listening = VoiceGreen + "YES"
	}

	sb.WriteString(fmt.Sprintf("\n%s🎤 VOICE ENGINE STATUS%s\n", VoiceCyan, VoiceReset))
	sb.WriteString(fmt.Sprintf("  Wake Word:    %s%s%s\n", VoiceYellow, ve.wakeWord, VoiceReset))
	sb.WriteString(fmt.Sprintf("  Engine:      %s\n", status))
	sb.WriteString(fmt.Sprintf("  Listening:    %s\n", listening))
	sb.WriteString(fmt.Sprintf("  Commands:     %d registered\n", len(ve.commands)))
	sb.WriteString(fmt.Sprintf("\n%sAvailable Commands:%s\n", VoiceCyan, VoiceReset))
	sb.WriteString(fmt.Sprintf("  • \"Hey Siby, build the project\"\n"))
	sb.WriteString(fmt.Sprintf("  • \"Hey Siby, run tests\"\n"))
	sb.WriteString(fmt.Sprintf("  • \"Hey Siby, deploy\"\n"))
	sb.WriteString(fmt.Sprintf("  • \"Hey Siby, scan project\"\n"))
	sb.WriteString(fmt.Sprintf("  • \"Hey Siby, god mode\"\n"))
	sb.WriteString(fmt.Sprintf("\n%s🦂 Powered by Ibrahim Siby 🦂%s\n", VoiceYellow, VoiceReset))

	return sb.String()
}

func (ve *VoiceEngine) GetSupportedLanguages() []string {
	return []string{
		"en-US",
		"en-GB",
		"fr-FR",
		"es-ES",
	}
}

type VoiceCapabilities struct {
	WakeWordDetection bool
	SpeechRecognition bool
	TextToSpeech      bool
	NoiseCancellation bool
	MultiLanguage     bool
}

func (ve *VoiceEngine) GetCapabilities() *VoiceCapabilities {
	return &VoiceCapabilities{
		WakeWordDetection: true,
		SpeechRecognition: ve.enabled,
		TextToSpeech:      false,
		NoiseCancellation: false,
		MultiLanguage:     true,
	}
}

type VoiceMemo struct {
	Text      string
	Timestamp time.Time
	Command   *VoiceCommand
}

func (ve *VoiceEngine) CreateMemo(text string, cmd *VoiceCommand) *VoiceMemo {
	return &VoiceMemo{
		Text:      text,
		Timestamp: time.Now(),
		Command:   cmd,
	}
}
