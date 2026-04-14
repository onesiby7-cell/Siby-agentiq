package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	mu       sync.RWMutex
	Provider string   `json:"provider"`
	Config   string   `json:"config"`
	APIKeys  []string `json:"api_keys"`
}

var globalConfig *Config
var configOnce sync.Once

func GetConfig() *Config {
	configOnce.Do(func() {
		globalConfig = &Config{
			APIKeys: make([]string, 0),
		}
		globalConfig.LoadCache()
	})
	return globalConfig
}

func (c *Config) GetCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".siby", "discovery.json")
}

func (c *Config) LoadCache() bool {
	path := c.GetCachePath()
	
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, c)
		return c.Provider != "" && c.Provider != "none"
	}
	return false
}

func (c *Config) SaveCache() {
	path := c.GetCachePath()
	dir := filepath.Dir(path)
	
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return
	}
	
	os.WriteFile(path, data, 0644)
}

func (c *Config) SetProvider(provider, config string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Provider = provider
	c.Config = config
	c.SaveCache()
}

func (c *Config) GetProvider() (string, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Provider, c.Config
}
