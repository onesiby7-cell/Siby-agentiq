package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"math"
)

type Memory struct {
	mu       sync.RWMutex
	entries  map[string]*MemoryEntry
	index    *VectorIndex
	cfg      MemoryConfig
	rootPath string
}

type MemoryConfig struct {
	MaxEntries    int
	SimilarityThreshold float32
}

type MemoryEntry struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Path      string    `json:"path,omitempty"`
	Embedding []float32 `json:"-"`
	Metadata  MemoryMetadata `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	Accessed  int       `json:"access_count"`
	Success   bool      `json:"success,omitempty"`
}

type MemoryMetadata struct {
	Language   string   `json:"language,omitempty"`
	Function   string   `json:"function,omitempty"`
	Class      string   `json:"class,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Hash       string   `json:"hash"`
	Lines      int      `json:"lines"`
	ErrorType  string   `json:"error_type,omitempty"`
	FixApplied string   `json:"fix_applied,omitempty"`
}

type SearchResult struct {
	Entry    *MemoryEntry `json:"entry"`
	Score    float32      `json:"score"`
	Relevant bool         `json:"relevant"`
}

func NewMemory(rootPath string) *Memory {
	cfg := MemoryConfig{
		MaxEntries: 10000,
		SimilarityThreshold: 0.6,
	}
	
	m := &Memory{
		entries:  make(map[string]*MemoryEntry),
		index:    NewVectorIndex(384),
		cfg:      cfg,
		rootPath: rootPath,
	}
	m.load()
	return m
}

func (m *Memory) Add(entry *MemoryEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry.ID = generateID(entry)
	entry.CreatedAt = time.Now()
	entry.Metadata.Hash = hashContent(entry.Content)
	entry.Embedding = m.index.Embed(entry.Content)

	m.entries[entry.ID] = entry
	m.index.Add(entry.ID, entry.Embedding)

	return m.saveEntry(entry)
}

func (m *Memory) Search(query string, limit int) []SearchResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	queryEmbedding := m.index.Embed(query)
	results := m.index.Search(queryEmbedding, limit)

	var searchResults []SearchResult
	for _, r := range results {
		if entry, ok := m.entries[r.ID]; ok {
			entry.Accessed++
			searchResults = append(searchResults, SearchResult{
				Entry:    entry,
				Score:    r.Score,
				Relevant: r.Score > m.cfg.SimilarityThreshold,
			})
		}
	}

	return searchResults
}

func (m *Memory) RememberError(errorMsg, fix string) error {
	entry := &MemoryEntry{
		Type:    "error_fix",
		Content: fmt.Sprintf("ERROR: %s\nFIX: %s", errorMsg, fix),
		Metadata: MemoryMetadata{
			ErrorType: categorizeError(errorMsg),
			FixApplied: fix,
			Tags: []string{"error", categorizeError(errorMsg)},
		},
		Success: true,
	}
	return m.Add(entry)
}

func (m *Memory) RecallFix(errorMsg string) (string, bool) {
	results := m.Search(errorMsg, 3)
	for _, r := range results {
		if r.Entry.Type == "error_fix" && r.Relevant {
			return r.Entry.Metadata.FixApplied, true
		}
	}
	return "", false
}

func (m *Memory) RememberDecision(decision, rationale string) error {
	entry := &MemoryEntry{
		Type:    "decision",
		Content: fmt.Sprintf("DECISION: %s\nRATIONALE: %s", decision, rationale),
		Metadata: MemoryMetadata{
			Tags: extractTags(decision),
		},
	}
	return m.Add(entry)
}

func (m *Memory) GetContextForQuery(query string) string {
	results := m.Search(query, 5)
	
	var ctx strings.Builder
	for _, r := range results {
		if r.Relevant {
			ctx.WriteString(r.Entry.Content)
			ctx.WriteString("\n---\n")
		}
	}

	return ctx.String()
}

func (m *Memory) Stats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]int{
		"total":     len(m.entries),
		"errors":    0,
		"decisions": 0,
		"code":      0,
	}

	for _, e := range m.entries {
		switch e.Type {
		case "error_fix":
			stats["errors"]++
		case "decision":
			stats["decisions"]++
		case "code":
			stats["code"]++
		}
	}

	return stats
}

func (m *Memory) Prune(maxEntries int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.entries) <= maxEntries {
		return
	}

	var entries []*MemoryEntry
	for _, e := range m.entries {
		entries = append(entries, e)
	}

	sortByAccess(entries)
	
	toDelete := len(m.entries) - maxEntries
	for i := 0; i < toDelete && i < len(entries); i++ {
		delete(m.entries, entries[i].ID)
		m.index.Remove(entries[i].ID)
	}
}

func (m *Memory) load() {
	dir := filepath.Join(m.rootPath, ".siby", "memory")
	os.MkdirAll(dir, 0755)
	
	indexPath := filepath.Join(dir, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return
	}

	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return
	}

	for _, id := range ids {
		if entry := m.loadEntry(id); entry != nil {
			m.entries[id] = entry
			m.index.Add(id, entry.Embedding)
		}
	}
}

func (m *Memory) save() error {
	dir := filepath.Join(m.rootPath, ".siby", "memory")
	os.MkdirAll(dir, 0755)

	ids := make([]string, 0, len(m.entries))
	for id := range m.entries {
		ids = append(ids, id)
	}

	data, _ := json.Marshal(ids)
	return os.WriteFile(filepath.Join(dir, "index.json"), data, 0644)
}

func (m *Memory) saveEntry(entry *MemoryEntry) error {
	dir := filepath.Join(m.rootPath, ".siby", "memory", "entries")
	os.MkdirAll(dir, 0755)

	data, _ := json.Marshal(entry)
	return os.WriteFile(filepath.Join(dir, entry.ID+".json"), data, 0644)
}

func (m *Memory) loadEntry(id string) *MemoryEntry {
	dir := filepath.Join(m.rootPath, ".siby", "memory", "entries")
	data, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		return nil
	}

	var entry MemoryEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil
	}

	entry.Embedding = m.index.Embed(entry.Content)
	return &entry
}

func generateID(entry *MemoryEntry) string {
	h := sha256.New()
	h.Write([]byte(entry.Content + entry.Type + time.Now().Format(time.RFC3339)))
	return hex.EncodeToString(h.Sum(nil))[:12]
}

func hashContent(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))[:8]
}

func categorizeError(err string) string {
	err = strings.ToLower(err)
	switch {
	case strings.Contains(err, "null"), strings.Contains(err, "nil"):
		return "null_pointer"
	case strings.Contains(err, "timeout"):
		return "timeout"
	case strings.Contains(err, "connection"):
		return "connection"
	case strings.Contains(err, "permission"):
		return "permission"
	case strings.Contains(err, "not found"), strings.Contains(err, "enoent"):
		return "not_found"
	case strings.Contains(err, "parse"), strings.Contains(err, "syntax"):
		return "parse_error"
	case strings.Contains(err, "import"), strings.Contains(err, "module"):
		return "import_error"
	case strings.Contains(err, "type"), strings.Contains(err, "cast"):
		return "type_error"
	case strings.Contains(err, "memory"), strings.Contains(err, "oom"), strings.Contains(err, "heap"):
		return "memory_error"
	default:
		return "unknown"
	}
}

func extractTags(content string) []string {
	var tags []string
	words := strings.Fields(content)
	for _, word := range words {
		if len(word) > 3 {
			tags = append(tags, strings.ToLower(word))
		}
	}
	if len(tags) > 10 {
		tags = tags[:10]
	}
	return tags
}

func sortByAccess(entries []*MemoryEntry) {
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Accessed > entries[j].Accessed {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}

type VectorIndex struct {
	dimension int
	vectors   map[string][]float32
}

func NewVectorIndex(dimension int) *VectorIndex {
	return &VectorIndex{
		dimension: dimension,
		vectors:   make(map[string][]float32),
	}
}

func (vi *VectorIndex) Add(id string, embedding []float32) {
	vi.vectors[id] = embedding
}

func (vi *VectorIndex) Remove(id string) {
	delete(vi.vectors, id)
}

func (vi *VectorIndex) Embed(text string) []float32 {
	h := sha256.New()
	h.Write([]byte(text))
	hash := h.Sum(nil)

	vec := make([]float32, vi.dimension)
	for i := 0; i < vi.dimension && i < len(hash); i++ {
		vec[i] = float32(hash[i]) / 255.0
	}
	return vec
}

func (vi *VectorIndex) Search(query []float32, limit int) []struct {
	ID    string
	Score float32
} {
	var results []struct {
		ID    string
		Score float32
	}

	for id, vec := range vi.vectors {
		score := cosineSimilarity(query, vec)
		results = append(results, struct {
			ID    string
			Score float32
		}{id, score})
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

func cosineSimilarity(a, b []float32) float32 {
	var dot, normA, normB float32
	for i := 0; i < len(a) && i < len(b); i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
