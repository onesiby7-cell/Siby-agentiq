package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	CurrentVersion = "2.0.0"
	GitHubAPI      = "https://api.github.com"
	GitHubRepo     = "siby-agentiq/siby-agentiq"
)

type UpdateChecker struct {
	client     *http.Client
	updateURL  string
	checkURL   string
	versionURL string
}

type ReleaseInfo struct {
	TagName    string    `json:"tag_name"`
	Name       string    `json:"name"`
	Body       string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Assets     []Asset   `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size              int64  `json:"size"`
}

type VersionInfo struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

type UpdateStatus struct {
	Available      bool
	CurrentVersion string
	LatestVersion  string
	ReleaseNotes   string
	DownloadURL    string
	DownloadSize   int64
}

func NewUpdateChecker() *UpdateChecker {
	return &UpdateChecker{
		client: &http.Client{Timeout: 10 * time.Second},
		updateURL:  fmt.Sprintf("https://github.com/%s/releases/latest", GitHubRepo),
		checkURL:   fmt.Sprintf("%s/repos/%s/releases/latest", GitHubAPI, GitHubRepo),
		versionURL: "https://raw.githubusercontent.com/siby-agentiq/siby-agentiq/main/VERSION",
	}
}

func (uc *UpdateChecker) CheckForUpdate(ctx context.Context) (*UpdateStatus, error) {
	status := &UpdateStatus{
		CurrentVersion: CurrentVersion,
		Available:     false,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", uc.checkURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Siby-Agentiq/"+CurrentVersion)

	resp, err := uc.client.Do(req)
	if err != nil {
		return status, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return status, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return status, nil
	}

	var release ReleaseInfo
	if err := json.Unmarshal(body, &release); err != nil {
		return status, nil
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	
	if uc.compareVersions(CurrentVersion, latestVersion) > 0 {
		status.Available = true
		status.LatestVersion = latestVersion
		status.ReleaseNotes = release.Body

		for _, asset := range release.Assets {
			expectedName := uc.getExpectedAssetName()
			if strings.Contains(asset.Name, expectedName) {
				status.DownloadURL = asset.BrowserDownloadURL
				status.DownloadSize = asset.Size
				break
			}
		}
	}

	return status, nil
}

func (uc *UpdateChecker) compareVersions(current, latest string) int {
	c := uc.parseVersion(current)
	l := uc.parseVersion(latest)

	if l.Major > c.Major {
		return 1
	}
	if l.Major < c.Major {
		return -1
	}

	if l.Minor > c.Minor {
		return 1
	}
	if l.Minor < c.Minor {
		return -1
	}

	if l.Patch > c.Patch {
		return 1
	}
	if l.Patch < c.Patch {
		return -1
	}

	return 0
}

func (uc *UpdateChecker) parseVersion(v string) VersionInfo {
	parts := strings.Split(v, "-")
	versionParts := strings.Split(parts[0], ".")

	info := VersionInfo{}
	
	if len(versionParts) >= 1 {
		fmt.Sscanf(versionParts[0], "%d", &info.Major)
	}
	if len(versionParts) >= 2 {
		fmt.Sscanf(versionParts[1], "%d", &info.Minor)
	}
	if len(versionParts) >= 3 {
		fmt.Sscanf(versionParts[2], "%d", &info.Patch)
	}
	
	if len(parts) > 1 {
		info.Prerelease = parts[1]
	}

	return info
}

func (uc *UpdateChecker) getExpectedAssetName() string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	ext := ""
	
	if os == "windows" {
		ext = ".exe"
	}
	
	return fmt.Sprintf("siby-agentiq-%s-%s%s", os, arch, ext)
}

func (uc *UpdateChecker) DownloadUpdate(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := uc.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "siby-agentiq-update")

	out, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(tmpFile)
		return "", err
	}

	os.Chmod(tmpFile, 0755)

	return tmpFile, nil
}

func (uc *UpdateChecker) ApplyUpdate(binaryPath string) error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	backupPath := execPath + ".bak"
	
	if _, err := os.Stat(backupPath); err == nil {
		os.Remove(backupPath)
	}

	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	if err := os.Rename(binaryPath, execPath); err != nil {
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to apply update: %w", err)
	}

	os.Chmod(execPath, 0755)

	return nil
}

func (uc *UpdateChecker) RenderUpdateStatus(status *UpdateStatus) string {
	if !status.Available {
		return fmt.Sprintf(`
╔══════════════════════════════════════════════════════════╗
║  🦂 SIBY-AGENTIQ UPDATE CHECKER                     ║
╠══════════════════════════════════════════════════════════╣
║                                                          ║
║  %s✓ Vous êtes à jour!%s                                ║
║  Version: %s%s%s                                          ║
║                                                          ║
║  %sProchaine vérification dans 24 heures.%s                ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝`,
			"\033[92m", "\033[0m",
			"\033[96m", status.CurrentVersion, "\033[0m",
			"\033[90m", "\033[0m",
		)
	}

	return fmt.Sprintf(`
╔══════════════════════════════════════════════════════════╗
║  🦂 SIBY-AGENTIQ UPDATE AVAILABLE                    ║
╠══════════════════════════════════════════════════════════╣
║                                                          ║
║  %s⚠ Nouvelle version disponible!%s                       ║
║                                                          ║
║  %sActuelle:%s %s%s                                     ║
║  %sDernière:%s %s%s                                      ║
║                                                          ║
║  %s📥 Taille:%s %s%s                                     ║
║                                                          ║
║  %sTapez /update pour installer la mise à jour.%s         ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝`,
		"\033[93m", "\033[0m",
		"\033[90m", "\033[96m", status.CurrentVersion, "\033[0m",
		"\033[90m", "\033[92m", status.LatestVersion, "\033[0m",
		"\033[90m", "\033[96m", formatSize(status.DownloadSize), "\033[0m",
		"\033[96m", "\033[0m",
	)
}

func (uc *UpdateChecker) RenderChangelog(releaseNotes string) string {
	lines := strings.Split(releaseNotes, "\n")
	var sb strings.Builder

	sb.WriteString("\n📋 Notes de version:\n\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "##") {
			sb.WriteString(fmt.Sprintf("\n%s\n", line))
			continue
		}
		sb.WriteString(fmt.Sprintf("  %s\n", line))
	}

	return sb.String()
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

type UpdateConfig struct {
	AutoCheck   bool
	CheckInterval time.Duration
	AutoDownload bool
	NotifyOnly   bool
}

func DefaultUpdateConfig() *UpdateConfig {
	return &UpdateConfig{
		AutoCheck:     true,
		CheckInterval: 24 * time.Hour,
		AutoDownload: false,
		NotifyOnly:   true,
	}
}

func (c *UpdateConfig) Load() error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".siby", "update.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, c)
}

func (c *UpdateConfig) Save() error {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".siby")
	os.MkdirAll(configDir, 0755)

	configPath := filepath.Join(configDir, "update.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
