package core

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

type NeuralMemory struct {
	LongTermStorage map[string]*MemoryBlock
	SuccessStories  []*SuccessRecord
	FailurePatterns []*FailurePattern
	LearningQueue   chan *LearningInput
}

type MemoryBlock struct {
	Key         string
	Value       string
	Context     string
	SuccessRate float32
	AccessCount int
	LastAccess time.Time
	CreatedAt   time.Time
	Tags        []string
}

type SuccessRecord struct {
	Task        string
	Solution    string
	TimeSpent   time.Duration
	Efficiency  float32
	Metadata    map[string]string
}

type FailurePattern struct {
	ErrorType    string
	Occurrences  int
	LastSeen    time.Time
	CommonFixes []string
	Severity    int
}

type LearningInput struct {
	Task       string
	Success    bool
	Result     string
	Duration   time.Duration
	Metadata   map[string]string
}

func NewNeuralMemory() *NeuralMemory {
	nm := &NeuralMemory{
		LongTermStorage: make(map[string]*MemoryBlock),
		SuccessStories:   make([]*SuccessRecord, 0),
		FailurePatterns: make([]*FailurePattern, 0),
		LearningQueue:   make(chan *LearningInput, 100),
	}
	go nm.learningLoop()
	return nm
}

func (nm *NeuralMemory) Learn(input *LearningInput) {
	nm.LearningQueue <- input
}

func (nm *NeuralMemory) learningLoop() {
	for input := range nm.LearningQueue {
		nm.processLearning(input)
	}
}

func (nm *NeuralMemory) processLearning(input *LearningInput) {
	block := &MemoryBlock{
		Key:         generateKey(input.Task),
		Value:       input.Result,
		Context:     input.Task,
		SuccessRate: 0.5,
		AccessCount: 0,
		LastAccess:  time.Now(),
		CreatedAt:   time.Now(),
		Tags:        extractTags(input.Task),
	}

	if input.Success {
		block.SuccessRate = 1.0
		nm.SuccessStories = append(nm.SuccessStories, &SuccessRecord{
			Task:       input.Task,
			Solution:   input.Result,
			TimeSpent:  input.Duration,
			Efficiency: calculateEfficiency(input),
			Metadata:   input.Metadata,
		})
	}

	nm.LongTermStorage[block.Key] = block
}

func (nm *NeuralMemory) Recall(task string) (string, bool) {
	key := generateKey(task)
	if block, ok := nm.LongTermStorage[key]; ok {
		block.AccessCount++
		block.LastAccess = time.Now()
		return block.Value, true
	}

	for _, block := range nm.LongTermStorage {
		if similarity(task, block.Context) > 0.8 {
			block.AccessCount++
			block.LastAccess = time.Now()
			return block.Value, true
		}
	}

	return "", false
}

func (nm *NeuralMemory) GetSuccessPattern(task string) *SuccessRecord {
	bestMatch := float32(0)
	var bestRecord *SuccessRecord

	for _, record := range nm.SuccessStories {
		sim := similarity(task, record.Task)
		if sim > bestMatch {
			bestMatch = sim
			bestRecord = record
		}
	}

	if bestMatch > 0.6 {
		return bestRecord
	}

	return nil
}

func (nm *NeuralMemory) GetFailureWarning(task string) []*FailurePattern {
	var warnings []*FailurePattern

	for _, pattern := range nm.FailurePatterns {
		if containsPattern(task, pattern.ErrorType) {
			warnings = append(warnings, pattern)
		}
	}

	return warnings
}

func generateKey(task string) string {
	h := sha256.New()
	h.Write([]byte(task))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func extractTags(task string) []string {
	words := strings.Fields(task)
	var tags []string
	for _, word := range words {
		if len(word) > 3 {
			tags = append(tags, strings.ToLower(word))
		}
	}
	return tags
}

func calculateEfficiency(input *LearningInput) float32 {
	baseEfficiency := float32(1.0)
	
	if input.Duration > 5*time.Minute {
		baseEfficiency *= 0.8
	}
	if input.Duration > 15*time.Minute {
		baseEfficiency *= 0.6
	}

	if strings.Contains(strings.ToLower(input.Result), "optimal") {
		baseEfficiency *= 1.2
	}

	return minFloat(baseEfficiency, 1.0)
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

func containsPattern(task, pattern string) bool {
	return strings.Contains(strings.ToLower(task), strings.ToLower(pattern))
}

func minFloat(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
