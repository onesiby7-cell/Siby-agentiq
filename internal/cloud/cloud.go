package cloud

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	CloudReset   = "\033[0m"
	CloudCyan    = "\033[96m"
	CloudGreen   = "\033[92m"
	CloudYellow  = "\033[93m"
	CloudRed     = "\033[91m"
	CloudPurple  = "\033[95m"
)

type CloudSync struct {
	enabled       bool
	encryptionKey []byte
	provider      CloudProvider
	localPath     string
	remotePath    string
	status        SyncStatus
	mu            sync.RWMutex
	devices       map[string]*Device
	lastSync      time.Time
}

type CloudProvider interface {
	Upload(ctx context.Context, data []byte, path string) error
	Download(ctx context.Context, path string) ([]byte, error)
	List(ctx context.Context, path string) ([]string, error)
	Delete(ctx context.Context, path string) error
}

type SyncStatus struct {
	State         string
	Progress      float64
	PendingChanges int
	LastSync      time.Time
	Error         string
}

type Device struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	LastSeen time.Time `json:"last_seen"`
	Synced   bool      `json:"synced"`
}

type SyncEntry struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      []byte                 `json:"data"`
	Hash      string                 `json:"hash"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type LocalProvider struct {
	basePath string
}

type S3Provider struct {
	bucket    string
	region    string
	accessKey string
	secretKey string
}

type WebDAVProvider struct {
	endpoint  string
	username  string
	password  string
}

func NewLocalProvider(basePath string) *LocalProvider {
	return &LocalProvider{basePath: basePath}
}

func (p *LocalProvider) Upload(ctx context.Context, data []byte, path string) error {
	fullPath := filepath.Join(p.basePath, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, data, 0644)
}

func (p *LocalProvider) Download(ctx context.Context, path string) ([]byte, error) {
	fullPath := filepath.Join(p.basePath, path)
	return os.ReadFile(fullPath)
}

func (p *LocalProvider) List(ctx context.Context, path string) ([]string, error) {
	fullPath := filepath.Join(p.basePath, path)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

func (p *LocalProvider) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(p.basePath, path)
	return os.Remove(fullPath)
}

func NewCloudSync(workDir string) *CloudSync {
	cs := &CloudSync{
		enabled:   false,
		localPath: filepath.Join(workDir, ".siby", "cloud"),
		devices:   make(map[string]*Device),
		status: SyncStatus{
			State:    "idle",
			Progress: 0,
		},
	}

	os.MkdirAll(cs.localPath, 0755)

	cs.provider = NewLocalProvider(cs.localPath)

	return cs
}

func (cs *CloudSync) Enable() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.enabled {
		return nil
	}

	if len(cs.encryptionKey) == 0 {
		defaultKey := []byte("siby-agentIq-2026-secure-key-32b")
		cs.encryptionKey = deriveKey(defaultKey)
	}

	cs.enabled = true
	cs.status.State = "ready"

	return nil
}

func (cs *CloudSync) Disable() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.enabled = false
	cs.status.State = "disabled"
}

func (cs *CloudSync) IsEnabled() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.enabled
}

func (cs *CloudSync) SetEncryptionKey(key string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if len(key) < 32 {
		return fmt.Errorf("encryption key must be at least 32 characters")
	}

	cs.encryptionKey = deriveKey([]byte(key))
	return nil
}

func deriveKey(password []byte) []byte {
	hash := sha256.Sum256(password)
	return hash[:32]
}

func (cs *CloudSync) Encrypt(data []byte) ([]byte, error) {
	cs.mu.RLock()
	key := cs.encryptionKey
	cs.mu.RUnlock()

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func (cs *CloudSync) Decrypt(data []byte) ([]byte, error) {
	cs.mu.RLock()
	key := cs.encryptionKey
	cs.mu.RUnlock()

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (cs *CloudSync) SyncMemory(ctx context.Context, data interface{}) error {
	cs.mu.Lock()
	cs.status.State = "syncing"
	cs.mu.Unlock()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	encrypted, err := cs.Encrypt(jsonData)
	if err != nil {
		cs.mu.Lock()
		cs.status.State = "error"
		cs.status.Error = err.Error()
		cs.mu.Unlock()
		return err
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("memory_%s.enc", timestamp)

	if err := cs.provider.Upload(ctx, encrypted, filename); err != nil {
		cs.mu.Lock()
		cs.status.State = "error"
		cs.status.Error = err.Error()
		cs.mu.Unlock()
		return err
	}

	cs.mu.Lock()
	cs.lastSync = time.Now()
	cs.status.State = "synced"
	cs.status.Progress = 100
	cs.mu.Unlock()

	return nil
}

func (cs *CloudSync) RestoreMemory(ctx context.Context) (interface{}, error) {
	cs.mu.Lock()
	cs.status.State = "restoring"
	cs.mu.Unlock()

	files, err := cs.provider.List(ctx, "")
	if err != nil {
		cs.mu.Lock()
		cs.status.State = "error"
		cs.mu.Unlock()
		return nil, err
	}

	var latestFile string
	var latestTime time.Time

	for _, f := range files {
		if strings.HasPrefix(f, "memory_") && strings.HasSuffix(f, ".enc") {
			t, _ := time.Parse("2006-01-02_15-04-05", strings.TrimPrefix(strings.TrimSuffix(f, ".enc"), "memory_"))
			if t.After(latestTime) {
				latestTime = t
				latestFile = f
			}
		}
	}

	if latestFile == "" {
		cs.mu.Lock()
		cs.status.State = "idle"
		cs.mu.Unlock()
		return nil, fmt.Errorf("no backup found")
	}

	encrypted, err := cs.provider.Download(ctx, latestFile)
	if err != nil {
		cs.mu.Lock()
		cs.status.State = "error"
		cs.mu.Unlock()
		return nil, err
	}

	decrypted, err := cs.Decrypt(encrypted)
	if err != nil {
		cs.mu.Lock()
		cs.status.State = "error"
		cs.mu.Unlock()
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(decrypted, &result); err != nil {
		cs.mu.Lock()
		cs.status.State = "error"
		cs.mu.Unlock()
		return nil, err
	}

	cs.mu.Lock()
	cs.status.State = "synced"
	cs.mu.Unlock()

	return result, nil
}

func (cs *CloudSync) AddDevice(device *Device) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.devices[device.ID] = device
}

func (cs *CloudSync) GetDevices() map[string]*Device {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	devices := make(map[string]*Device)
	for k, v := range cs.devices {
		devices[k] = v
	}
	return devices
}

func (cs *CloudSync) GetStatus() *SyncStatus {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return &SyncStatus{
		State:         cs.status.State,
		Progress:      cs.status.Progress,
		PendingChanges: cs.status.PendingChanges,
		LastSync:      cs.lastSync,
		Error:         cs.status.Error,
	}
}

func (cs *CloudSync) RenderSyncStatus() string {
	var sb strings.Builder

	status := cs.GetStatus()

	statusColor := CloudRed
	if status.State == "synced" {
		statusColor = CloudGreen
	} else if status.State == "syncing" {
		statusColor = CloudYellow
	}

	sb.WriteString(fmt.Sprintf("\n%s☁️  CLOUD SYNC STATUS%s\n", CloudCyan, CloudReset))
	sb.WriteString(fmt.Sprintf("  Status:       %s%s%s\n", statusColor, status.State, CloudReset))
	sb.WriteString(fmt.Sprintf("  Progress:     %.0f%%\n", status.Progress))
	sb.WriteString(fmt.Sprintf("  Last Sync:    %s\n", status.LastSync.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("  Pending:      %d changes\n", status.PendingChanges))
	sb.WriteString(fmt.Sprintf("  Encryption:   %sAES-256-GCM%s\n", CloudGreen, CloudReset))

	if len(cs.devices) > 0 {
		sb.WriteString(fmt.Sprintf("\n  %sRegistered Devices:%s\n", CloudCyan, CloudReset))
		for _, device := range cs.devices {
			synced := CloudGreen + "✓" + CloudReset
			if !device.Synced {
				synced = CloudRed + "✗" + CloudReset
			}
			sb.WriteString(fmt.Sprintf("    • %s %s [%s]\n", device.Name, synced, device.Type))
		}
	}

	if status.Error != "" {
		sb.WriteString(fmt.Sprintf("\n  %s⚠ Error: %s%s\n", CloudRed, status.Error, CloudReset))
	}

	sb.WriteString(fmt.Sprintf("\n  %s🦂 Powered by Ibrahim Siby • E2E Encrypted 🦂%s\n", CloudYellow, CloudReset))

	return sb.String()
}

func (cs *CloudSync) ExportEncrypted(exportPath string) error {
	cs.mu.RLock()
	key := make([]byte, len(cs.encryptionKey))
	copy(key, cs.encryptionKey)
	cs.mu.RUnlock()

	data := struct {
		Key      []byte              `json:"key"`
		Status   *SyncStatus         `json:"status"`
		Devices  map[string]*Device `json:"devices"`
	}{
		Key:      key,
		Status:   cs.GetStatus(),
		Devices:  cs.GetDevices(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	encrypted, err := cs.Encrypt(jsonData)
	if err != nil {
		return err
	}

	return os.WriteFile(exportPath, encrypted, 0600)
}

func (cs *CloudSync) ImportEncrypted(importPath string) error {
	encrypted, err := os.ReadFile(importPath)
	if err != nil {
		return err
	}

	decrypted, err := cs.Decrypt(encrypted)
	if err != nil {
		return err
	}

	var data struct {
		Key     []byte              `json:"key"`
		Status  *SyncStatus         `json:"status"`
		Devices map[string]*Device  `json:"devices"`
	}

	if err := json.Unmarshal(decrypted, &data); err != nil {
		return err
	}

	cs.mu.Lock()
	cs.encryptionKey = data.Key
	for id, device := range data.Devices {
		cs.devices[id] = device
	}
	cs.mu.Unlock()

	return nil
}

func GenerateSyncEntry(entryType string, data interface{}) (*SyncEntry, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(jsonData)

	return &SyncEntry{
		ID:        fmt.Sprintf("sync_%d", time.Now().UnixNano()),
		Type:      entryType,
		Data:      jsonData,
		Hash:      base64.StdEncoding.EncodeToString(hash[:]),
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"version": "2.0.0",
			"creator": "Ibrahim Siby",
		},
	}, nil
}
