package deepmemory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type PersistentStore struct {
	mu        sync.RWMutex
	storePath string
	history   []*HistoryEntry
}

type HistoryEntry struct {
	ID        int64     `json:"id"`
	Query     string    `json:"query"`
	Solution  string    `json:"solution"`
	Timestamp time.Time `json:"timestamp"`
	Tags      string    `json:"tags"`
}

var persistentStore *PersistentStore

func InitPersistentStore() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	sibyDir := filepath.Join(homeDir, ".siby-agentiq")
	if err := os.MkdirAll(sibyDir, 0755); err != nil {
		return fmt.Errorf("failed to create siby directory: %w", err)
	}

	persistentStore = &PersistentStore{
		storePath: filepath.Join(sibyDir, "history.json"),
		history:   make([]*HistoryEntry, 0),
	}

	persistentStore.load()
	return nil
}

func (p *PersistentStore) load() {
	data, err := os.ReadFile(p.storePath)
	if err != nil {
		return
	}

	if err := json.Unmarshal(data, &p.history); err != nil {
		return
	}
}

func (p *PersistentStore) save() error {
	data, err := json.MarshalIndent(p.history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.storePath, data, 0644)
}

func (p *PersistentStore) AddSuccess(query, solution, tags string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry := &HistoryEntry{
		ID:        time.Now().UnixMilli(),
		Query:     query,
		Solution:  solution,
		Timestamp: time.Now(),
		Tags:      tags,
	}

	p.history = append([]*HistoryEntry{entry}, p.history...)

	if len(p.history) > 1000 {
		p.history = p.history[:1000]
	}

	return p.save()
}

func (p *PersistentStore) FindSimilar(query string) (*HistoryEntry, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	queryLower := strings.ToLower(query)

	for _, entry := range p.history {
		entryLower := strings.ToLower(entry.Query)
		if strings.Contains(entryLower, queryLower) ||
			strings.Contains(queryLower, entryLower) {
			return entry, nil
		}
	}

	return nil, nil
}

func (p *PersistentStore) GetRecent(limit int) []*HistoryEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.history) <= limit {
		return p.history
	}
	return p.history[:limit]
}

func (p *PersistentStore) SearchByTag(tag string) []*HistoryEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var results []*HistoryEntry
	for _, entry := range p.history {
		if strings.Contains(strings.ToLower(entry.Tags), strings.ToLower(tag)) {
			results = append(results, entry)
		}
	}
	return results
}

func GetPersistentStore() *PersistentStore {
	return persistentStore
}

func RemindSimilarPast(query string) (string, bool) {
	if persistentStore == nil {
		return "", false
	}

	entry, err := persistentStore.FindSimilar(query)
	if err != nil || entry == nil {
		return "", false
	}

	reminder := fmt.Sprintf(
		"\n🦂 [DEEPMEMORY] Ibrahim, la derniere fois on a regle ce bug de cette maniere:\n\n%s\n\nVoulez-vous que je recommence?\n",
		entry.Solution,
	)

	return reminder, true
}
