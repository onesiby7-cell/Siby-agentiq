package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SessionManager struct {
	mu          sync.RWMutex
	current     *Session
	sessionsDir string
	autoSave    bool
	autoSaveInterval time.Duration
	lastSave    time.Time
	ctx         context.Context
	cancel      context.CancelFunc
}

type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages"`
	Context   SessionContext `json:"context"`
	Metadata  SessionMetadata `json:"metadata"`
}

type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Tokens    int       `json:"tokens,omitempty"`
	Files     []string  `json:"files,omitempty"`
}

type SessionContext struct {
	WorkingDir    string            `json:"working_dir"`
	ProjectPath   string            `json:"project_path"`
	Variables     map[string]string `json:"variables"`
	LastProvider  string            `json:"last_provider"`
	LastModel     string            `json:"last_model"`
}

type SessionMetadata struct {
	TotalTokens     int       `json:"total_tokens"`
	MessageCount    int       `json:"message_count"`
	FilesModified   int       `json:"files_modified"`
	ErrorsCount     int       `json:"errors_count"`
	SuccessCount    int       `json:"success_count"`
}

func NewSessionManager(sessionsDir string) (*SessionManager, error) {
	if sessionsDir == "" {
		home, _ := os.UserHomeDir()
		sessionsDir = filepath.Join(home, ".siby", "sessions")
	}

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	sm := &SessionManager{
		sessionsDir: sessionsDir,
		autoSave:    true,
		autoSaveInterval: 30 * time.Second,
		ctx:    ctx,
		cancel: cancel,
	}

	sm.current = sm.newSession()

	go sm.autoSaveLoop()

	return sm, nil
}

func (sm *SessionManager) newSession() *Session {
	return &Session{
		ID:        uuid.New().String()[:8],
		Name:      fmt.Sprintf("session-%s", time.Now().Format("2006-01-02-15-04")),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages:  make([]Message, 0),
		Context: SessionContext{
			Variables: make(map[string]string),
		},
		Metadata: SessionMetadata{},
	}
}

func (sm *SessionManager) AddMessage(role, content string, tokens int, files []string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	msg := Message{
		ID:        uuid.New().String()[:12],
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Tokens:    tokens,
		Files:     files,
	}

	sm.current.Messages = append(sm.current.Messages, msg)
	sm.current.UpdatedAt = time.Now()
	sm.current.Metadata.MessageCount++
	sm.current.Metadata.TotalTokens += tokens

	if role == "user" && files != nil {
		sm.current.Metadata.FilesModified += len(files)
	}
}

func (sm *SessionManager) RecordSuccess() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.current.Metadata.SuccessCount++
}

func (sm *SessionManager) RecordError() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.current.Metadata.ErrorsCount++
}

func (sm *SessionManager) SetContext(key, value string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.current.Context.Variables[key] = value
}

func (sm *SessionManager) GetContext(key string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current.Context.Variables[key]
}

func (sm *SessionManager) Save() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	filename := filepath.Join(sm.sessionsDir, sm.current.Name+".json")
	data, err := json.MarshalIndent(sm.current, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return err
	}

	sm.lastSave = time.Now()
	return nil
}

func (sm *SessionManager) Load(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	filename := filepath.Join(sm.sessionsDir, name+".json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("session introuvable: %s", name)
	}

	session := &Session{}
	if err := json.Unmarshal(data, session); err != nil {
		return err
	}

	sm.current = session
	return nil
}

func (sm *SessionManager) ListSessions() ([]SessionInfo, error) {
	entries, err := os.ReadDir(sm.sessionsDir)
	if err != nil {
		return nil, err
	}

	sessions := make([]SessionInfo, 0, len(entries))
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		name := entry.Name()[:len(entry.Name())-5]
		info, err := sm.GetSessionInfo(name)
		if err != nil {
			continue
		}
		sessions = append(sessions, *info)
	}

	return sessions, nil
}

func (sm *SessionManager) GetSessionInfo(name string) (*SessionInfo, error) {
	filename := filepath.Join(sm.sessionsDir, name+".json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	session := &Session{}
	if err := json.Unmarshal(data, session); err != nil {
		return nil, err
	}

	return &SessionInfo{
		Name:         session.Name,
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
		MessageCount: session.Metadata.MessageCount,
		TotalTokens:  session.Metadata.TotalTokens,
		SuccessRate:  calculateSuccessRate(session.Metadata),
	}, nil
}

func calculateSuccessRate(m SessionMetadata) float64 {
	total := m.SuccessCount + m.ErrorsCount
	if total == 0 {
		return 0
	}
	return float64(m.SuccessCount) / float64(total) * 100
}

func (sm *SessionManager) Delete(name string) error {
	filename := filepath.Join(sm.sessionsDir, name+".json")
	return os.Remove(filename)
}

func (sm *SessionManager) NewSession() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.current != nil && len(sm.current.Messages) > 0 {
		sm.Save()
	}

	sm.current = sm.newSession()
}

func (sm *SessionManager) GetCurrent() *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

func (sm *SessionManager) GetCurrentMessages() []Message {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current.Messages
}

func (sm *SessionManager) SetWorkingDir(dir string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.current.Context.WorkingDir = dir
}

func (sm *SessionManager) SetProvider(provider, model string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.current.Context.LastProvider = provider
	sm.current.Context.LastModel = model
}

func (sm *SessionManager) autoSaveLoop() {
	ticker := time.NewTicker(sm.autoSaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			if sm.autoSave {
				sm.Save()
			}
		}
	}
}

func (sm *SessionManager) Close() error {
	sm.cancel()
	return sm.Save()
}

func (sm *SessionManager) SetAutoSave(enabled bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.autoSave = enabled
}

type SessionInfo struct {
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
	TotalTokens  int       `json:"total_tokens"`
	SuccessRate  float64   `json:"success_rate"`
}

type InterruptHandler struct {
	mu           sync.RWMutex
	interrupted  bool
	pauseChan    chan bool
	resumeChan   chan bool
	stopChan     chan struct{}
	isPaused     bool
	handlers     []func() error
}

func NewInterruptHandler() *InterruptHandler {
	return &InterruptHandler{
		pauseChan:  make(chan bool, 1),
		resumeChan: make(chan bool, 1),
		stopChan:   make(chan struct{}, 1),
		handlers:   make([]func() error, 0),
	}
}

func (ih *InterruptHandler) AddHandler(handler func() error) {
	ih.mu.Lock()
	defer ih.mu.Unlock()
	ih.handlers = append(ih.handlers, handler)
}

func (ih *InterruptHandler) HandleInterrupt() {
	ih.mu.Lock()
	ih.interrupted = true
	handlers := ih.handlers
	ih.mu.Unlock()

	for _, h := range handlers {
		h()
	}
}

func (ih *InterruptHandler) IsInterrupted() bool {
	ih.mu.RLock()
	defer ih.mu.RUnlock()
	return ih.interrupted
}

func (ih *InterruptHandler) Clear() {
	ih.mu.Lock()
	defer ih.mu.Unlock()
	ih.interrupted = false
}

func (ih *InterruptHandler) Pause() {
	ih.mu.Lock()
	defer ih.mu.Unlock()
	if !ih.isPaused {
		ih.pauseChan <- true
		ih.isPaused = true
	}
}

func (ih *InterruptHandler) Resume() {
	ih.mu.Lock()
	defer ih.mu.Unlock()
	if ih.isPaused {
		ih.resumeChan <- true
		ih.isPaused = false
	}
}

func (ih *InterruptHandler) Stop() {
	close(ih.stopChan)
}

func (ih *InterruptHandler) GetStopChan() <-chan struct{} {
	return ih.stopChan
}

func (ih *InterruptHandler) GetPauseChan() <-chan bool {
	return ih.pauseChan
}

func (ih *InterruptHandler) GetResumeChan() <-chan bool {
	return ih.resumeChan
}
