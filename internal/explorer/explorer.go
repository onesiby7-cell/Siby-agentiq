package explorer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	NordBackground = "#2E3440"
	NordPanel = "#3B4252"
	NordCyan = "#88C0D0"
	NordGreen = "#A3BE8C"
	NordYellow = "#EBCB8B"
	NordRed = "#BF616A"
	NordText = "#D8DEE9"
	NordTextMuted = "#4C566A"
	GuineaGreen = "#009460"
	GuineaYellow = "#FCD116"
	GuineaRed = "#CE1126"
)

type FileExplorer struct {
	currentPath string
	history     []string
	maxHistory  int
	width       int
	height      int
	selected    int
	files       []FileEntry
	fuzzyQuery  string
	viewMode    ViewMode
	sortBy      SortBy
}

type FileEntry struct {
	Name      string
	Path      string
	IsDir     bool
	Size      int64
	ModTime   time.Time
	Perms     string
	Icon      string
	Extension string
}

type ViewMode string
type SortBy string

const (
	ViewList  ViewMode = "list"
	ViewGrid  ViewMode = "grid"
	ViewTree  ViewMode = "tree"
)

const (
	SortName    SortBy = "name"
	SortSize    SortBy = "size"
	SortTime    SortBy = "time"
	SortType    SortBy = "type"
)

func NewFileExplorer(width, height int) *FileExplorer {
	cwd, _ := os.Getwd()
	return &FileExplorer{
		currentPath: cwd,
		history:    make([]string, 0),
		maxHistory: 50,
		width:      width,
		height:     height,
		selected:   0,
		viewMode:   ViewList,
		sortBy:     SortName,
	}
}

func (fe *FileExplorer) List(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	fe.files = make([]FileEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		ext := filepath.Ext(entry.Name())
		fe.files = append(fe.files, FileEntry{
			Name:      entry.Name(),
			Path:      filepath.Join(path, entry.Name()),
			IsDir:     entry.IsDir(),
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			Perms:     info.Mode().String(),
			Icon:      fe.getIcon(entry.Name(), entry.IsDir(), ext),
			Extension: ext,
		})
	}

	fe.sort()
	fe.selected = 0
	fe.currentPath = path

	return nil
}

func (fe *FileExplorer) getIcon(name string, isDir bool, ext string) string {
	if isDir {
		return "📁"
	}

	icons := map[string]string{
		".go":     "🐹",
		".rs":     "🦀",
		".py":     "🐍",
		".js":     "📜",
		".ts":     "🔷",
		".tsx":    "⚛️",
		".jsx":    "⚛️",
		".json":   "📋",
		".md":     "📝",
		".yaml":   "⚙️",
		".yml":    "⚙️",
		".toml":   "⚙️",
		".sh":     "🔧",
		".bash":   "🔧",
		".zsh":    "🔧",
		".css":    "🎨",
		".scss":   "🎨",
		".html":   "🌐",
		".htm":    "🌐",
		".svg":    "🖼️",
		".png":    "🖼️",
		".jpg":    "🖼️",
		".jpeg":   "🖼️",
		".gif":    "🖼️",
		".pdf":    "📕",
		".zip":    "📦",
		".tar":    "📦",
		".gz":     "📦",
		".exe":    "⚡",
		".dll":    "⚙️",
		".so":     "⚙️",
		".db":     "💾",
		".sql":    "🗃️",
		".git":    "📂",
		".env":    "🔐",
		".lock":   "🔒",
		"Makefile": "🔨",
		"Dockerfile": "🐳",
	}

	if icon, ok := icons[name]; ok {
		return icon
	}
	if icon, ok := icons[ext]; ok {
		return icon
	}
	return "📄"
}

func (fe *FileExplorer) sort() {
	switch fe.sortBy {
	case SortName:
		sort.Slice(fe.files, func(i, j int) bool {
			if fe.files[i].IsDir != fe.files[j].IsDir {
				return fe.files[i].IsDir
			}
			return strings.ToLower(fe.files[i].Name) < strings.ToLower(fe.files[j].Name)
		})
	case SortSize:
		sort.Slice(fe.files, func(i, j int) bool {
			if fe.files[i].IsDir != fe.files[j].IsDir {
				return fe.files[i].IsDir
			}
			return fe.files[i].Size > fe.files[j].Size
		})
	case SortTime:
		sort.Slice(fe.files, func(i, j int) bool {
			if fe.files[i].IsDir != fe.files[j].IsDir {
				return fe.files[i].IsDir
			}
			return fe.files[i].ModTime.After(fe.files[j].ModTime)
		})
	case SortType:
		sort.Slice(fe.files, func(i, j int) bool {
			if fe.files[i].IsDir != fe.files[j].IsDir {
				return fe.files[i].IsDir
			}
			return fe.files[i].Extension < fe.files[j].Extension
		})
	}
}

func (fe *FileExplorer) FuzzySearch(query string) []FileEntry {
	if query == "" {
		return fe.files
	}

	query = strings.ToLower(query)
	results := make([]FileEntry, 0)

	for _, file := range fe.files {
		if fe.fuzzyMatch(file.Name, query) {
			results = append(results, file)
		}
	}

	return results
}

func (fe *FileExplorer) fuzzyMatch(name, query string) bool {
	name = strings.ToLower(name)
	query = strings.ToLower(query)

	if strings.Contains(name, query) {
		return true
	}

	runes := []rune(query)
	idx := 0
	for _, r := range name {
		if idx < len(runes) && r == runes[idx] {
			idx++
		}
	}
	return idx == len(runes)
}

func (fe *FileExplorer) Cd(path string) error {
	var newPath string

	if filepath.IsAbs(path) {
		newPath = path
	} else if path == ".." {
		newPath = filepath.Dir(fe.currentPath)
	} else if path == "~" {
		home, _ := os.UserHomeDir()
		newPath = home
	} else {
		newPath = filepath.Join(fe.currentPath, path)
	}

	info, err := os.Stat(newPath)
	if err != nil {
		return fmt.Errorf("chemin introuvable: %s", path)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s n'est pas un dossier", path)
	}

	fe.history = append(fe.history, fe.currentPath)
	if len(fe.history) > fe.maxHistory {
		fe.history = fe.history[len(fe.history)-fe.maxHistory:]
	}

	return fe.List(newPath)
}

func (fe *FileExplorer) Back() error {
	if len(fe.history) == 0 {
		return nil
	}

	last := fe.history[len(fe.history)-1]
	fe.history = fe.history[:len(fe.history)-1]
	return fe.List(last)
}

func (fe *FileExplorer) GetSelected() *FileEntry {
	if fe.selected >= 0 && fe.selected < len(fe.files) {
		return &fe.files[fe.selected]
	}
	return nil
}

func (fe *FileExplorer) SelectNext() {
	if fe.selected < len(fe.files)-1 {
		fe.selected++
	}
}

func (fe *FileExplorer) SelectPrev() {
	if fe.selected > 0 {
		fe.selected--
	}
}

func (fe *FileExplorer) SetFuzzyQuery(query string) {
	fe.fuzzyQuery = query
}

func (fe *FileExplorer) Render() string {
	var sb strings.Builder

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(NordCyan)).
		Background(lipgloss.Color(NordBackground)).
		Bold(true)

	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(GuineaYellow))

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(NordCyan))

	sb.WriteString(headerStyle.Render(fmt.Sprintf(" 📁 %s ", fe.currentPath)))
	sb.WriteString("\n")

	dirStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(NordCyan))

	fileStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(NordText))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(NordPanel)).
		Foreground(lipgloss.Color(GuineaYellow)).
		Bold(true)

	sizeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(NordTextMuted))

	for i, file := range fe.files {
		style := fileStyle
		nameStyle := dirStyle
		if file.IsDir {
			nameStyle = dirStyle
		}

		if i == fe.selected {
			style = selectedStyle
			nameStyle = selectedStyle
		}

		var line string
		if file.IsDir {
			line = fmt.Sprintf("  %s %s/", file.Icon, file.Name)
		} else {
			line = fmt.Sprintf("  %s %s", file.Icon, file.Name)
		}

		padding := fe.width - len(stripANSI(line)) - 12
		if padding < 0 {
			padding = 0
		}

		sizeStr := fe.formatSize(file.Size)
		line += strings.Repeat(" ", padding)
		line += sizeStyle.Render(sizeStr)

		sb.WriteString(style.Render(line))
		sb.WriteString("\n")
	}

	return borderStyle.Render(sb.String())
}

func (fe *FileExplorer) formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func (fe *FileExplorer) GetPath() string {
	return fe.currentPath
}

func (fe *FileExplorer) SetViewMode(mode ViewMode) {
	fe.viewMode = mode
}

func (fe *FileExplorer) SetSortBy(sort SortBy) {
	fe.sortBy = sort
	fe.sort()
}

func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape && r == 'm' {
			inEscape = false
			continue
		}
		if !inEscape {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func (fe *FileExplorer) GetFiles() []FileEntry {
	if fe.fuzzyQuery != "" {
		return fe.FuzzySearch(fe.fuzzyQuery)
	}
	return fe.files
}

type FilePicker struct {
	explorer   *FileExplorer
	selected   []string
	multiSelect bool
}

func NewFilePicker(width, height int, multiSelect bool) *FilePicker {
	return &FilePicker{
		explorer:   NewFileExplorer(width, height),
		selected:   make([]string, 0),
		multiSelect: multiSelect,
	}
}

func (fp *FilePicker) Select() string {
	entry := fp.explorer.GetSelected()
	if entry == nil {
		return ""
	}

	if entry.IsDir {
		fp.explorer.Cd(entry.Name)
		return ""
	}

	if fp.multiSelect {
		fp.toggleSelection(entry.Path)
		return ""
	}

	return entry.Path
}

func (fp *FilePicker) toggleSelection(path string) {
	for i, p := range fp.selected {
		if p == path {
			fp.selected = append(fp.selected[:i], fp.selected[i+1:]...)
			return
		}
	}
	fp.selected = append(fp.selected, path)
}

func (fp *FilePicker) GetSelectedPaths() []string {
	return fp.selected
}
