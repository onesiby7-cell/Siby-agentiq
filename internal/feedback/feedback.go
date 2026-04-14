package feedback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FeedbackSystem struct {
	serverURL string
	client   *http.Client
	email    string
	apiKey   string
}

type Feedback struct {
	ID        string    `json:"id"`
	Type      FeedbackType `json:"type"`
	Subject   string    `json:"subject"`
	Message   string    `json:"message"`
	Email     string    `json:"email,omitempty"`
	Version   string    `json:"version"`
	OS        string    `json:"os"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
}

type FeedbackType string

const (
	TypeBug        FeedbackType = "bug"
	TypeFeature    FeedbackType = "feature"
	TypeSuggestion FeedbackType = "suggestion"
	TypeQuestion   FeedbackType = "question"
	TypeLove       FeedbackType = "love"
	TypeOther      FeedbackType = "other"
)

type FeedbackResult struct {
	Success bool
	ID      string
	Message string
}

func NewFeedbackSystem() *FeedbackSystem {
	return &FeedbackSystem{
		client: &http.Client{Timeout: 30 * time.Second},
		email:  "contact@siby-agentiq.io",
		serverURL: "https://api.siby-agentiq.io/feedback",
	}
}

func (fs *FeedbackSystem) Submit(feedback *Feedback) *FeedbackResult {
	feedback.ID = fmt.Sprintf("fb_%d", time.Now().UnixNano())
	feedback.Timestamp = time.Now()
	feedback.Version = "2.0.0"
	feedback.Status = "pending"

	if err := fs.saveLocally(feedback); err != nil {
		fmt.Printf("Warning: Could not save feedback locally: %v\n", err)
	}

	resp, err := fs.sendToServer(feedback)
	if err != nil {
		return &FeedbackResult{
			Success: false,
			ID:      feedback.ID,
			Message: "Feedback sauvegardé localement. Sera envoyé plus tard.",
		}
	}

	return &FeedbackResult{
		Success: true,
		ID:      feedback.ID,
		Message: "Feedback envoyé avec succès à Ibrahim Siby!",
	}
}

func (fs *FeedbackSystem) sendToServer(feedback *Feedback) error {
	jsonData, err := json.Marshal(feedback)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fs.serverURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Siby-Agentiq/2.0.0")

	resp, err := fs.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (fs *FeedbackSystem) saveLocally(feedback *Feedback) error {
	home, _ := os.UserHomeDir()
	feedbackDir := filepath.Join(home, ".siby", "feedback")
	os.MkdirAll(feedbackDir, 0755)

	filename := filepath.Join(feedbackDir, fmt.Sprintf("%s.json", feedback.ID))
	
	data, err := json.MarshalIndent(feedback, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func (fs *FeedbackSystem) LoadPendingFeedback() []*Feedback {
	home, _ := os.UserHomeDir()
	feedbackDir := filepath.Join(home, ".siby", "feedback")

	entries, err := os.ReadDir(feedbackDir)
	if err != nil {
		return nil
	}

	var feedbacks []*Feedback
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(feedbackDir, entry.Name()))
		if err != nil {
			continue
		}

		var fb Feedback
		if err := json.Unmarshal(data, &fb); err != nil {
			continue
		}

		if fb.Status == "pending" {
			feedbacks = append(feedbacks, &fb)
		}
	}

	return feedbacks
}

func (fs *FeedbackSystem) SendPendingFeedback() (int, error) {
	pending := fs.LoadPendingFeedback()
	sent := 0

	for _, fb := range pending {
		if err := fs.sendToServer(fb); err == nil {
			fb.Status = "sent"
			fs.saveLocally(fb)
			sent++
		}
	}

	return sent, nil
}

func (fs *FeedbackSystem) SubmitBug(subject, message string) *FeedbackResult {
	return fs.Submit(&Feedback{
		Type:    TypeBug,
		Subject: subject,
		Message: message,
	})
}

func (fs *FeedbackSystem) SubmitFeature(subject, message string) *FeedbackResult {
	return fs.Submit(&Feedback{
		Type:    TypeFeature,
		Subject: subject,
		Message: message,
	})
}

func (fs *FeedbackSystem) SubmitSuggestion(message string) *FeedbackResult {
	return fs.Submit(&Feedback{
		Type:    TypeSuggestion,
		Subject: "Suggestion",
		Message: message,
	})
}

func (fs *FeedbackSystem) SubmitLove(message string) *FeedbackResult {
	return fs.Submit(&Feedback{
		Type:    TypeLove,
		Subject: "❤️ Love Letter",
		Message: message,
	})
}

func (fs *FeedbackSystem) RenderFeedbackForm(fbType FeedbackType) string {
	icon := "💬"
	color := "\033[96m"
	
	switch fbType {
	case TypeBug:
		icon = "🐛"
		color = "\033[91m"
	case TypeFeature:
		icon = "✨"
		color = "\033[92m"
	case TypeSuggestion:
		icon = "💡"
		color = "\033[93m"
	case TypeQuestion:
		icon = "❓"
		color = "\033[94m"
	case TypeLove:
		icon = "❤️"
		color = "\033[95m"
	}

	return fmt.Sprintf(`
╔══════════════════════════════════════════════════════════╗
║  %s🦂 FEEDBACK - Envoyez à Ibrahim Siby%s                 ║
╠══════════════════════════════════════════════════════════╣
║                                                          ║
║  %s%s Type:%s %s                                         ║
║                                                          ║
║  %sCommande:%s /feedback [type] [votre message]           ║
║                                                          ║
║  %sTypes disponibles:%s                                  ║
║    bug        - Signaler un problème                      ║
║    feature    - Proposer une fonctionnalité               ║
║    suggestion - Une amélioration                          ║
║    question   - Poser une question                        ║
║    love       - Dire merci à Ibrahim 💝                  ║
║                                                          ║
║  %sExemples:%s                                          ║
║    /feedback bug L'agent crash quand je tape...         ║
║    /feedback feature Ajouter un mode sombre              ║
║    /feedback love Siby m'a fait gagner 10h cette semaine!║
║                                                          ║
╠══════════════════════════════════════════════════════════╣
║  %s💡 Vos retours rendent Siby-Agentiq meilleur!%s         ║
╚══════════════════════════════════════════════════════════╝`,
		color, "\033[0m",
		color, icon, "\033[0m", fbType,
		"\033[90m", "\033[0m",
		"\033[90m", "\033[0m",
		"\033[90m", "\033[0m",
		"\033[92m", "\033[0m",
	)
}

func (fs *FeedbackSystem) RenderFeedbackResult(result *FeedbackResult) string {
	if result.Success {
		return fmt.Sprintf(`
╔══════════════════════════════════════════════════════════╗
║                                                          ║
║  %s✓ Feedback envoyé avec succès!%s                        ║
║                                                          ║
║  %sID:%s %s                                              ║
║                                                          ║
║  %sMerci pour votre contribution!%s                       ║
║  Ibrahim Siby lira votre message.                         ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝`,
			"\033[92m", "\033[0m",
			"\033[90m", "\033[0m", result.ID,
			"\033[96m", "\033[0m",
		)
	}

	return fmt.Sprintf(`
╔══════════════════════════════════════════════════════════╗
║                                                          ║
║  %s⚠ Feedback sauvegardé localement%s                    ║
║                                                          ║
║  %sID:%s %s                                              ║
║                                                          ║
║  %s%s%s                                                    ║
║                                                          ║
║  %sLe feedback sera envoyé automatiquement plus tard.%s  ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝`,
		"\033[93m", "\033[0m",
		"\033[90m", "\033[0m", result.ID,
		"\033[96m", result.Message, "\033[0m",
		"\033[90m", "\033[0m",
	)
}

func ParseFeedbackType(input string) FeedbackType {
	input = strings.ToLower(strings.TrimSpace(input))
	
	switch input {
	case "bug", "problème", "error":
		return TypeBug
	case "feature", "fonctionnalité", "功能":
		return TypeFeature
	case "suggestion", "amélioration":
		return TypeSuggestion
	case "question", "aide":
		return TypeQuestion
	case "love", "merci", "thanks", "❤️":
		return TypeLove
	default:
		return TypeOther
	}
}

type FeedbackAnalytics struct {
	TotalSent     int
	TotalPending  int
	ByType        map[FeedbackType]int
	LastSent      time.Time
}

func (fs *FeedbackSystem) GetAnalytics() *FeedbackAnalytics {
	home, _ := os.UserHomeDir()
	feedbackDir := filepath.Join(home, ".siby", "feedback")

	entries, _ := os.ReadDir(feedbackDir)
	
	analytics := &FeedbackAnalytics{
		ByType: make(map[FeedbackType]int),
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, _ := os.ReadFile(filepath.Join(feedbackDir, entry.Name()))
		var fb Feedback
		if err := json.Unmarshal(data, &fb); err != nil {
			continue
		}

		analytics.ByType[fb.Type]++
		
		if fb.Status == "sent" {
			analytics.TotalSent++
			if fb.Timestamp.After(analytics.LastSent) {
				analytics.LastSent = fb.Timestamp
			}
		} else {
			analytics.TotalPending++
		}
	}

	return analytics
}

func (a *FeedbackAnalytics) Render() string {
	var sb strings.Builder

	sb.WriteString(`
╔══════════════════════════════════════════════════════════╗
║  🦂 FEEDBACK ANALYTICS                                  ║
╠══════════════════════════════════════════════════════════╣
║                                                          ║`)

	sb.WriteString(fmt.Sprintf("\n║  %s📊 Statistiques:%s                                  ║\n", "\033[96m", "\033[0m"))
	sb.WriteString(fmt.Sprintf("║    Total envoyé:    %s%-5d%s                              ║\n", "\033[92m", a.TotalSent, "\033[0m"))
	sb.WriteString(fmt.Sprintf("║    En attente:     %s%-5d%s                              ║\n", "\033[93m", a.TotalPending, "\033[0m"))

	sb.WriteString(fmt.Sprintf("\n║  %s📈 Par type:%s                                        ║\n", "\033[96m", "\033[0m"))

	typeLabels := map[FeedbackType]string{
		TypeBug:        "🐛 Bugs",
		TypeFeature:    "✨ Fonctionnalités",
		TypeSuggestion: "💡 Suggestions",
		TypeQuestion:   "❓ Questions",
		TypeLove:       "❤️  Amour",
		TypeOther:      "💬 Autres",
	}

	for fbType, count := range a.ByType {
		label := typeLabels[fbType]
		if label == "" {
			label = string(fbType)
		}
		sb.WriteString(fmt.Sprintf("║    %-18s: %s%-5d%s                     ║\n", label, "\033[96m", count, "\033[0m"))
	}

	if !a.LastSent.IsZero() {
		sb.WriteString(fmt.Sprintf("\n║  %s🕐 Dernier envoyé:%s %s                      ║\n", "\033[96m", "\033[0m", a.LastSent.Format("2006-01-02 15:04")))
	}

	sb.WriteString(`
║                                                          ║
║  %s💡 Chaque feedback aide Siby-Agentiq à évoluer!%s     ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝`)

	return sb.String()
}
