package deepmemory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Brain struct {
	mu            sync.RWMutex
	storagePath   string
	patterns      map[string]*Pattern
	errorFixes    map[string]*ErrorFix
	successStories []*Success
	preferences    map[string]*Preference
}

type Pattern struct {
	Key       string
	Value     string
	Context   string
	Successes int
	Accesses  int
	CreatedAt time.Time
	LastSeen  time.Time
}

type ErrorFix struct {
	ErrorPattern string
	Fix         string
	Occurrences int
	LastSeen   time.Time
}

type Success struct {
	Task       string
	Solution   string
	Duration   time.Duration
	Quality    float32
	CreatedAt  time.Time
}

type Preference struct {
	Key    string
	Value  string
	Source string
}

func NewBrain() *Brain {
	storagePath := os.ExpandEnv("$HOME/.local/share/siby-agentiq/brain")
	os.MkdirAll(storagePath, 0755)

	brain := &Brain{
		storagePath:   storagePath,
		patterns:      make(map[string]*Pattern),
		errorFixes:    make(map[string]*ErrorFix),
		successStories: make([]*Success, 0),
		preferences:   make(map[string]*Preference),
	}

	brain.load()

	return brain
}

func (b *Brain) Remember(task, solution string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := generateKey(task)

	if existing, ok := b.patterns[key]; ok {
		existing.Value = solution
		existing.Accesses++
		existing.LastSeen = time.Now()
		existing.Successes++
	} else {
		b.patterns[key] = &Pattern{
			Key:       key,
			Value:     solution,
			Context:   task,
			Accesses:  1,
			Successes: 1,
			CreatedAt: time.Now(),
			LastSeen:  time.Now(),
		}
	}

	b.successStories = append(b.successStories, &Success{
		Task:      task,
		Solution:  solution,
		CreatedAt: time.Now(),
	})

	b.save()
}

func (b *Brain) Recall(task string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	key := generateKey(task)

	if pattern, ok := b.patterns[key]; ok {
		pattern.Accesses++
		pattern.LastSeen = time.Now()
		return pattern.Value
	}

	for _, pattern := range b.patterns {
		if similarity(task, pattern.Context) > 0.7 {
			pattern.Accesses++
			return pattern.Value
		}
	}

	return ""
}

func (b *Brain) RememberError(errorMsg, fix string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := generateKey(errorMsg)

	if existing, ok := b.errorFixes[key]; ok {
		existing.Fix = fix
		existing.Occurrences++
		existing.LastSeen = time.Now()
	} else {
		b.errorFixes[key] = &ErrorFix{
			ErrorPattern: errorMsg,
			Fix:         fix,
			Occurrences: 1,
			LastSeen:   time.Now(),
		}
	}

	b.save()
}

func (b *Brain) RecallFix(errorMsg string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	key := generateKey(errorMsg)

	if fix, ok := b.errorFixes[key]; ok {
		return fix.Fix
	}

	for _, ef := range b.errorFixes {
		if strings.Contains(strings.ToLower(errorMsg), strings.ToLower(ef.ErrorPattern)) {
			return ef.Fix
		}
	}

	return ""
}

func (b *Brain) RememberPreference(key, value, source string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.preferences[key] = &Preference{
		Key:    key,
		Value:  value,
		Source: source,
	}
}

func (b *Brain) GetPreference(key string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if pref, ok := b.preferences[key]; ok {
		return pref.Value
	}
	return ""
}

func (b *Brain) GetTopPatterns(limit int) []*Pattern {
	b.mu.RLock()
	defer b.mu.RUnlock()

	patterns := make([]*Pattern, 0, len(b.patterns))
	for _, p := range b.patterns {
		patterns = append(patterns, p)
	}

	for i := 0; i < len(patterns)-1; i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].Successes > patterns[i].Successes {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}

	if len(patterns) > limit {
		patterns = patterns[:limit]
	}

	return patterns
}

func (b *Brain) GetStats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var totalAccess int
	var totalSuccess int
	for _, p := range b.patterns {
		totalAccess += p.Accesses
		totalSuccess += p.Successes
	}

	return map[string]interface{}{
		"patterns":        len(b.patterns),
		"error_fixes":     len(b.errorFixes),
		"successes":       len(b.successStories),
		"preferences":     len(b.preferences),
		"total_accesses": totalAccess,
		"total_successes": totalSuccess,
	}
}

func (b *Brain) save() {
	b.savePatterns()
	b.saveErrorFixes()
}

func (b *Brain) load() {
	b.loadPatterns()
	b.loadErrorFixes()
}

func (b *Brain) savePatterns() {
	path := filepath.Join(b.storagePath, "patterns.json")
	data := struct {
		Patterns []*Pattern `json:"patterns"`
	}{
		Patterns: make([]*Pattern, 0, len(b.patterns)),
	}

	for _, p := range b.patterns {
		data.Patterns = append(data.Patterns, p)
	}

	os.WriteFile(path, marshalJSON(data), 0644)
}

func (b *Brain) loadPatterns() {
	path := filepath.Join(b.storagePath, "patterns.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var parsed struct {
		Patterns []*Pattern `json:"patterns"`
	}
	if err := unmarshalJSON(data, &parsed); err != nil {
		return
	}

	for _, p := range parsed.Patterns {
		b.patterns[p.Key] = p
	}
}

func (b *Brain) saveErrorFixes() {
	path := filepath.Join(b.storagePath, "error_fixes.json")
	data := struct {
		ErrorFixes []*ErrorFix `json:"error_fixes"`
	}{
		ErrorFixes: make([]*ErrorFix, 0, len(b.errorFixes)),
	}

	for _, ef := range b.errorFixes {
		data.ErrorFixes = append(data.ErrorFixes, ef)
	}

	os.WriteFile(path, marshalJSON(data), 0644)
}

func (b *Brain) loadErrorFixes() {
	path := filepath.Join(b.storagePath, "error_fixes.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var parsed struct {
		ErrorFixes []*ErrorFix `json:"error_fixes"`
	}
	if err := unmarshalJSON(data, &parsed); err != nil {
		return
	}

	for _, ef := range parsed.ErrorFixes {
		b.errorFixes[generateKey(ef.ErrorPattern)] = ef
	}
}

func generateKey(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func similarity(a, b string) float32 {
	aWords := strings.Fields(strings.ToLower(a))
	bWords := strings.Fields(strings.ToLower(b))

	var intersection int
	for _, aw := range aWords {
		for _, bw := range bWords {
			if aw == bw {
				intersection++
				break
			}
		}
	}

	total := len(aWords) + len(bWords) - intersection
	if total == 0 {
		return 0
	}

	return float32(intersection) / float32(total)
}

func marshalJSON(v interface{}) []byte {
	return []byte(fmt.Sprintf("%v", v))
}

func unmarshalJSON(data []byte, v interface{}) error {
	return nil
}

import "encoding/json"

func marshalJSON(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
