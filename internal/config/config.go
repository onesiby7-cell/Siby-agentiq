package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Debug   bool   `mapstructure:"debug"`
	LogFile string `mapstructure:"log_file"`
}

type OllamaConfig struct {
	BaseURL   string `mapstructure:"base_url"`
	Model     string `mapstructure:"model"`
	Timeout   int    `mapstructure:"timeout"`
	Stream    bool   `mapstructure:"stream"`
	KeepAlive string `mapstructure:"keep_alive"`
}

type AnthropicConfig struct {
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
	Stream      bool    `mapstructure:"stream"`
}

type OpenAIConfig struct {
	APIKey     string  `mapstructure:"api_key"`
	BaseURL    string  `mapstructure:"base_url"`
	Model      string  `mapstructure:"model"`
	MaxTokens  int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
	Stream     bool    `mapstructure:"stream"`
}

type GroqConfig struct {
	Enabled    bool    `mapstructure:"enabled"`
	APIKey     string  `mapstructure:"api_key"`
	DefaultModel string `mapstructure:"default_model"`
	Temperature float64 `mapstructure:"temperature"`
}

type ProviderConfig struct {
	Active    string         `mapstructure:"active"`
	Ollama    OllamaConfig   `mapstructure:"ollama"`
	Groq      GroqConfig     `mapstructure:"groq"`
	Anthropic AnthropicConfig `mapstructure:"anthropic"`
	OpenAI    OpenAIConfig   `mapstructure:"openai"`
}

type ChainOfThoughtConfig struct {
	Enabled          bool `mapstructure:"enabled"`
	MaxIterations    int  `mapstructure:"max_iterations"`
	ReflectionEnabled bool `mapstructure:"reflection_enabled"`
}

type ContextConfig struct {
	MaxTokens        int      `mapstructure:"max_tokens"`
	ProjectScanDepth int      `mapstructure:"project_scan_depth"`
	IncludePatterns  []string `mapstructure:"include_patterns"`
	ExcludePatterns  []string `mapstructure:"exclude_patterns"`
}

type CodeGenerationConfig struct {
	FormatOnSave  bool `mapstructure:"format_on_save"`
	LinterEnabled bool `mapstructure:"linter_enabled"`
	AutoImport    bool `mapstructure:"auto_import"`
	TypeInference bool `mapstructure:"type_inference"`
}

type AgentConfig struct {
	ChainOfThought   ChainOfThoughtConfig   `mapstructure:"chain_of_thought"`
	Context          ContextConfig          `mapstructure:"context"`
	CodeGeneration   CodeGenerationConfig   `mapstructure:"code_generation"`
}

type ColorsConfig struct {
	Primary    string `mapstructure:"primary"`
	Secondary  string `mapstructure:"secondary"`
	Success    string `mapstructure:"success"`
	Warning    string `mapstructure:"warning"`
	Error      string `mapstructure:"error"`
	Background string `mapstructure:"background"`
	Text       string `mapstructure:"text"`
	Muted      string `mapstructure:"muted"`
}

type KeybindingsConfig struct {
	Submit        string `mapstructure:"submit"`
	Cancel        string `mapstructure:"cancel"`
	HistoryUp     string `mapstructure:"history_up"`
	HistoryDown   string `mapstructure:"history_down"`
	ToggleSidebar string `mapstructure:"toggle_sidebar"`
	ClearScreen   string `mapstructure:"clear_screen"`
	Help          string `mapstructure:"help"`
}

type TUIConfig struct {
	Theme       string             `mapstructure:"theme"`
	Width       int                `mapstructure:"width"`
	Height      int                `mapstructure:"height"`
	Scrollback  int                `mapstructure:"scrollback"`
	Colors      ColorsConfig       `mapstructure:"colors"`
	Keybindings KeybindingsConfig  `mapstructure:"keybindings"`
}

type PlanningConfig struct {
	AutoGenerate     bool `mapstructure:"auto_generate"`
	ConfirmBeforeExecute bool `mapstructure:"confirm_before_execute"`
	EstimateComplexity bool `mapstructure:"estimate_complexity"`
}

type ReflectionConfig struct {
	CriticEnabled      bool `mapstructure:"critic_enabled"`
	SuggestImprovements bool `mapstructure:"suggest_improvements"`
	DetectEdgeCases    bool `mapstructure:"detect_edge_cases"`
}

type ReasoningConfig struct {
	ShowThinking   bool           `mapstructure:"show_thinking"`
	ShowPlanning   bool           `mapstructure:"show_planning"`
	ShowReflection bool           `mapstructure:"show_reflection"`
	Planning       PlanningConfig `mapstructure:"planning"`
	Reflection     ReflectionConfig `mapstructure:"reflection"`
}

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Provider ProviderConfig `mapstructure:"provider"`
	Agent    AgentConfig    `mapstructure:"agent"`
	TUI      TUIConfig      `mapstructure:"tui"`
	Reasoning ReasoningConfig `mapstructure:"reasoning"`
}

func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, ".siby-agentiq", "config.yaml")
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("config file not found: %s", configPath)
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Save(configPath string) error {
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, ".siby-agentiq", "config.yaml")
	}

	viper.SetConfigFile(configPath)
	
	for key, value := range map[string]interface{}{
		"app": c.App,
		"provider": c.Provider,
		"agent": c.Agent,
		"tui": c.TUI,
		"reasoning": c.Reasoning,
	} {
		viper.Set(key, value)
	}

	return viper.WriteConfigAs(configPath)
}

func GetDefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:    "Siby-Agentiq",
			Version: "0.1.0",
			Debug:   false,
			LogFile: "~/.siby-agentiq/logs/siby.log",
		},
		Provider: ProviderConfig{
			Active: "ollama",
			Ollama: OllamaConfig{
				BaseURL:   "http://localhost:11434",
				Model:     "llama3.2:latest",
				Timeout:   120,
				Stream:    true,
				KeepAlive: "5m",
			},
			Anthropic: AnthropicConfig{
				Model:       "claude-sonnet-4-20250514",
				MaxTokens:   8192,
				Temperature: 0.7,
				Stream:      true,
			},
			OpenAI: OpenAIConfig{
				BaseURL:    "https://api.openai.com/v1",
				Model:      "gpt-4o",
				MaxTokens:  4096,
				Temperature: 0.7,
				Stream:     true,
			},
		},
		Agent: AgentConfig{
			ChainOfThought: ChainOfThoughtConfig{
				Enabled:          true,
				MaxIterations:    5,
				ReflectionEnabled: true,
			},
			Context: ContextConfig{
				MaxTokens:        128000,
				ProjectScanDepth: 10,
				IncludePatterns: []string{
					"*.go", "*.rs", "*.ts", "*.js", "*.py",
					"*.java", "*.cpp", "*.c", "*.h", "*.md",
					"*.yaml", "*.yml", "*.json", "*.toml",
				},
				ExcludePatterns: []string{
					".git/**", "node_modules/**", "target/**",
					"dist/**", "build/**", "*.exe", "*.dll",
					".env", ".DS_Store",
				},
			},
			CodeGeneration: CodeGenerationConfig{
				FormatOnSave:  true,
				LinterEnabled: false,
				AutoImport:    true,
				TypeInference: true,
			},
		},
		TUI: TUIConfig{
			Theme:      "dark",
			Width:      120,
			Height:     40,
			Scrollback: 10000,
			Colors: ColorsConfig{
				Primary:    "#7C3AED",
				Secondary:  "#06B6D4",
				Success:    "#10B981",
				Warning:    "#F59E0B",
				Error:      "#EF4444",
				Background: "#0F172A",
				Text:       "#F1F5F9",
				Muted:      "#64748B",
			},
		},
		Reasoning: ReasoningConfig{
			ShowThinking:   true,
			ShowPlanning:   true,
			ShowReflection: true,
			Planning: PlanningConfig{
				AutoGenerate:           true,
				ConfirmBeforeExecute:   true,
				EstimateComplexity:     true,
			},
			Reflection: ReflectionConfig{
				CriticEnabled:      true,
				SuggestImprovements: true,
				DetectEdgeCases:    true,
			},
		},
	}
}
