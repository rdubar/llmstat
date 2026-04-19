package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type ProviderConfig struct {
	Tier           string  `toml:"tier"`
	DailyBudgetUSD float64 `toml:"daily_budget_usd"`
	LogPath        string  `toml:"log_path"`
	Enabled        *bool   `toml:"enabled"`
}

type Config struct {
	Claude ProviderConfig `toml:"claude"`
	Codex  ProviderConfig `toml:"codex"`
	Gemini ProviderConfig `toml:"gemini"`
	Cursor ProviderConfig `toml:"cursor"`
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "llmstat", "config.toml")
}

// Load reads config from path. Missing file returns a zero Config (valid).
func Load(path string) (Config, error) {
	var cfg Config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	_, err := toml.DecodeFile(path, &cfg)
	return cfg, err
}

// Save writes cfg to path, creating parent directories as needed.
func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
