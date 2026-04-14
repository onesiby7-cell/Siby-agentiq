package evolution

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	EvolutionGold    = "\033[93m"
	EvolutionCyan    = "\033[96m"
	EvolutionGreen   = "\033[92m"
	EvolutionPurple  = "\033[95m"
	EvolutionReset   = "\033[0m"
)

type EvolutionCore struct {
	mu           sync.RWMutex
	lessons      map[string]*Lesson
	performance  *PerformanceMetrics
	knowledgeDB  *KnowledgeDB
	prompts      *PromptLibrary
	workDir      string
	enabled      bool
	nightlyMode  bool
}

type Lesson struct {
	ID          string    `json:"id"`
	Type        LessonType `json:"type"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Context     string    `json:"context"`
	Solution    string    `json:"solution"`
	Confidence  float64   `json:"confidence"`
	UsageCount  int       `json:"usage_count"`
	Tags        []string  `json:"tags"`
	Source      string    `json:"source"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used"`
	SuccessRate float64   `json:"success_rate"`
}

type LessonType string

const (
	LessonBugFix     LessonType = "bug_fix"
	LessonPattern    LessonType = "pattern"
	LessonDoc        LessonType = "documentation"
	LessonAPI        LessonType = "api_usage"
	LessonPerformance LessonType = "performance"
	LessonSecurity   LessonType = "security"
)

type KnowledgeDB struct {
	entries map[string]*KnowledgeEntry
	mu      sync.RWMutex
	index   *SearchIndex
}

type KnowledgeEntry struct {
	ID        string    `json:"id"`
	Query     string    `json:"query"`
	Answer    string    `json:"answer"`
	Embedding []float32 `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"created_at"`
}

type SearchIndex struct {
	keywords map[string][]string
	mu       sync.RWMutex
}

type PerformanceMetrics struct {
	TotalInteractions int                `json:"total_interactions"`
	SuccessCount      int                `json:"success_count"`
	FailureCount      int                `json:"failure_count"`
	DailyMetrics      map[string]*DailyMetric `json:"daily_metrics"`
	AvgResponseTime   time.Duration      `json:"avg_response_time"`
	TopicsMastered    map[string]float64 `json:"topics_mastered"`
}

type DailyMetric struct {
	Date            string    `json:"date"`
	Interactions    int       `json:"interactions"`
	Successes       int       `json:"successes"`
	Failures        int       `json:"failures"`
	TopicsLearned   int       `json:"topics_learned"`
	ResponseTimeAvg time.Duration `json:"response_time_avg"`
}

type PromptLibrary struct {
	prompts map[string]*OptimizedPrompt
	mu      sync.RWMutex
}

type OptimizedPrompt struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	Original   string  `json:"original"`
	Optimized  string  `json:"optimized"`
	UsageCount int     `json:"usage_count"`
	SuccessRate float64 `json:"success_rate"`
	Iterations int     `json:"iterations"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewEvolutionCore(workDir string) *EvolutionCore {
	ec := &EvolutionCore{
		lessons:     make(map[string]*Lesson),
		performance: &PerformanceMetrics{
			DailyMetrics:   make(map[string]*DailyMetric),
			TopicsMastered: make(map[string]float64),
		},
		knowledgeDB: &KnowledgeDB{
			entries: make(map[string]*KnowledgeEntry),
			index:   &SearchIndex{keywords: make(map[string][]string)},
		},
		prompts: &PromptLibrary{
			prompts: make(map[string]*OptimizedPrompt),
		},
		workDir:  workDir,
		enabled: true,
	}

	ec.loadFromDisk()
	return ec
}

func (ec *EvolutionCore) LearnFromBug(ctx context.Context, bug BugReport) *Lesson {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	lesson := &Lesson{
		ID:        uuid.New().String(),
		Type:      LessonBugFix,
		Title:     bug.Title,
		Content:   bug.Description,
		Solution:  bug.Solution,
		Context:   bug.Context,
		Tags:      bug.Tags,
		Source:    "scorpion",
		CreatedAt: time.Now(),
		Confidence: 0.5,
		SuccessRate: 0.0,
	}

	ec.lessons[lesson.ID] = lesson
	ec.indexLesson(lesson)
	ec.saveToDisk()

	ec.updateTopicMastery(bug.Tags, 0.1)

	return lesson
}

func (ec *EvolutionCore) LearnFromDocumentation(ctx context.Context, doc DocEntry) *Lesson {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	lesson := &Lesson{
		ID:        uuid.New().String(),
		Type:      LessonDoc,
		Title:     doc.Title,
		Content:   doc.Content,
		Solution:  doc.Summary,
		Tags:      doc.Tags,
		Source:    "scorpion",
		CreatedAt: time.Now(),
		Confidence: 0.8,
		SuccessRate: 0.0,
	}

	ec.lessons[lesson.ID] = lesson
	ec.indexLesson(lesson)
	ec.saveToDisk()

	return lesson
}

func (ec *EvolutionCore) StoreScorpionLearning(ctx context.Context, query, answer string, sources []string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	lesson := &Lesson{
		ID:        uuid.New().String(),
		Type:      LessonAPI,
		Title:     query,
		Content:   answer,
		Tags:      extractTags(query),
		Source:    "scorpion",
		CreatedAt: time.Now(),
		Confidence: 0.7,
		SuccessRate: 0.0,
	}

	ec.lessons[lesson.ID] = lesson
	ec.indexLesson(lesson)

	entry := &KnowledgeEntry{
		ID:        uuid.New().String(),
		Query:     query,
		Answer:    answer,
		Metadata:  map[string]interface{}{"sources": sources},
		CreatedAt: time.Now(),
	}
	ec.knowledgeDB.entries[entry.ID] = entry

	ec.saveToDisk()
}

func (ec *EvolutionCore) RetrieveLesson(ctx context.Context, query string) *Lesson {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	tags := extractTags(query)
	
	var bestMatch *Lesson
	var bestScore float64

	for _, lesson := range ec.lessons {
		score := ec.calculateMatchScore(lesson, tags, query)
		if score > bestScore && score > 0.3 {
			bestScore = score
			bestMatch = lesson
		}
	}

	if bestMatch != nil {
		bestMatch.UsageCount++
		bestMatch.LastUsed = time.Now()
	}

	return bestMatch
}

func (ec *EvolutionCore) calculateMatchScore(lesson *Lesson, queryTags []string, query string) float64 {
	var score float64

	for _, qt := range queryTags {
		for _, lt := range lesson.Tags {
			if strings.Contains(strings.ToLower(qt), strings.ToLower(lt)) ||
				strings.Contains(strings.ToLower(lt), strings.ToLower(qt)) {
				score += 0.4
			}
		}
	}

	if strings.Contains(strings.ToLower(query), strings.ToLower(lesson.Title)) {
		score += 0.3
	}

	score += lesson.Confidence * 0.2

	if lesson.UsageCount > 0 {
		usageFactor := float64(lesson.UsageCount) / float64(lesson.UsageCount+10)
		score += usageFactor * 0.1
	}

	return score
}

func (ec *EvolutionCore) indexLesson(lesson *Lesson) {
	for _, tag := range lesson.Tags {
		tagLower := strings.ToLower(tag)
		if _, ok := ec.knowledgeDB.index.keywords[tagLower]; !ok {
			ec.knowledgeDB.index.keywords[tagLower] = []string{}
		}
		ec.knowledgeDB.index.keywords[tagLower] = append(
			ec.knowledgeDB.index.keywords[tagLower],
			lesson.ID,
		)
	}
}

func (ec *EvolutionCore) RecordPerformance(success bool, topic string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	
	if _, ok := ec.performance.DailyMetrics[today]; !ok {
		ec.performance.DailyMetrics[today] = &DailyMetric{
			Date: today,
		}
	}

	ec.performance.TotalInteractions++
	ec.performance.DailyMetrics[today].Interactions++

	if success {
		ec.performance.SuccessCount++
		ec.performance.DailyMetrics[today].Successes++
	} else {
		ec.performance.FailureCount++
		ec.performance.DailyMetrics[today].Failures++
	}

	if topic != "" {
		ec.updateTopicMastery([]string{topic}, 0.05)
	}
}

func (ec *EvolutionCore) updateTopicMastery(topics []string, delta float64) {
	for _, topic := range topics {
		if current, ok := ec.performance.TopicsMastered[topic]; ok {
			ec.performance.TopicsMastered[topic] = min(current+delta, 1.0)
		} else {
			ec.performance.TopicsMastered[topic] = delta
		}
	}
}

func (ec *EvolutionCore) RunNightlyOptimization() *OptimizationReport {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	report := &OptimizationReport{
		Date:        time.Now(),
		Optimizations: make([]PromptOptimization, 0),
		Statistics:   make(map[string]interface{}),
	}

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if metrics, ok := ec.performance.DailyMetrics[yesterday]; ok {
		report.Statistics["yesterday_interactions"] = metrics.Interactions
		report.Statistics["yesterday_success_rate"] = float64(metrics.Successes) / float64(max(metrics.Interactions, 1))
	}

	report.Optimizations = ec.analyzeAndOptimizePrompts()
	report.LessonsLearned = ec.summarizeLessons()
	report.TopicsProgress = ec.performance.TopicsMastered

	ec.saveOptimizationReport(report)

	return report
}

func (ec *EvolutionCore) analyzeAndOptimizePrompts() []PromptOptimization {
	optimizations := make([]PromptOptimization, 0)

	for _, lesson := range ec.lessons {
		if lesson.SuccessRate < 0.7 && lesson.UsageCount >= 3 {
			opt := PromptOptimization{
				LessonID:   lesson.ID,
				Original:   lesson.Content,
				Suggestion: ec.generatePromptImprovement(lesson),
				Reason:     "Low success rate detected",
			}
			optimizations = append(optimizations, opt)
			lesson.Confidence = min(lesson.Confidence+0.1, 1.0)
		}
	}

	return optimizations
}

func (ec *EvolutionCore) generatePromptImprovement(lesson *Lesson) string {
	return fmt.Sprintf(`Optimized version of "%s":

1. Clarify context requirements
2. Add error handling examples
3. Include validation steps
4. Add success criteria

Previous attempts: %d
Success rate: %.0f%%`, lesson.Title, lesson.UsageCount, lesson.SuccessRate*100)
}

func (ec *EvolutionCore) summarizeLessons() []string {
	lessons := make([]string, 0, 10)

	typeCount := make(map[LessonType]int)
	for _, lesson := range ec.lessons {
		typeCount[lesson.Type]++
	}

	for lt, count := range typeCount {
		lessons = append(lessons, fmt.Sprintf("Learned %d %s lessons", count, lt))
	}

	return lessons
}

func (ec *EvolutionCore) EnableNightlyMode(enabled bool) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.nightlyMode = enabled
}

func (ec *EvolutionCore) IsNightlyModeEnabled() bool {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.nightlyMode
}

func (ec *EvolutionCore) saveToDisk() {
	dataDir := filepath.Join(ec.workDir, ".siby", "evolution")
	os.MkdirAll(dataDir, 0755)

	lessonsFile := filepath.Join(dataDir, "lessons.json")
	data, _ := json.MarshalIndent(ec.lessons, "", "  ")
	os.WriteFile(lessonsFile, data, 0644)

	perfFile := filepath.Join(dataDir, "performance.json")
	perfData, _ := json.MarshalIndent(ec.performance, "", "  ")
	os.WriteFile(perfFile, perfData, 0644)
}

func (ec *EvolutionCore) loadFromDisk() {
	dataDir := filepath.Join(ec.workDir, ".siby", "evolution")

	lessonsFile := filepath.Join(dataDir, "lessons.json")
	if data, err := os.ReadFile(lessonsFile); err == nil {
		json.Unmarshal(data, &ec.lessons)
		for _, lesson := range ec.lessons {
			ec.indexLesson(lesson)
		}
	}

	perfFile := filepath.Join(dataDir, "performance.json")
	if data, err := os.ReadFile(perfFile); err == nil {
		json.Unmarshal(data, &ec.performance)
	}
}

func (ec *EvolutionCore) saveOptimizationReport(report *OptimizationReport) {
	dataDir := filepath.Join(ec.workDir, ".siby", "evolution", "reports")
	os.MkdirAll(dataDir, 0755)

	reportFile := filepath.Join(dataDir, fmt.Sprintf("report_%s.json", report.Date.Format("2006-01-02")))
	data, _ := json.MarshalIndent(report, "", "  ")
	os.WriteFile(reportFile, data, 0644)
}

func (ec *EvolutionCore) RenderEvolutionStatus() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s%s\n", EvolutionGold, strings.Repeat("═", 70)))
	sb.WriteString(fmt.Sprintf("%s  🧬 EVOLUTION-CORE STATUS 🧬%s\n", EvolutionCyan, EvolutionReset))
	sb.WriteString(fmt.Sprintf("%s%s\n\n", EvolutionGold, strings.Repeat("═", 70)))

	sb.WriteString(fmt.Sprintf("  %s📊 LEÇONS APPRISES:%s\n", EvolutionPurple, EvolutionReset))
	
	typeCount := make(map[LessonType]int)
	for _, lesson := range ec.lessons {
		typeCount[lesson.Type]++
	}
	
	lessonTypes := []struct {
		t     LessonType
		label string
	}{
		{LessonBugFix, "🐛 Bug Fixes"},
		{LessonPattern, "📋 Patterns"},
		{LessonDoc, "📚 Documentation"},
		{LessonAPI, "🔌 API Usage"},
		{LessonPerformance, "⚡ Performance"},
		{LessonSecurity, "🔒 Security"},
	}
	
	for _, lt := range lessonTypes {
		count := typeCount[lt.t]
		bar := strings.Repeat("█", min(count, 30))
		sb.WriteString(fmt.Sprintf("  %s: %s (%d)\n", lt.label, bar, count))
	}

	sb.WriteString(fmt.Sprintf("\n  %s📈 PERFORMANCES:%s\n", EvolutionPurple, EvolutionReset))
	successRate := float64(ec.performance.SuccessCount) / float64(max(ec.performance.TotalInteractions, 1))
	sb.WriteString(fmt.Sprintf("  Taux de succès: %s%.1f%% %s\n", 
		EvolutionGreen, successRate*100, EvolutionReset))
	sb.WriteString(fmt.Sprintf("  Interactions totales: %d\n", ec.performance.TotalInteractions))

	sb.WriteString(fmt.Sprintf("\n  %s🧠 TOPICS MAÎTRISÉS:%s\n", EvolutionPurple, EvolutionReset))
	for topic, mastery := range ec.performance.TopicsMastered {
		bar := strings.Repeat("█", int(mastery*20))
		sb.WriteString(fmt.Sprintf("  %s: %s %.0f%%\n", topic, bar, mastery*100))
	}

	sb.WriteString(fmt.Sprintf("\n  %s🌙 MODE NOCTURNE:%s %s\n", EvolutionPurple, EvolutionReset, 
		map[bool]string{true: EvolutionGreen + "ACTIVÉ" + EvolutionReset, false: EvolutionRed + "DÉSACTIVÉ" + EvolutionReset}[ec.nightlyMode]))

	sb.WriteString(fmt.Sprintf("\n%s%s\n", EvolutionGold, strings.Repeat("═", 70)))
	sb.WriteString(fmt.Sprintf("  %sCréé par Ibrahim Siby • Auto-apprentissage récursif actif 🦂%s\n", EvolutionCyan, EvolutionReset))
	sb.WriteString(fmt.Sprintf("%s%s\n", EvolutionGold, strings.Repeat("═", 70)))

	return sb.String()
}

type BugReport struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Solution    string   `json:"solution"`
	Context     string   `json:"context"`
	Tags        []string `json:"tags"`
}

type DocEntry struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Summary  string   `json:"summary"`
	Tags     []string `json:"tags"`
}

type OptimizationReport struct {
	Date          time.Time          `json:"date"`
	Optimizations []PromptOptimization `json:"optimizations"`
	LessonsLearned []string           `json:"lessons_learned"`
	TopicsProgress map[string]float64 `json:"topics_progress"`
	Statistics    map[string]interface{} `json:"statistics"`
}

type PromptOptimization struct {
	LessonID   string `json:"lesson_id"`
	Original   string `json:"original"`
	Suggestion string `json:"suggestion"`
	Reason     string `json:"reason"`
}

const EvolutionRed = "\033[91m"

func extractTags(query string) []string {
	keywords := []string{
		"go", "golang", "javascript", "typescript", "python", "react", "vue",
		"api", "database", "sql", "mongodb", "redis", "docker", "kubernetes",
		"bug", "error", "fix", "performance", "security", "auth", "oauth",
		"test", "deployment", "ci", "cd", "microservice", "graphql", "rest",
	}

	queryLower := strings.ToLower(query)
	tags := make([]string, 0)

	for _, kw := range keywords {
		if strings.Contains(queryLower, kw) {
			tags = append(tags, kw)
		}
	}

	return tags
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
