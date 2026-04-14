package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/siby-agentiq/siby-agentiq/internal/config"
)

type FileEntry struct {
	Path     string
	Content  string
	Language string
	Size     int64
}

type ProjectScanner struct {
	config    config.ContextConfig
	excludePatterns []string
}

func NewProjectScanner(cfg config.ContextConfig) *ProjectScanner {
	return &ProjectScanner{
		config:          cfg,
		excludePatterns: cfg.ExcludePatterns,
	}
}

func (ps *ProjectScanner) ScanProject(rootPath string) (string, error) {
	var context strings.Builder
	
	fileCount := 0
	totalSize := int64(0)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if ps.shouldExclude(path) {
				return filepath.SkipDir
			}
			return nil
		}

		if !ps.shouldInclude(path) {
			return nil
		}

		if info.Size() > 1*1024*1024 {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(rootPath, path)
		context.WriteString(fmt.Sprintf("\n// File: %s\n", relPath))
		context.Write(content)
		context.WriteString("\n")
		
		fileCount++
		totalSize += info.Size()

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to scan project: %w", err)
	}

	header := fmt.Sprintf("=== PROJECT CONTEXT (%d files, %d KB) ===\n", fileCount, totalSize/1024)
	return header + context.String(), nil
}

func (ps *ProjectScanner) shouldExclude(path string) bool {
	path = filepath.ToSlash(path)
	
	for _, pattern := range ps.excludePatterns {
		pattern = filepath.ToSlash(pattern)
		
		if strings.HasPrefix(pattern, "**/") {
			pattern = pattern[3:]
			if strings.Contains(path, pattern) {
				return true
			}
			continue
		}
		
		if strings.Contains(pattern, "*") {
			matched, _ := filepath.Match(pattern, filepath.Base(path))
			if matched {
				return true
			}
			patternDir := strings.TrimSuffix(pattern, "*")
			if strings.Contains(path, patternDir) {
				return true
			}
			continue
		}
		
		if strings.Contains(path, pattern) {
			return true
		}
	}
	
	return false
}

func (ps *ProjectScanner) shouldInclude(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	includeExts := []string{
		".go", ".rs", ".ts", ".tsx", ".js", ".jsx", ".py", ".java",
		".cpp", ".c", ".h", ".hpp", ".cs", ".rb", ".php", ".swift",
		".kt", ".scala", ".lua", ".r", ".sql", ".sh", ".bash",
		".zsh", ".ps1", ".bat", ".yaml", ".yml", ".json", ".toml",
		".xml", ".html", ".css", ".scss", ".md", ".rst", ".txt",
	}
	
	for _, includeExt := range includeExts {
		if ext == includeExt {
			return true
		}
	}
	
	return false
}

func (ps *ProjectScanner) GetProjectStructure(rootPath string) (map[string]interface{}, error) {
	structure := make(map[string]interface{})
	
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(rootPath, path)
		
		if ps.shouldExclude(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			structure[relPath] = map[string]string{"type": "directory"}
		} else {
			ext := filepath.Ext(path)
			structure[relPath] = map[string]interface{}{
				"type":     "file",
				"language": getLanguage(ext),
				"size":     info.Size(),
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return structure, nil
}

func getLanguage(ext string) string {
	ext = strings.ToLower(ext)
	languages := map[string]string{
		".go": "Go", ".rs": "Rust", ".ts": "TypeScript", ".tsx": "TypeScript",
		".js": "JavaScript", ".jsx": "JavaScript", ".py": "Python", ".java": "Java",
		".cpp": "C++", ".c": "C", ".h": "C", ".hpp": "C++", ".cs": "C#",
		".rb": "Ruby", ".php": "PHP", ".swift": "Swift", ".kt": "Kotlin",
		".scala": "Scala", ".lua": "Lua", ".r": "R", ".sql": "SQL",
		".sh": "Shell", ".bash": "Bash", ".ps1": "PowerShell",
		".yaml": "YAML", ".yml": "YAML", ".json": "JSON", ".toml": "TOML",
		".xml": "XML", ".html": "HTML", ".css": "CSS", ".md": "Markdown",
	}
	
	if lang, ok := languages[ext]; ok {
		return lang
	}
	return "Unknown"
}

func (ps *ProjectScanner) ReadFile(path string) (*FileEntry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	return &FileEntry{
		Path:     path,
		Content:  string(content),
		Language: getLanguage(filepath.Ext(path)),
		Size:     info.Size(),
	}, nil
}

func (ps *ProjectScanner) WriteFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
