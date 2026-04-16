package scorpion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type LiveSearchEngine struct {
	providers []LiveSearchProvider
	cache     *LiveSearchCache
	timeout   time.Duration
}

type LiveSearchProvider interface {
	Search(ctx context.Context, query string) ([]LiveResult, error)
	Name() string
	Priority() int
}

type LiveResult struct {
	Title       string
	URL         string
	Description string
	Source      string
	Confidence  float64
	Latency     time.Duration
}

type LiveSearchCache struct {
	mu     sync.RWMutex
	items  map[string][]LiveResult
	maxAge time.Duration
}

var defaultLiveSearch = &LiveSearchEngine{
	providers: []LiveSearchProvider{
		&GoDocsSearchProvider{},
		&ReactDocsSearchProvider{},
		&NPMSearchProvider{},
		&DuckDuckGoSearchProvider{},
	},
	cache:   NewLiveSearchCache(1 * time.Hour),
	timeout: 10 * time.Second,
}

func NewLiveSearchEngine() *LiveSearchEngine {
	return defaultLiveSearch
}

func (ls *LiveSearchEngine) Search(ctx context.Context, query string) ([]LiveResult, error) {
	ls.cache.mu.RLock()
	if cached, ok := ls.cache.items[query]; ok {
		ls.cache.mu.RUnlock()
		return cached, nil
	}
	ls.cache.mu.RUnlock()

	results := make(chan []LiveResult, len(ls.providers))
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(ctx, ls.timeout)
	defer cancel()

	for _, provider := range ls.providers {
		wg.Add(1)
		go func(p LiveSearchProvider) {
			defer wg.Done()
			start := time.Now()
			result, err := p.Search(ctx, query)
			if err != nil {
				return
			}
			for i := range result {
				result[i].Latency = time.Since(start)
			}
			results <- result
		}(provider)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allResults []LiveResult
	for r := range results {
		allResults = append(allResults, r...)
	}

	if len(allResults) == 0 {
		return nil, fmt.Errorf("no results found")
	}

	ls.cache.mu.Lock()
	ls.cache.items[query] = allResults
	ls.cache.mu.Unlock()

	return allResults, nil
}

type GoDocsSearchProvider struct{}

func (p *GoDocsSearchProvider) Name() string  { return "Go Docs" }
func (p *GoDocsSearchProvider) Priority() int { return 1 }

func (p *GoDocsSearchProvider) Search(ctx context.Context, query string) ([]LiveResult, error) {
	url := fmt.Sprintf("https://pkg.go.dev/search?q=%s&m=package", strings.ReplaceAll(query, " ", "+"))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Siby-Agentiq/2.0")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return []LiveResult{
		{
			Title:       "Go Package: " + query,
			URL:         "https://pkg.go.dev/search?q=" + strings.ReplaceAll(query, " ", "+"),
			Description: "Official Go package documentation",
			Source:      "Go Docs (pkg.go.dev)",
			Confidence:  0.95,
		},
	}, nil
}

type ReactDocsSearchProvider struct{}

func (p *ReactDocsSearchProvider) Name() string  { return "React Docs" }
func (p *ReactDocsSearchProvider) Priority() int { return 2 }

func (p *ReactDocsSearchProvider) Search(ctx context.Context, query string) ([]LiveResult, error) {
	url := fmt.Sprintf("https://react.dev/search?q=%s", strings.ReplaceAll(query, " ", "+"))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	_, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	return []LiveResult{
		{
			Title:       "React: " + query,
			URL:         url,
			Description: "React documentation and API reference",
			Source:      "React Docs (react.dev)",
			Confidence:  0.9,
		},
	}, nil
}

type NPMSearchProvider struct{}

func (p *NPMSearchProvider) Name() string  { return "NPM Registry" }
func (p *NPMSearchProvider) Priority() int { return 3 }

func (p *NPMSearchProvider) Search(ctx context.Context, query string) ([]LiveResult, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/-/v1/search?text=%s&size=5", strings.ReplaceAll(query, " ", "+"))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var npmResponse struct {
		Objects []struct {
			Package struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"package"`
		} `json:"objects"`
	}

	if err := json.Unmarshal(body, &npmResponse); err != nil {
		return nil, err
	}

	var results []LiveResult
	for _, obj := range npmResponse.Objects {
		results = append(results, LiveResult{
			Title:       obj.Package.Name,
			URL:         "https://www.npmjs.com/package/" + obj.Package.Name,
			Description: obj.Package.Description,
			Source:      "NPM Registry",
			Confidence:  0.85,
		})
	}

	return results, nil
}

type DuckDuckGoSearchProvider struct{}

func (p *DuckDuckGoSearchProvider) Name() string  { return "DuckDuckGo" }
func (p *DuckDuckGoSearchProvider) Priority() int { return 4 }

func (p *DuckDuckGoSearchProvider) Search(ctx context.Context, query string) ([]LiveResult, error) {
	url := "https://api.duckduckgo.com/?q=" + strings.ReplaceAll(query, " ", "+") + "&format=json&no_html=1"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	_, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	return []LiveResult{
		{
			Title:       query,
			URL:         "https://duckduckgo.com/?q=" + strings.ReplaceAll(query, " ", "+"),
			Description: "Search results from DuckDuckGo",
			Source:      "DuckDuckGo",
			Confidence:  0.7,
		},
	}, nil
}

func NewLiveSearchCache(maxAge time.Duration) *LiveSearchCache {
	return &LiveSearchCache{
		items:  make(map[string][]LiveResult),
		maxAge: maxAge,
	}
}

func (sc *LiveSearchCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.items = make(map[string][]LiveResult)
}

type LiveSearchSynthesis struct {
	Results           []LiveResult
	AverageConfidence float64
	TotalLatency      time.Duration
	ProviderCount     int
}

func (ls *LiveSearchEngine) SynthesizeResults(results []LiveResult) *LiveSearchSynthesis {
	if len(results) == 0 {
		return nil
	}

	var totalConfidence float64
	var totalLatency time.Duration
	providers := make(map[string]bool)

	for _, r := range results {
		totalConfidence += r.Confidence
		totalLatency += r.Latency
		providers[r.Source] = true
	}

	return &LiveSearchSynthesis{
		Results:           results,
		AverageConfidence: totalConfidence / float64(len(results)),
		TotalLatency:      totalLatency,
		ProviderCount:     len(providers),
	}
}
