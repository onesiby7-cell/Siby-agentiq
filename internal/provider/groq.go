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

type GroqProvider struct {
	apiKey      string
	model       string
	baseURL     string
	temperature float64
	maxTokens   int
	client      *http.Client
}

type GroqConfig struct {
	Enabled    bool
	APIKey     string
	DefaultModel string
	Temperature float64
}

var GroqModels = map[string]string{
	"llama-3.3-70b":         "Llama 3.3 70B Versatile",
	"mixtral-8x7b":          "Mixtral 8x7B",
	"mixtral-8x22b":         "Mixtral 8x22B",
	"llama-3.1-8b":         "Llama 3.1 8B Instant",
	"llama-3.2-1b":         "Llama 3.2 1B Preview",
	"llama-3.2-3b":         "Llama 3.2 3B Preview",
	"gemma2-9b":             "Gemma 2 9B",
}

func NewGroqProvider(cfg config.GroqConfig) *GroqProvider {
	model := "llama-3.3-70b-versatile"
	if cfg.DefaultModel != "" {
		model = cfg.DefaultModel
	}
	temp := 0.7
	if cfg.Temperature > 0 {
		temp = cfg.Temperature
	}
	return &GroqProvider{
		apiKey:      cfg.APIKey,
		model:       model,
		baseURL:     "https://api.groq.com/openai/v1",
		temperature: temp,
		maxTokens:   8192,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *GroqProvider) Name() string { return "groq" }
func (p *GroqProvider) Priority() int { return 0 }

func (p *GroqProvider) IsAvailable() bool {
	if p.apiKey == "" || p.apiKey == "${GROQ_API_KEY}" {
		return false
	}
	return true
}

func (p *GroqProvider) Chat(ctx context.Context, messages []Message) (*ChatResponse, error) {
	start := time.Now()
	
	groqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		content := msg.Content
		if msg.Role == "system" {
			groqMessages[i] = map[string]interface{}{
				"role":    "system",
				"content": content,
			}
		} else {
			groqMessages[i] = map[string]interface{}{
				"role":    msg.Role,
				"content": content,
			}
		}
	}

	reqBody := map[string]interface{}{
		"model":       p.model,
		"messages":    groqMessages,
		"temperature": p.temperature,
		"max_tokens": p.maxTokens,
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
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("groq unmarshal error: %w - body: %s", err, string(body))
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("empty response from Groq")
	}

	return &ChatResponse{
		Message: Message{
			Role:    "assistant",
			Content: result.Choices[0].Message.Content,
		},
		Usage: Usage{
			InputTokens:  result.Usage.PromptTokens,
			OutputTokens: result.Usage.CompletionTokens,
		},
		LatencyMS: time.Since(start).Milliseconds(),
	}, nil
}

func (p *GroqProvider) ChatStream(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	groqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		content := msg.Content
		if msg.Role == "system" {
			groqMessages[i] = map[string]interface{}{
				"role":    "system",
				"content": content,
			}
		} else {
			groqMessages[i] = map[string]interface{}{
				"role":    msg.Role,
				"content": content,
			}
		}
	}

	reqBody := map[string]interface{}{
		"model":       p.model,
		"messages":    groqMessages,
		"temperature": p.temperature,
		"max_tokens": p.maxTokens,
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
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
					FinishReason string `json:"finish_reason"`
				} `json:"choices"`
			}
			if err := dec.Decode(&r); err != nil {
				if err == io.EOF {
					return
				}
				return
			}
			if len(r.Choices) > 0 {
				chunk := StreamChunk{
					Content: r.Choices[0].Delta.Content,
					Done:    r.Choices[0].FinishReason == "stop",
				}
				select {
				case ch <- chunk:
				case <-ctx.Done():
					return
				}
				if chunk.Done {
					return
				}
			}
		}
	}()
	return ch, nil
}

func (p *GroqProvider) SetModel(model string) {
	if _, ok := GroqModels[model]; ok {
		p.model = model
	}
}

func ListGroqModels() []string {
	models := make([]string, 0, len(GroqModels))
	for id := range GroqModels {
		models = append(models, id)
	}
	return models
}
