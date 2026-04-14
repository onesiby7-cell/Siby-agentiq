package browser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Browser struct {
	client *http.Client
	ctx    context.Context
}

type PageResult struct {
	URL         string
	Title       string
	Content     string
	Links       []string
	CodeSnippets []string
	Success     bool
	Error       string
}

type SearchResult struct {
	Title       string
	URL         string
	Snippet     string
	Score       float64
}

func NewBrowser() *Browser {
	return &Browser{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		ctx: context.Background(),
	}
}

func (b *Browser) Search(query string) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(b.ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SibyBot/1.0)")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	doc.Find(".result").Each(func(i int, s *goquery.Selection) {
		title := s.Find(".result__title").Text()
		link, _ := s.Find(".result__url").Attr("href")
		snippet := s.Find(".result__snippet").Text()

		if title != "" && link != "" {
			results = append(results, SearchResult{
				Title:   strings.TrimSpace(title),
				URL:     link,
				Snippet: strings.TrimSpace(snippet),
			})
		}
	})

	return results, nil
}

func (b *Browser) Fetch(urlStr string) (*PageResult, error) {
	req, err := http.NewRequestWithContext(b.ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SibyBot/1.0)")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	title := doc.Find("title").Text()
	content := b.extractContent(doc)
	snippets := b.extractCode(doc)
	links := b.extractLinks(doc, urlStr)

	return &PageResult{
		URL:         urlStr,
		Title:       strings.TrimSpace(title),
		Content:     content,
		Links:       links,
		CodeSnippets: snippets,
		Success:     true,
	}, nil
}

func (b *Browser) extractContent(doc *goquery.Document) string {
	doc.Find("script, style, nav, header, footer, aside").Remove()

	text := doc.Find("body").Text()
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	if len(text) > 5000 {
		text = text[:5000] + "..."
	}

	return text
}

func (b *Browser) extractCode(doc *goquery.Document) []string {
	var codeBlocks []string

	doc.Find("pre code, .highlight pre, .code-block").Each(func(i int, s *goquery.Selection) {
		code := s.Text()
		if len(code) > 20 && len(code) < 2000 {
			codeBlocks = append(codeBlocks, strings.TrimSpace(code))
		}
	})

	return codeBlocks
}

func (b *Browser) extractLinks(doc *goquery.Document, baseURL string) []string {
	var links []string

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if href != "" && !strings.HasPrefix(href, "#") && !strings.HasPrefix(href, "javascript:") {
			if !strings.HasPrefix(href, "http") {
				if strings.HasPrefix(href, "/") {
					href = baseURL + href
				} else {
					href = baseURL + "/" + href
				}
			}
			links = append(links, href)
		}
	})

	return links
}

func (b *Browser) SearchStackOverflow(query string) ([]SearchResult, error) {
	searchURL := fmt.Sprintf(
		"https://api.allorigins.win/raw?url=%s",
		url.QueryEscape("https://stackoverflow.com/search?q="+query),
	)

	req, err := http.NewRequestWithContext(b.ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return b.Search(query + " site:stackoverflow.com")
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return b.Search(query + " site:stackoverflow.com")
	}

	var results []SearchResult
	doc.Find(".question-summary").Each(func(i int, s *goquery.Selection) {
		title := s.Find(".question-hyperlink").Text()
		link, _ := s.Find(".question-hyperlink").Attr("href")
		snippet := s.Find(".excerpt").Text()

		if title != "" {
			results = append(results, SearchResult{
				Title:   strings.TrimSpace(title),
				URL:     "https://stackoverflow.com" + link,
				Snippet: strings.TrimSpace(snippet),
			})
		}
	})

	return results, nil
}

func (b *Browser) GetDocumentation(lang, topic string) (*PageResult, error) {
	docURLs := map[string]string{
		"go":     fmt.Sprintf("https://pkg.go.dev/search?q=%s", url.QueryEscape(topic)),
		"python": fmt.Sprintf("https://docs.python.org/3/search.html?q=%s", url.QueryEscape(topic)),
		"rust":   fmt.Sprintf("https://doc.rust-lang.org/std/?search=%s", url.QueryEscape(topic)),
		"node":   fmt.Sprintf("https://nodejs.org/api/%s.html", topic),
		"react":  fmt.Sprintf("https://react.dev/reference/react/%s", topic),
	}

	baseURL, ok := docURLs[lang]
	if !ok {
		return b.Search(topic + " documentation " + lang)
	}

	return b.Fetch(baseURL)
}

func (b *Browser) FetchGitHub(path string) (*PageResult, error) {
	githubURL := fmt.Sprintf("https://raw.githubusercontent.com%s", path)
	return b.Fetch(githubURL)
}

func (b *Browser) Close() {
}

type WebAgent struct {
	browser *Browser
}

func NewWebAgent() *WebAgent {
	return &WebAgent{
		browser: NewBrowser(),
	}
}

func (wa *WebAgent) SearchAndFetch(query string) (string, error) {
	results, err := wa.browser.SearchStackOverflow(query)
	if err != nil || len(results) == 0 {
		results, err = wa.browser.Search(query)
		if err != nil {
			return "", err
		}
	}

	if len(results) == 0 {
		return "No results found", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d results:\n\n", len(results)))

	for i, r := range results {
		if i >= 5 {
			break
		}
		sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n   %s\n\n", i+1, r.Title, r.URL, r.Snippet))
	}

	firstResult := results[0]
	page, err := wa.browser.Fetch(firstResult.URL)
	if err == nil && page.Success {
		sb.WriteString("\n--- Top Answer ---\n")
		for _, code := range page.CodeSnippets[:3] {
			sb.WriteString("```\n")
			sb.WriteString(code)
			sb.WriteString("\n```\n\n")
		}
	}

	return sb.String(), nil
}
