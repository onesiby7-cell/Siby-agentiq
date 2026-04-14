package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

const asciiArt = `
   _____ _____ ____  __  __   ___ _    _____ _  __
  |_   _| ____|  _ \|  \/  | |_ _|__|___ /| |/ /___ _   _____ _ __ 
    | | |  _| | |_) | |\/| |  | |/ __|_ \| ' // _ \ | / / _ \ '__|
    | | | |___|  _ <| |  | |  | |\__ \__) | . \  __/ | \ \  __/ |   
    |_| |_____|_| \_\_|  |_| |___|___/____|_|\_\___|_| \_/\___|_|   
                                                                      
                     The Last Agent You Will Ever Need

`

const asciiArtSimple = `
  ╔═══════════════════════════════════════════════════════╗
  ║                                                       ║
  ║   ██████╗  ██████╗ ██╗     ██╗     ███████╗██╗  ██╗  ║
  ║   ██╔══██╗██╔═══██╗██║     ██║     ██╔════╝╚██╗██╔╝  ║
  ║   ██████╔╝██║   ██║██║     ██║     █████╗   ╚███╔╝   ║
  ║   ██╔═══╝ ██║   ██║██║     ██║     ██╔══╝   ██╔██╗   ║
  ║   ██║     ╚██████╔╝███████╗███████╗███████╗██╔╝ ██╗  ║
  ║   ╚═╝      ╚═════╝ ╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝  ║
  ║                                                       ║
  ║        A-G-E-N-T-I-Q                                    ║
  ║        The Last Agent You Will Ever Need                ║
  ║                                                       ║
  ╚═══════════════════════════════════════════════════════╝
`

func showSplash(verbose bool) {
	if runtime.GOOS == "windows" {
		fmt.Println("\033[2J\033[H")
	}

	if verbose {
		fmt.Print("\033[96m")
		fmt.Print(asciiArt)
		fmt.Print("\033[0m")
	} else {
		fmt.Print("\033[96m")
		fmt.Print(asciiArtSimple)
		fmt.Print("\033[0m")
	}
}

func showLoadingBar() {
	frames := []string{"[■□□□□□□□□□]", "[■■□□□□□□□□□]", "[■■■□□□□□□□□]", "[■■■■□□□□□□□]", 
	                   "[■■■■■□□□□□□]", "[■■■■■■□□□□□]", "[■■■■■■■□□□□]", 
	                   "[■■■■■■■■□□□]", "[■■■■■■■■■□□]", "[■■■■■■■■■■□]"}
	
	for _, frame := range frames {
		fmt.Printf("\r  Loading %s", frame)
	}
	fmt.Printf("\r  Done!          \n")
}

func main() {
	args := os.Args[1:]
	verbose := true
	for _, arg := range args {
		if arg == "--quiet" || arg == "-q" {
			verbose = false
		}
	}

	showSplash(verbose)
	
	if verbose {
		showLoadingBar()
		fmt.Println("  Starting Siby-Agentiq...\n")
	}
}
