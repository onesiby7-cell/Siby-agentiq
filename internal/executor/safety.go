package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SafetyManager struct {
	cfg ExecutorConfig
}

type SafetyResult struct {
	BackupPath string
	Approved   bool
}

type SafetyPolicy struct {
	AllowCreate  bool
	AllowModify  bool
	AllowDelete  bool
	AllowExecute bool
}

var DefaultPolicy = SafetyPolicy{
	AllowCreate:  true,
	AllowModify:  true,
	AllowDelete:  false,
	AllowExecute: false,
}

func NewSafetyManager(cfg ExecutorConfig) *SafetyManager {
	if cfg.MaxFileSize == 0 {
		cfg.MaxFileSize = 1024 * 1024
	}
	return &SafetyManager{cfg: cfg}
}

func (s *SafetyManager) ExecuteWithProtection(change FileChange, fn func(FileChange) error) (*SafetyResult, error) {
	result := &SafetyResult{}

	switch change.Action {
	case ActionCreate:
		if !DefaultPolicy.AllowCreate {
			return result, fmt.Errorf("create not allowed by policy")
		}

	case ActionModify:
		if !DefaultPolicy.AllowModify {
			return result, fmt.Errorf("modify not allowed by policy")
		}

		backupPath, err := CreateBackup(change.Path)
		if err != nil {
			return result, fmt.Errorf("backup failed: %w", err)
		}
		result.BackupPath = backupPath

	case ActionDelete:
		if !DefaultPolicy.AllowDelete {
			return result, fmt.Errorf("delete not allowed by policy")
		}

		backupPath, err := CreateBackup(change.Path)
		if err != nil {
			return result, fmt.Errorf("backup before delete failed: %w", err)
		}
		result.BackupPath = backupPath
	}

	if err := fn(change); err != nil {
		if result.BackupPath != "" {
			RestoreBackup(change.Path)
		}
		return result, fmt.Errorf("execution failed: %w", err)
	}

	return result, nil
}

func (s *SafetyManager) ValidateContent(path, content string) error {
	if len(content) > int(s.cfg.MaxFileSize) {
		return fmt.Errorf("content exceeds max size (%d bytes)", s.cfg.MaxFileSize)
	}

	dangerousPatterns := []string{
		"rm -rf /",
		":(){ :|:& };:",
		"fork bomb",
		"curl .* | sh",
		"wget .* | sh",
	}

	contentLower := strings.ToLower(content)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(contentLower, pattern) {
			return fmt.Errorf("content contains dangerous pattern: %s", pattern)
		}
	}

	ext := strings.ToLower(filepath.Ext(path))
	dangerousExts := []string{".exe", ".dll", ".so", ".dylib"}
	for _, de := range dangerousExts {
		if ext == de {
			return fmt.Errorf("extension %s not allowed", ext)
		}
	}

	return nil
}

func (s *SafetyManager) ShouldConfirm(change FileChange) bool {
	switch change.Action {
	case ActionDelete:
		return true
	case ActionModify:
		if _, err := os.Stat(change.Path); err == nil {
			return true
		}
	}
	return false
}

type BackupStore struct {
	backups map[string]string
}

func NewBackupStore() *BackupStore {
	return &BackupStore{
		backups: make(map[string]string),
	}
}

func (bs *BackupStore) Create(path, content string) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s.%s.bak", filepath.Base(path), timestamp)
	backupDir := filepath.Join(os.TempDir(), "siby-backups")
	
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	backupPath := filepath.Join(backupDir, backupName)
	if err := os.WriteFile(backupPath, []byte(content), 0644); err != nil {
		return "", err
	}

	bs.backups[path] = backupPath
	return backupPath, nil
}

func (bs *BackupStore) Restore(path string) error {
	backupPath, ok := bs.backups[path]
	if !ok {
		return fmt.Errorf("no backup found for %s", path)
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	delete(bs.backups, path)
	return nil
}

func (bs *BackupStore) Cleanup() {
	for _, backupPath := range bs.backups {
		os.Remove(backupPath)
	}
	bs.backups = make(map[string]string)
}

func (bs *BackupStore) List() map[string]string {
	result := make(map[string]string)
	for k, v := range bs.backups {
		result[k] = v
	}
	return result
}
