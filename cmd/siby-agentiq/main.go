package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/siby-agentiq/siby-agentiq/internal/ui"
)

const (
	ASCII_RESET = "\033[0m"
	ASCII_CYAN  = "\033[96m"
	ASCII_GOLD  = "\033[93m"
	ASCII_GREEN = "\033[92m"
	ASCII_RED   = "\033[91m"
)

var splash = `
    _____ _____ ____  __  __   ___ _    _____ _  __
   |_   _| ____|  _ \|  \/  | |_ _|__|___ /| |/ /___ _   _____ _ __ 
     | | |  _| | |_) | |\/| |  | |/ __|_ \| ' // _ \ | / / _ \ '__|
     | | | |___|  _ <| |  | |  | |\__ \__) | . \  __/ | \ \  __/ |   
     |_| |_____|_| \_\_|  |_| |___|___/____|_|\_\___|_| \_|\___|_|   
                                                                         
                      SIBY-AGENTIQ v2.0.0 SOVEREIGN
                      Created by Ibrahim Siby 🇬🇳
`

func main() {
	fmt.Print(ASCII_CYAN)
	fmt.Print(splash)
	fmt.Print(ASCII_RESET)

	fmt.Println()
	fmt.Println("  Status: Ready")
	fmt.Println("  Mode: SOVEREIGN")
	fmt.Println("  Agents: 45 ready")
	fmt.Println()

	if len(os.Args) > 1 && os.Args[1] == "--tui" {
		fmt.Println("  Starting TUI...")
		p := tea.NewProgram(ui.NewModel())
		if err := p.Start(); err != nil {
			fmt.Printf("Error starting TUI: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("  Use --tui to start the interactive TUI")
		fmt.Println("  Use /help for commands")
	}
}
