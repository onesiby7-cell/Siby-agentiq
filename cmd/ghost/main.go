package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/config"
	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

const GHOST_PORT = 3847

type GhostRequest struct {
	Command    string `json:"command"`
	Context    string `json:"context,omitempty"`
	FileFilter string `json:"file_filter,omitempty"`
}

type GhostResponse struct {
	Success bool     `json:"success"`
	Result  string   `json:"result,omitempty"`
	Error   string   `json:"error,omitempty"`
	Files   []string `json:"files,omitempty"`
}

type GhostBridge struct {
	pm     *provider.ProviderManager
	config *config.Config
	server *http.Server
}

func NewGhostBridge() *GhostBridge {
	cfg, _ := config.LoadConfig("")
	if cfg == nil {
		cfg = config.GetDefaultConfig()
	}
	pm := provider.NewProviderManager(cfg.Provider)
	return &GhostBridge{pm: pm, config: cfg}
}

func (g *GhostBridge) Start(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ask", g.handleAsk)
	mux.HandleFunc("/search", g.handleSearch)
	mux.HandleFunc("/status", g.handleStatus)

	g.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	fmt.Printf("Siby Ghost listening on port %d\n", port)
	fmt.Println("Configure your IDE to connect to this port for AI assistance.")
	return g.server.ListenAndServe()
}

func (g *GhostBridge) handleAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req GhostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(GhostResponse{Success: false, Error: err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := []provider.Message{
		{Role: "system", Content: getGhostSystemPrompt()},
		{Role: "user", Content: req.Command},
	}

	resp, err := g.pm.SmartChat(ctx, provider.SmartChatRequest{Messages: messages})
	if err != nil {
		json.NewEncoder(w).Encode(GhostResponse{Success: false, Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(GhostResponse{Success: true, Result: resp.Message.Content})
}

func (g *GhostBridge) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "q parameter required", http.StatusBadRequest)
		return
	}
	files := g.searchFiles(query)
	json.NewEncoder(w).Encode(GhostResponse{Success: true, Files: files})
}

func (g *GhostBridge) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := g.pm.CheckAllAvailability()
	resp := map[string]interface{}{
		"success":   true,
		"active":    g.pm.GetActiveName(),
		"providers": status,
	}
	json.NewEncoder(w).Encode(resp)
}

func (g *GhostBridge) searchFiles(query string) []string {
	var results []string
	workDir, _ := os.Getwd()
	filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.Contains(path, "node_modules") || strings.Contains(path, ".git") || strings.Contains(path, "target") {
				return filepath.SkipDir
			}
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(strings.ToLower(string(content)), strings.ToLower(query)) {
			relPath, _ := filepath.Rel(workDir, path)
			results = append(results, relPath)
		}
		return nil
	})
	return results
}

func getGhostSystemPrompt() string {
	return `You are Siby-Ghost, an AI assistant for IDE integration. Provide concise responses about code.`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	bridge := NewGhostBridge()

	switch os.Args[1] {
	case "start":
		port := GHOST_PORT
		if len(os.Args) > 2 {
			fmt.Sscanf(os.Args[2], "%d", &port)
		}
		if err := bridge.Start(port); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	case "ask":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: siby-ghost ask 'question'")
			return
		}
		query := strings.Join(os.Args[2:], " ")
		resp := askQuestion(query)
		fmt.Println(resp)
	case "listen":
		listenMode()
	default:
		printUsage()
	}
}

func askQuestion(query string) string {
	cfg, _ := config.LoadConfig("")
	if cfg == nil {
		cfg = config.GetDefaultConfig()
	}
	pm := provider.NewProviderManager(cfg.Provider)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	messages := []provider.Message{{Role: "user", Content: query}}
	resp, err := pm.SmartChat(ctx, provider.SmartChatRequest{Messages: messages})
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return resp.Message.Content
}

func listenMode() {
	fmt.Println("Listening for IDE commands...")
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("siby> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "quit" || input == "exit" {
			break
		}
		if input != "" {
			fmt.Println(askQuestion(input))
		}
	}
}

func printUsage() {
	fmt.Println("Siby-Ghost - IDE Bridge for Siby-Agentiq")
	fmt.Println("Usage:")
	fmt.Println("  siby-ghost start [port]  - Start server for IDE")
	fmt.Println("  siby-ghost ask 'question' - Ask a question")
	fmt.Println("  siby-ghost listen        - Interactive mode")
}
