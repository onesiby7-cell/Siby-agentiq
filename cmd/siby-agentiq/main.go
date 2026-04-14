package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/siby-agentiq/siby-agentiq/internal/cloud"
	"github.com/siby-agentiq/siby-agentiq/internal/config"
	"github.com/siby-agentiq/siby-agentiq/internal/core"
	"github.com/siby-agentiq/siby-agentiq/internal/discovery"
	"github.com/siby-agentiq/siby-agentiq/internal/evolution"
	"github.com/siby-agentiq/siby-agentiq/internal/explorer"
	"github.com/siby-agentiq/siby-agentiq/internal/feedback"
	"github.com/siby-agentiq/siby-agentiq/internal/godIA"
	"github.com/siby-agentiq/siby-agentiq/internal/hologram"
	"github.com/siby-agentiq/siby-agentiq/internal/lsp"
	"github.com/siby-agentiq/siby-agentiq/internal/memory"
	"github.com/siby-agentiq/siby-agentiq/internal/orchestrator"
	"github.com/siby-agentiq/siby-agentiq/internal/provider"
	"github.com/siby-agentiq/siby-agentiq/internal/scanner"
	"github.com/siby-agentiq/siby-agentiq/internal/scorpion"
	"github.com/siby-agentiq/siby-agentiq/internal/session"
	"github.com/siby-agentiq/siby-agentiq/internal/synthesis"
	"github.com/siby-agentiq/siby-agentiq/internal/ui"
	"github.com/siby-agentiq/siby-agentiq/internal/update"
	"github.com/siby-agentiq/siby-agentiq/internal/validator"
	"github.com/siby-agentiq/siby-agentiq/internal/voice"
)

const (
	ASCII_RESET  = "\033[0m"
	ASCII_CYAN  = "\033[96m"
	ASCII_GOLD  = "\033[93m"
	ASCII_GREEN = "\033[92m"
	ASCII_RED   = "\033[91m"
)

var guineaPride = `
   ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
   █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░█
   █░░▄▄▄░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░▄▄▄░░░░░░░░░█
   █░░████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░████░░░░░░░░█
   █░░████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░████░░░░░░░░█
   █░░▀▀▀▀░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░▀▀▀▀░░░░░░░░█
   █░░░░░░░░░▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄░░░░░░░░░░░█
   █░░░░░░░▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀░░░░░░░░░░░░░█
   █▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀█
   █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░█
   █░░  ██████╗ ███████╗████████╗██████╗  ██████╗ ███╗   ███╗    ░░░░░░░█
   █░░ ██╔══██╗██╔════╝╚══██╔══╝██╔══██╗██╔═══██╗████╗ ████║    ░░░░░░░█
   █░░ ██████╔╝█████╗     ██║   ██████╔╝██║   ██║██╔████╔██║    ░░░░░░░█
   █░░ ██╔══██╗██╔══╝     ██║   ██╔══██╗██║   ██║██║╚██╔╝██║    ░░░░░░░█
   █░░ ██║  ██║███████╗   ██║   ██║  ██║╚██████╔╝██║ ╚═╝ ██║    ░░░░░░░█
   █░░ ╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚═╝    ░░░░░░░█
   █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░█
   █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░█
   █▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄█
`

var splash = `
   _____ _____ ____  __  __   ___ _    _____ _  __
  |_   _| ____|  _ \|  \/  | |_ _|__|___ /| |/ /___ _   _____ _ __ 
    | | |  _| | |_) | |\/| |  | |/ __|_ \| ' // _ \ | / / _ \ '__|
    | | | |___|  _ <| |  | |  | |\__ \__) | . \  __/ | \ \  __/ |   
    |_| |_____|_| \_\_|  |_| |___|___/____|_|\_\___|_| \_|\___|_|   
                                                                        
                     The Last Agent You Will Ever Need
`

func showBanner(verbose bool) {
	if runtime.GOOS == "windows" {
		fmt.Println("\033[2J\033[H]")
	}

	if verbose {
		fmt.Print(ASCII_GOLD)
		fmt.Print(guineaPride)
		fmt.Print(ASCII_RESET)
	} else {
		fmt.Print(ASCII_CYAN)
		fmt.Print(splash)
		fmt.Print(ASCII_RESET)
	}
}

func showLoadingAnimation() {
	frames := []string{"[■□□□□□□□□□] 10%", "[■■□□□□□□□□] 20%", "[■■■□□□□□□□] 30%", 
	                   "[■■■■□□□□□□] 40%", "[■■■■■□□□□□] 50%", "[■■■■■■□□□□] 60%",
	                   "[■■■■■■■□□□] 70%", "[■■■■■■■■□□] 80%", "[■■■■■■■■■□] 90%",
	                   "[■■■■■■■■■■] 100%"}
	
	for _, frame := range frames {
		fmt.Printf("\r  Loading %s", ASCII_GREEN+frame+ASCII_RESET)
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Printf("\r  %s[COMPLETE]%s          \n", ASCII_GREEN, ASCII_RESET)
}

func printStatus(label, value string, color string) {
	fmt.Printf("  %s%-15s%s: %s\n", color, label, ASCII_RESET, value)
}

type ExtendedModel struct {
	*ui.Model
	scorpion      *scorpion.Scorpion
	godIA         *godIA.GODIA
	evolution     *evolution.EvolutionCore
	hologram      *hologram.HologramMode
	voice         *voice.VoiceEngine
	cloud         *cloud.CloudSync
	synthesizer   *synthesis.Synthesizer
	orch          *orchestrator.Orchestrator
	explorer      *explorer.FileExplorer
	session       *session.SessionManager
	interrupt     *session.InterruptHandler
	validator     *validator.APIValidator
	costTracker   *validator.CostTracker
	tokenCounter  *validator.TokenCounter
	lsp           *lsp.LSPServer
	updateChecker  *update.UpdateChecker
	feedbackSystem *feedback.FeedbackSystem
}

func main() {
	verbose := true
	godMode := false
	evolutionMode := false
	showHelp := false
	scorpionMode := false
	scorpionProvider := ""

	for _, arg := range os.Args[1:] {
		if arg == "--quiet" || arg == "-q" {
			verbose = false
		}
		if arg == "leader-siby" {
			godMode = true
		}
		if arg == "--evolve" || arg == "-e" {
			evolutionMode = true
		}
		if arg == "--help" || arg == "-h" {
			showHelp = true
		}
		if arg == "scorpion" {
			scorpionMode = true
		}
	}

	if scorpionMode && len(os.Args) > 2 {
		scorpionProvider = os.Args[len(os.Args)-2]
		providerArg := os.Args[len(os.Args)-1]
		if providerArg == "fetch" || providerArg == "check" || providerArg == "help" {
			cmd := discovery.NewScorpionCommand([]string{scorpionProvider, providerArg})
			cmd.Execute()
			return
		}
	}

	if showHelp {
		showExtendedHelp()
		return
	}

	showBanner(verbose)

	if verbose {
		fmt.Println("")
		showLoadingAnimation()
		fmt.Println("")
	}

	startTime := time.Now()

	fmt.Printf("  %s[INIT]%s Starting SIBY-AGENTIQ v%s\n", ASCII_CYAN, ASCII_RESET, core.Version)
	fmt.Printf("  %s[CREATOR]%s Ibrahim Siby • République de Guinée 🇬🇳\n", ASCII_GOLD, ASCII_RESET)

	cfg, err := config.LoadConfig("")
	if err != nil {
		cfg = config.GetDefaultConfig()
		fmt.Printf("  %s[CONFIG]%s Using default configuration\n", ASCII_GOLD, ASCII_RESET)
	}

	pm := provider.NewProviderManager(cfg.Provider)
	provider.InitAutoConfig()

	workDir, _ := os.Getwd()

	fmt.Printf("  %s[SCORPION]%s Scanning for available intelligence...\n", ASCII_GOLD, ASCII_RESET)
	discoveryResult := discovery.ScanWithScorpion()

	if verbose {
		providers := discovery.GetAllConfiguredProviders()
		if len(providers) > 0 {
			fmt.Printf("  %s[SCORPION]%s Found: %s\n", ASCII_GREEN, ASCII_RESET, strings.Join(providers, ", "))
		} else {
			fmt.Printf("%s", discoveryResult.SuggestKeyCreation())
		}
	}

	scorpionEngine := scorpion.NewScorpion(pm)
	godIAEngine := godIA.NewGODIA()
	evolutionEngine := evolution.NewEvolutionCore(workDir)
	hologramEngine := hologram.NewHologramMode()
	voiceEngine := voice.NewVoiceEngine()
	cloudEngine := cloud.NewCloudSync(workDir)
	orchEngine := orchestrator.NewOrchestrator()

	synth := synthesis.NewSynthesizer(scorpionEngine, godIAEngine, orchEngine)

	explorerEngine := explorer.NewFileExplorer(80, 30)
	explorerEngine.List(workDir)

	sessionManager, _ := session.NewSessionManager("")
	interruptHandler := session.NewInterruptHandler()

	validatorEngine := validator.NewAPIValidator()
	costTracker := validator.NewCostTracker()
	tokenCounter := validator.NewTokenCounter(128000)

	lspServer := lsp.NewLSPServer(workDir)
	updateChecker := update.NewUpdateChecker()
	feedbackSystem := feedback.NewFeedbackSystem()

	if verbose {
		autoCfg := provider.InitAutoConfig()
		providers := pm.ListProviders()
		recProvider := autoCfg.GetRecommendedProvider()
		
		fmt.Printf("  %s[PROVIDER]%s Loaded: %s\n", ASCII_CYAN, ASCII_RESET, strings.Join(providers, ", "))
		fmt.Printf("  %s[PROVIDER]%s Recommended: %s\n", ASCII_GREEN, ASCII_RESET, recProvider)
		fmt.Printf("  %s[SCORPION]%s Deep Search Engine Ready\n", ASCII_GOLD, ASCII_RESET)
		fmt.Printf("  %s[EVOLUTION]%s Learning Engine Ready\n", ASCII_GOLD, ASCII_RESET)
		fmt.Printf("  %s[HOLOGRAM]%s Visual Mode Ready\n", ASCII_GOLD, ASCII_RESET)
		fmt.Printf("  %s[EXPLORER]%s File Navigation Ready\n", ASCII_GOLD, ASCII_RESET)
		fmt.Printf("  %s[SESSION]%s Auto-Save Ready\n", ASCII_GOLD, ASCII_RESET)
		fmt.Printf("  %s[VALIDATOR]%s API Validation Ready\n", ASCII_GOLD, ASCII_RESET)
		fmt.Printf("  %s[LSP]%s Code Analysis Ready\n", ASCII_GOLD, ASCII_RESET)
		fmt.Printf("  %s[UPDATE]%s Auto-Update Ready\n", ASCII_GOLD, ASCII_RESET)
		fmt.Printf("  %s[FEEDBACK]%s User Feedback System Ready\n", ASCII_GOLD, ASCII_RESET)
		fmt.Printf("  %s[VERSION]%s %s\n", ASCII_CYAN, ASCII_RESET, update.CurrentVersion)
	}

	projScanner := scanner.NewProjectScanner(cfg.Agent.Context)
	var projCtx *scanner.ProjectContext
	ctx := context.Background()

	if verbose {
		fmt.Printf("  %s[SCANNER]%s Analyzing project...\n", ASCII_CYAN, ASCII_RESET)
	}
	projCtx, _ = projScanner.Scan(ctx, workDir)

	if projCtx != nil && verbose {
		fmt.Printf("  %s[SCANNER]%s Found %d files, %d lines, deps: %v\n", 
			ASCII_GREEN, ASCII_RESET,
			projCtx.Summary.TotalFiles, 
			projCtx.Summary.TotalLines,
			projCtx.Summary.Dependencies)
	}

	mem := memory.NewMemory(workDir)
	if verbose {
		stats := mem.Stats()
		fmt.Printf("  %s[MEMORY]%s %d entries loaded\n", ASCII_CYAN, ASCII_RESET, stats["total"])
	}

	if godMode {
		fmt.Println()
		godIAEngine.Activate("leader-siby")
		fmt.Println(godIAEngine.RenderDashboard())
		
		snapshot, _ := godIAEngine.TakeSnapshot()
		if snapshot != nil {
			opts := godIAEngine.Optimize()
			fmt.Printf("  %s[OPTIMIZATION]%s %d recommendations\n", ASCII_CYAN, ASCII_RESET, len(opts))
		}
		
		synth.ProcessQuery(ctx, "")
	}

	if evolutionMode {
		fmt.Println()
		fmt.Printf("  %s[EVOLUTION]%s Running nightly optimization...\n", ASCII_GOLD, ASCII_RESET)
		report := evolutionEngine.RunNightlyOptimization()
		fmt.Printf("  %s[EVOLUTION]%s Optimized %d prompts\n", ASCII_GREEN, ASCII_RESET, len(report.Optimizations))
	}

	if verbose {
		fmt.Printf("  %s[READY]%s Ready in %v\n", ASCII_GREEN, ASCII_RESET, time.Since(startTime))
		fmt.Println("")
		fmt.Println("  Type /help for commands or ask me anything!")
		fmt.Println("")
	}

	baseModel := ui.New(cfg, pm, projScanner, projCtx)

	extModel := &ExtendedModel{
		Model:         baseModel,
		scorpion:      scorpionEngine,
		godIA:         godIAEngine,
		evolution:     evolutionEngine,
		hologram:      hologramEngine,
		voice:         voiceEngine,
		cloud:         cloudEngine,
		synthesizer:   synth,
		orch:          orchEngine,
		explorer:      explorerEngine,
		session:       sessionManager,
		interrupt:     interruptHandler,
		validator:     validatorEngine,
		costTracker:   costTracker,
		tokenCounter:  tokenCounter,
		lsp:           lspServer,
		updateChecker:  updateChecker,
		feedbackSystem: feedbackSystem,
	}

	fmt.Println(synth.GenerateIBRAHSignature())

	p := tea.NewProgram(extModel, tea.WithAltScreen())
	if err := p.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func showExtendedHelp() {
	showBanner(true)
	fmt.Println()
	fmt.Printf("  %s🦂 SIBY-AGENTIQ v2.0.0 SOVEREIGN - Help%s\n", ASCII_CYAN, ASCII_RESET)
	fmt.Println()
	fmt.Println("  Usage: siby [OPTIONS] [COMMAND]")
	fmt.Println()
	fmt.Printf("  %s🕹️  Commandes Principales:%s\n", ASCII_GOLD, ASCII_RESET)
	fmt.Println("    leader-siby              Activer GOD-IA (mode secret)")
	fmt.Println("    --evolve, -e            Optimisation nocturne")
	fmt.Println("    scorpion [p] [action]   Gestion des clés API")
	fmt.Println("    --quiet, -q             Sortie minimale")
	fmt.Println("    --help, -h              Cette aide")
	fmt.Println()
	fmt.Printf("  %s⌨️  Commandes TUI:%s\n", ASCII_GOLD, ASCII_RESET)
	fmt.Println("    /help, /h               Afficher l'aide")
	fmt.Println("    /ls [path]              Lister fichiers 📁")
	fmt.Println("    /cd [path]              Changer répertoire")
	fmt.Println("    /scan                   Analyser projet")
	fmt.Println("    /lsp                    Analyse LSP du code 🔍")
	fmt.Println("    /model [name]           Changer provider")
	fmt.Println("    /providers              Providers disponibles")
	fmt.Println("    /cost                   Coût API 💰")
	fmt.Println("    /tokens                 Usage tokens 📊")
	fmt.Println("    /sessions               Liste sessions 💾")
	fmt.Println("    /restore                Restaurer session")
	fmt.Println("    /update                 Vérifier mise à jour 🔄")
	fmt.Println("    /feedback [type] [msg]  Envoyer feedback 💬")
	fmt.Println("    /god                   Dashboard GOD-IA 👁️")
	fmt.Println("    /scorpion [q]          Recherche deep web 🦂")
	fmt.Println("    /quit, /q               Quitter")
	fmt.Println()
	fmt.Printf("  %s🧠 Modules Actifs (45 Agents):%s\n", ASCII_GOLD, ASCII_RESET)
	fmt.Println("    🦂 SCORPION    Deep web search multi-API")
	fmt.Println("    🧬 EVOLUTION   Auto-apprentissage récursif")
	fmt.Println("    👁️  GOD-IA     Vision omnisciente OS")
	fmt.Println("    📁 EXPLORER   Navigation fichiers")
	fmt.Println("    💾 SESSION    Auto-save & Ctrl+C safe")
	fmt.Println("    🔍 LSP        Analyse syntaxe Go")
	fmt.Println("    💰 COST       Tracking coût API")
	fmt.Println("    📊 TOKENS     Gestion contexte (128K)")
	fmt.Println("    🔄 UPDATE     Auto-update GitHub")
	fmt.Println("    💬 FEEDBACK   Système feedback")
	fmt.Println("    🌈 HOLOGRAM   Mode visuel")
	fmt.Println("    ☁️  CLOUD     Sync E2E encrypted")
	fmt.Println()
	fmt.Printf("  %s🎨 Design: Nord Theme + Neon Guinea 🇬🇳%s\n", ASCII_CYAN, ASCII_RESET)
	fmt.Printf("  %sBuilt with ❤️ by Ibrahim Siby 🦂%s\n", ASCII_GOLD, ASCII_RESET)
	fmt.Println()
}
