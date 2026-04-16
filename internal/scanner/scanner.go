package scanner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/siby-agentiq/siby-agentiq/internal/provider"
)

type ScannerConfig struct {
	MaxFiles        int
	MaxFileSizeKB   int
	ContextMode     string
	IncludePatterns []string
	ExcludePatterns []string
	RootPatterns    []string
}

type ProjectScanner struct {
	cfg   ScannerConfig
	mu    sync.RWMutex
	cache map[string]*ProjectContext
}

type ProjectContext struct {
	Files    []provider.FileInfo
	Contents map[string]string
	Summary  ProjectSummary
}

type ProjectSummary struct {
	TotalFiles   int
	TotalLines   int
	Languages    map[string]int
	MainFiles    []string
	Dependencies []string
}

func NewProjectScanner(cfg ScannerConfig) *ProjectScanner {
	return &ProjectScanner{
		cfg:   cfg,
		cache: make(map[string]*ProjectContext),
	}
}

func (ps *ProjectScanner) Scan(ctx context.Context, rootPath string) (*ProjectContext, error) {
	ps.mu.RLock()
	if cached, ok := ps.cache[rootPath]; ok {
		ps.mu.RUnlock()
		return cached, nil
	}
	ps.mu.RUnlock()

	result := &ProjectContext{
		Contents: make(map[string]string),
	}

	files, err := ps.walkDirectory(ctx, rootPath)
	if err != nil {
		return nil, err
	}

	result.Files = files

	if ps.cfg.ContextMode != "minimal" {
		if err := ps.loadFileContents(ctx, rootPath, result); err != nil {
			return nil, err
		}
	}

	result.Summary = ps.buildSummary(result)

	ps.mu.Lock()
	ps.cache[rootPath] = result
	ps.mu.Unlock()

	return result, nil
}

func (ps *ProjectScanner) walkDirectory(ctx context.Context, rootPath string) ([]provider.FileInfo, error) {
	var files []provider.FileInfo
	var mu sync.Mutex
	var wg sync.WaitGroup
	fileChan := make(chan provider.FileInfo, 1000)
	errChan := make(chan error, 100)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for f := range fileChan {
			mu.Lock()
			files = append(files, f)
			mu.Unlock()
		}
	}()

	scanFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			if ps.shouldExclude(path) {
				return filepath.SkipDir
			}
			mu.Lock()
			files = append(files, provider.FileInfo{
				Name:  info.Name(),
				IsDir: true,
				Depth: strings.Count(path[len(rootPath):], string(filepath.Separator)),
			})
			mu.Unlock()
			return nil
		}

		if !ps.shouldInclude(info.Name()) {
			return nil
		}

		if info.Size() > int64(ps.cfg.MaxFileSizeKB)*1024 {
			return nil
		}

		depth := strings.Count(path[len(rootPath):], string(filepath.Separator))
		wg.Add(1)
		go func(p string, d int) {
			defer wg.Done()
			fileChan <- provider.FileInfo{Name: filepath.Base(p), Depth: d}
		}(path, depth)

		return nil
	}

	err := filepath.Walk(rootPath, scanFunc)
	close(fileChan)
	wg.Wait()

	select {
	case err := <-errChan:
		if err != nil {
			return nil, err
		}
	default:
	}

	if err != nil && err != filepath.SkipAll {
		return nil, err
	}

	return files, nil
}

func (ps *ProjectScanner) shouldExclude(path string) bool {
	relPath, _ := filepath.Rel(".", path)
	for _, pattern := range ps.cfg.ExcludePatterns {
		pattern = strings.TrimSuffix(pattern, "/**")
		pattern = strings.TrimSuffix(pattern, "/**")
		if strings.HasPrefix(relPath, pattern) || strings.Contains(relPath, pattern) {
			return true
		}
	}
	return false
}

func (ps *ProjectScanner) shouldInclude(name string) bool {
	for _, pattern := range ps.cfg.RootPatterns {
		pattern = strings.TrimPrefix(pattern, "*")
		if strings.HasSuffix(name, pattern) {
			return true
		}
	}
	return false
}

func (ps *ProjectScanner) loadFileContents(ctx context.Context, rootPath string, result *ProjectContext) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 50)
	mu := sync.Mutex{}

	for _, file := range result.Files {
		if file.IsDir {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{}
		go func(f provider.FileInfo) {
			defer wg.Done()
			defer func() { <-semaphore }()

			fullPath := filepath.Join(rootPath, f.Name)
			content, err := ps.readFile(fullPath)
			if err != nil {
				return
			}

			mu.Lock()
			result.Contents[f.Name] = content
			mu.Unlock()
		}(file)
	}

	wg.Wait()
	return nil
}

func (ps *ProjectScanner) readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if !utf8.Valid(data) {
		return "", nil
	}

	content := string(data)
	if ps.cfg.ContextMode == "smart" && len(content) > 50000 {
		content = content[:50000] + "\n... [truncated]"
	}

	return content, nil
}

func (ps *ProjectScanner) buildSummary(ctx *ProjectContext) ProjectSummary {
	summary := ProjectSummary{
		Languages: make(map[string]int),
	}

	for _, file := range ctx.Files {
		if file.IsDir {
			continue
		}
		summary.TotalFiles++
		ext := getExtension(file.Name)
		summary.Languages[ext]++

		if isMainFile(file.Name, ext) {
			summary.MainFiles = append(summary.MainFiles, file.Name)
		}

		if content, ok := ctx.Contents[file.Name]; ok {
			lines := strings.Count(content, "\n")
			summary.TotalLines += lines
		}
	}

	summary.Dependencies = ps.detectDependencies(ctx)
	return summary
}

func (ps *ProjectScanner) detectDependencies(ctx *ProjectContext) []string {
	var deps []string
	depPatterns := map[string][]string{
		"go":     {"go.mod", "go.sum"},
		"rust":   {"Cargo.toml", "Cargo.lock"},
		"node":   {"package.json", "package-lock.json"},
		"python": {"requirements.txt", "pyproject.toml", "Pipfile"},
		"java":   {"pom.xml", "build.gradle"},
	}

	for lang, files := range depPatterns {
		for _, file := range files {
			if _, ok := ctx.Contents[file]; ok {
				deps = append(deps, lang)
				break
			}
		}
	}

	return deps
}

func getExtension(name string) string {
	ext := filepath.Ext(name)
	return strings.TrimPrefix(ext, ".")
}

func isMainFile(name, ext string) bool {
	mains := map[string]bool{
		"main.go":  true,
		"main.rs":  true,
		"main.py":  true,
		"index.ts": true,
		"index.js": true,
		"app.go":   true,
		"lib.rs":   true,
		"App.tsx":  true,
		"App.jsx":  true,
	}
	return mains[name]
}

func (ps *ProjectScanner) GetFormattedContext(ctx *ProjectContext, mode string) string {
	var sb strings.Builder

	formatter := provider.NewContextFormatter()
	sb.WriteString(formatter.FormatFileTree(ctx.Files))

	if mode != "minimal" {
		var fileContents []provider.FileContent
		for name, content := range ctx.Contents {
			fileContents = append(fileContents, provider.FileContent{
				Path:     name,
				Content:  content,
				Language: getExtension(name),
			})
		}
		sb.WriteString(formatter.FormatFileContent(fileContents))
	}

	return sb.String()
}
