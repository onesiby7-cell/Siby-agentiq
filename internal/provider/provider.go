package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/siby-agentiq/siby-agentiq/internal/config"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type ChatResponse struct {
	Message     Message `json:"message"`
	Done        bool    `json:"done"`
	Usage       Usage   `json:"usage,omitempty"`
	LatencyMS   int64   `json:"-"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type StreamChunk struct {
	Content string
	Done    bool
	Usage   Usage
}

type Provider interface {
	Chat(ctx context.Context, messages []Message) (*ChatResponse, error)
	ChatStream(ctx context.Context, messages []Message) (<-chan StreamChunk, error)
	Name() string
	IsAvailable() bool
	Priority() int
}

type OllamaProvider struct {
	baseURL   string
	model     string
	client    *http.Client
	keepAlive string
}

func NewOllamaProvider(cfg config.OllamaConfig) *OllamaProvider {
	timeout := 120
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}
	return &OllamaProvider{
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		keepAlive: cfg.KeepAlive,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

func (p *OllamaProvider) Name() string  { return "ollama" }
func (p *OllamaProvider) Priority() int { return 1 }

func (p *OllamaProvider) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (p *OllamaProvider) Chat(ctx context.Context, messages []Message) (*ChatResponse, error) {
	start := time.Now()
	reqBody := map[string]interface{}{
		"model":      p.model,
		"messages":   messages,
		"stream":      false,
		"keep_alive":  p.keepAlive,
	}
	return p.doRequest(ctx, reqBody, start)
}

func (p *OllamaProvider) ChatStream(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	reqBody := map[string]interface{}{
		"model":      p.model,
		"messages":   messages,
		"stream":      true,
		"keep_alive":  p.keepAlive,
	}
	return p.doStreamRequest(ctx, reqBody)
}

func (p *OllamaProvider) doRequest(ctx context.Context, reqBody map[string]interface{}, start time.Time) (*ChatResponse, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}
	return &ChatResponse{
		Message:   Message{Role: "assistant", Content: result.Message.Content},
		Done:      result.Done,
		LatencyMS: time.Since(start).Milliseconds(),
	}, nil
}

func (p *OllamaProvider) doStreamRequest(ctx context.Context, reqBody map[string]interface{}) (<-chan StreamChunk, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	ch := make(chan StreamChunk, 100)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		dec := json.NewDecoder(resp.Body)
		for {
			var r struct {
				Message struct{ Content string } `json:"message"`
				Done bool `json:"done"`
			}
			if err := dec.Decode(&r); err != nil {
				if err == io.EOF {
					return
				}
				return
			}
			select {
			case ch <- StreamChunk{Content: r.Message.Content, Done: r.Done}:
			case <-ctx.Done():
				return
			}
			if r.Done {
				return
			}
		}
	}()
	return ch, nil
}

type AnthropicProvider struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	client      *http.Client
}

func NewAnthropicProvider(cfg config.AnthropicConfig) *AnthropicProvider {
	maxTokens := 8192
	if cfg.MaxTokens > 0 {
		maxTokens = cfg.MaxTokens
	}
	temp := 0.7
	if cfg.Temperature > 0 {
		temp = cfg.Temperature
	}
	return &AnthropicProvider{
		apiKey:      cfg.APIKey,
		model:       cfg.Model,
		maxTokens:   maxTokens,
		temperature: temp,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *AnthropicProvider) Name() string  { return "anthropic" }
func (p *AnthropicProvider) Priority() int { return 3 }

func (p *AnthropicProvider) IsAvailable() bool {
	return p.apiKey != "" && p.apiKey != "${ANTHROPIC_API_KEY}"
}

func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message) (*ChatResponse, error) {
	start := time.Now()
	anthropicMessages := make([]map[string]string, len(messages))
	for i, msg := range messages {
		anthropicMessages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}
	reqBody := map[string]interface{}{
		"model":         p.model,
		"messages":      anthropicMessages,
		"max_tokens":    p.maxTokens,
		"temperature":    p.temperature,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	return &ChatResponse{
		Message:   Message{Role: "assistant", Content: result.Content[0].Text},
		Usage:     Usage{InputTokens: result.Usage.InputTokens, OutputTokens: result.Usage.OutputTokens},
		LatencyMS: time.Since(start).Milliseconds(),
	}, nil
}

func (p *AnthropicProvider) ChatStream(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 100)
	go func() {
		defer close(ch)
		resp, err := p.Chat(ctx, messages)
		if err != nil {
			return
		}
		for _, r := range resp.Message.Content {
			ch <- StreamChunk{Content: string(r)}
		}
		ch <- StreamChunk{Done: true, Usage: resp.Usage}
	}()
	return ch, nil
}

type OpenAIProvider struct {
	apiKey      string
	baseURL     string
	model       string
	temperature float64
	client      *http.Client
}

func NewOpenAIProvider(cfg config.OpenAIConfig) *OpenAIProvider {
	baseURL := "https://api.openai.com/v1"
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}
	temp := 0.7
	if cfg.Temperature > 0 {
		temp = cfg.Temperature
	}
	return &OpenAIProvider{
		apiKey:      cfg.APIKey,
		baseURL:     baseURL,
		model:       cfg.Model,
		temperature: temp,
		client:      &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string  { return "openai" }
func (p *OpenAIProvider) Priority() int { return 2 }

func (p *OpenAIProvider) IsAvailable() bool {
	return p.apiKey != "" && p.apiKey != "${OPENAI_API_KEY}"
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message) (*ChatResponse, error) {
	start := time.Now()
	reqBody := map[string]interface{}{
		"model":       p.model,
		"messages":    messages,
		"temperature": p.temperature,
		"stream":      false,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	return &ChatResponse{
		Message:   Message{Role: "assistant", Content: result.Choices[0].Message.Content},
		Usage:     Usage{InputTokens: result.Usage.PromptTokens, OutputTokens: result.Usage.CompletionTokens},
		LatencyMS: time.Since(start).Milliseconds(),
	}, nil
}

func (p *OpenAIProvider) ChatStream(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	reqBody := map[string]interface{}{
		"model":       p.model,
		"messages":    messages,
		"temperature": p.temperature,
		"stream":      true,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	ch := make(chan StreamChunk, 100)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		dec := json.NewDecoder(resp.Body)
		for {
			var r struct {
				Choices []struct {
					Delta struct{ Content string } `json:"delta"`
					Finish bool `json:"finish_reason"`
				} `json:"choices"`
			}
			if err := dec.Decode(&r); err != nil {
				if err == io.EOF {
					return
				}
				return
			}
			if len(r.Choices) > 0 {
				chunk := StreamChunk{Content: r.Choices[0].Delta.Content, Done: r.Choices[0].Finish}
				select {
				case ch <- chunk:
				case <-ctx.Done():
					return
				}
				if r.Choices[0].Finish {
					return
				}
			}
		}
	}()
	return ch, nil
}
