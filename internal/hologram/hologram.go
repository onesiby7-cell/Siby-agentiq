package hologram

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	HoloCyan     = "\033[96m"
	HoloMagenta  = "\033[95m"
	HoloGold     = "\033[93m"
	HoloGreen    = "\033[92m"
	HoloRed      = "\033[91m"
	HoloBlue     = "\033[94m"
	HoloReset    = "\033[0m"
	HoloBold     = "\033[1m"
	HoloBlink    = "\033[5m"
)

type HologramMode struct {
	enabled      bool
	theme        string
	animations   map[string]*Animation
	activeEffects []Effect
	mu           sync.RWMutex
}

type Animation struct {
	Name    string
	Frames  []string
	FPS     int
	Current int
}

type Effect struct {
	Type    EffectType
	Content string
	Duration time.Duration
}

type EffectType string

const (
	EffectMatrix   EffectType = "matrix"
	EffectGlitch   EffectType = "glitch"
	EffectPulse    EffectType = "pulse"
	EffectScanline EffectType = "scanline"
	EffectHologram EffectType = "hologram"
)

type HologramTheme struct {
	Name      string
	Primary   string
	Secondary string
	Accent    string
	Background string
}

var themes = map[string]HologramTheme{
	"cyberpunk": {
		Name:      "Cyberpunk",
		Primary:   "\033[96m",
		Secondary: "\033[95m",
		Accent:    "\033[93m",
		Background: "\033[40m",
	},
	"terminal": {
		Name:      "Terminal",
		Primary:   "\033[92m",
		Secondary: "\033[32m",
		Accent:    "\033[36m",
		Background: "\033[0m",
	},
	"neon": {
		Name:      "Neon",
		Primary:   "\033[95m",
		Secondary: "\033[96m",
		Accent:    "\033[35m",
		Background: "\033[45m",
	},
}

func NewHologramMode() *HologramMode {
	h := &HologramMode{
		enabled: false,
		theme:   "cyberpunk",
		animations: make(map[string]*Animation),
		activeEffects: make([]Effect, 0),
	}
	h.initAnimations()
	return h
}

func (h *HologramMode) initAnimations() {
	h.animations["loading"] = &Animation{
		Name: "loading",
		Frames: []string{
			"[■□□□□□□□□□] 10%",
			"[■■□□□□□□□□] 20%",
			"[■■■□□□□□□□] 30%",
			"[■■■■□□□□□□] 40%",
			"[■■■■■□□□□□] 50%",
			"[■■■■■■□□□□] 60%",
			"[■■■■■■■□□□] 70%",
			"[■■■■■■■■□□] 80%",
			"[■■■■■■■■■□] 90%",
			"[■■■■■■■■■■] 100%",
		},
		FPS: 10,
	}

	h.animations["scorpion"] = &Animation{
		Name: "scorpion",
		Frames: []string{
			"🦂",
			"🦂",
			"🪳",
			"🪲",
			"🐛",
			"🕷️",
			"🦂",
		},
		FPS: 5,
	}

	h.animations["pulse"] = &Animation{
		Name: "pulse",
		Frames: []string{
			"●",
			"◐",
			"○",
			"◑",
		},
		FPS: 8,
	}
}

func (h *HologramMode) Enable() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = true
}

func (h *HologramMode) Disable() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = false
}

func (h *HologramMode) IsEnabled() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.enabled
}

func (h *HologramMode) SetTheme(theme string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := themes[theme]; ok {
		h.theme = theme
	}
}

func (h *HologramMode) AddEffect(effect Effect) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeEffects = append(h.activeEffects, effect)
}

func (h *HologramMode) Render(content string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.enabled {
		return content
	}

	theme := themes[h.theme]
	var sb strings.Builder

	sb.WriteString(h.applyEffect(EffectGlitch, 10))
	sb.WriteString(h.renderFrame("scorpion"))
	sb.WriteString(HoloReset)

	return h.wrapInHologram(content, theme)
}

func (h *HologramMode) wrapInHologram(content string, theme HologramTheme) string {
	var sb strings.Builder

	width := 78
	border := strings.Repeat("═", width)

	sb.WriteString(fmt.Sprintf("\n%s%s%s\n", theme.Primary, border, HoloReset))
	sb.WriteString(fmt.Sprintf("%s║%s 🦂 HOLOGRAM MODE ACTIVE 🦂 %s║%s\n", 
		theme.Primary, theme.Accent, theme.Primary, HoloReset))
	sb.WriteString(fmt.Sprintf("%s║%s Theme: %-15s │ Powered by Ibrahim Siby %s║%s\n", 
		theme.Primary, theme.Secondary, h.theme, theme.Primary, HoloReset))
	sb.WriteString(fmt.Sprintf("%s%s%s\n", theme.Primary, border, HoloReset))

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		padding := width - len(stripANSI(line))
		if padding < 0 {
			padding = 0
		}
		sb.WriteString(fmt.Sprintf("%s║%s %s%s ║%s\n", 
			theme.Primary, HoloReset, line, strings.Repeat(" ", padding), HoloReset))
	}

	sb.WriteString(fmt.Sprintf("%s%s%s\n", theme.Primary, border, HoloReset))

	return sb.String()
}

func (h *HologramMode) renderFrame(name string) string {
	if anim, ok := h.animations[name]; ok {
		anim.Current = (anim.Current + 1) % len(anim.Frames)
		return anim.Frames[anim.Current]
	}
	return ""
}

func (h *HologramMode) applyEffect(effect EffectType, intensity int) string {
	switch effect {
	case EffectGlitch:
		return h.glitchEffect(intensity)
	case EffectMatrix:
		return h.matrixEffect()
	case EffectScanline:
		return h.scanlineEffect()
	case EffectPulse:
		return h.pulseEffect()
	default:
		return ""
	}
}

func (h *HologramMode) glitchEffect(intensity int) string {
	if intensity > 7 {
		return fmt.Sprintf("%s%s%s", HoloRed, HoloBlink, "▓▒░")
	}
	return ""
}

func (h *HologramMode) matrixEffect() string {
	return fmt.Sprintf("%s%s%s", HoloGreen, HoloBold, "\n▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄\n")
}

func (h *HologramMode) scanlineEffect() string {
	return fmt.Sprintf("%s%s", HoloCyan, "▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄")
}

func (h *HologramMode) pulseEffect() string {
	return fmt.Sprintf("%s%s", HoloMagenta, "●◐○◑")
}

func (h *HologramMode) RenderAgentStatus(agents map[string]AgentStatus) string {
	theme := themes[h.theme]
	var sb strings.Builder

	width := 60
	sb.WriteString(fmt.Sprintf("\n%s╔%s╗\n", theme.Primary, strings.Repeat("═", width-2)))
	sb.WriteString(fmt.Sprintf("%s║%s 🦂 SIBY-AGENTIQ SQUAD STATUS %s║%s\n", 
		theme.Primary, theme.Accent, theme.Primary, HoloReset))
	sb.WriteString(fmt.Sprintf("%s╠%s╣\n", theme.Primary, strings.Repeat("═", width-2)))

	for name, status := range agents {
		color := HoloGreen
		if status.State == "busy" {
			color = HoloGold
		} else if status.State == "error" {
			color = HoloRed
		}

		line := fmt.Sprintf("  %s: %s%s", name, color, status.State)
		padding := width - 4 - len(stripANSI(line))
		sb.WriteString(fmt.Sprintf("%s║%s%s%s ║%s\n", theme.Primary, HoloReset, line, strings.Repeat(" ", padding), HoloReset))
	}

	sb.WriteString(fmt.Sprintf("%s╚%s╝\n", theme.Primary, strings.Repeat("═", width-2)))
	sb.WriteString(fmt.Sprintf("%s  🦂 Built with ❤️ by Ibrahim Siby 🦂%s\n", HoloGold, HoloReset))

	return sb.String()
}

func (h *HologramMode) RenderWelcome() string {
	theme := themes[h.theme]
	return fmt.Sprintf(`
%s
%s╔═══════════════════════════════════════════════════════════════════╗
║                                                                   ║
║   %s██╗    ██╗███████╗███╗   ██╗██╗  ██╗    ██╗     ██╗███████╗   ║
║   %s██║    ██║██╔════╝████╗  ██║██║ ██╔╝    ██║     ██║██╔════╝   ║
║   %s██║ █╗ ██║█████╗  ██╔██╗ ██║█████╔╝     ██║     ██║███████╗   ║
║   %s██║███╗██║██╔══╝  ██║╚██╗██║██╔═██╗     ██║     ██║╚════██║   ║
║   %s╚███╔███╔╝███████╗██║ ╚████║██║  ██╗    ███████╗██║███████║   ║
║    %s╚══╝╚══╝ ╚══════╝╚═╝  ╚═══╝╚═╝  ╚═╝    ╚══════╝╚═╝╚══════╝   ║
║                                                                   ║
║   %s┌─────────────────────────────────────────────────────────┐     ║
║   %s│          The Last Agent You Will Ever Need              │     ║
║   %s└─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║   %s🌟 SCORPION: Deep Web Search                                  ║
║   %s🧬 EVOLUTION: Self-Learning Engine                             ║
║   %s👁️  GOD-IA: Type 'leader-siby' to activate (secret)          ║
║   %s🌈 HOLOGRAM: Visual Mode Active                                ║
║   %s🎤 VOICE: Say "Hey Siby" (coming soon)                         ║
║   %s☁️ CLOUD: E2E Encrypted Sync (coming soon)                    ║
║                                                                   ║
║   %s═══════════════════════════════════════════════════════════════ ║
║   %s  🦂 Built with ❤️ by Ibrahim Siby • République de Guinée 🇬🇳   ║
║   %s═══════════════════════════════════════════════════════════════ ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
%s`, 
		theme.Primary, theme.Primary,
		theme.Accent, theme.Accent, theme.Accent, theme.Accent, theme.Accent, theme.Accent,
		theme.Secondary, theme.Secondary, theme.Secondary,
		theme.Primary,
		theme.Secondary, theme.Primary, theme.Primary,
		HoloGold, HoloGold,
		theme.Primary,
		HoloReset)
}

type AgentStatus struct {
	Name  string
	State string
}

func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape && r == 'm' {
			inEscape = false
			continue
		}
		if !inEscape {
			result.WriteRune(r)
		}
	}
	return result.String()
}
