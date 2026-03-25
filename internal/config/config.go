package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	Subreddits []string `yaml:"subreddits"`
	RedditSort string   `yaml:"reddit_sort"` // hot, new, top, rising
	HNSort     string   `yaml:"hn_sort"`     // top, new, best
}

// Valid sort options
var (
	RedditSorts = []string{"hot", "new", "top", "rising"}
	HNSorts     = []string{"top", "new", "best"}
)

// DefaultSubreddits is the default list of subreddits to fetch.
var DefaultSubreddits = []string{"programming", "linux", "opencodecli", "claudecode"}

// Load reads the config from ~/.config/technews-tui/config.yaml.
// If the file doesn't exist, it creates one with defaults.
func Load() (*Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return defaultConfig(), nil
	}

	dir := filepath.Join(configDir, "technews-tui")
	path := filepath.Join(dir, "config.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist — create default
		cfg := defaultConfig()
		_ = writeDefault(dir, path, cfg)
		return cfg, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultConfig(), nil
	}

	if len(cfg.Subreddits) == 0 {
		cfg.Subreddits = DefaultSubreddits
	}
	cfg.ValidateSort()
	return &cfg, nil
}

// Save writes the config back to ~/.config/technews-tui/config.yaml.
func Save(cfg *Config) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(configDir, "technews-tui")
	path := filepath.Join(dir, "config.yaml")
	return writeDefault(dir, path, cfg)
}

func defaultConfig() *Config {
	return &Config{
		Subreddits: DefaultSubreddits,
		RedditSort: "hot",
		HNSort:     "top",
	}
}

// ValidateSort ensures sort values are valid, resetting to defaults if not.
func (c *Config) ValidateSort() {
	if !contains(RedditSorts, c.RedditSort) {
		c.RedditSort = "hot"
	}
	if !contains(HNSorts, c.HNSort) {
		c.HNSort = "top"
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func writeDefault(dir, path string, cfg *Config) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	header := []byte("# technews-tui configuration\n# Add or remove subreddits below.\n\n")
	return os.WriteFile(path, append(header, data...), 0o644)
}
