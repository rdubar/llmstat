// Package gemini reads ~/.gemini/telemetry.log and optionally calls
// `gemini /stats` for live quota. TODO milestone 7: implement full collection.
package gemini

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rdubar/llmstat/internal/provider"
)

type Provider struct{}

func (Provider) Name() string { return "gemini" }

func (Provider) Detect() bool {
	home, _ := os.UserHomeDir()
	_, err := os.Stat(filepath.Join(home, ".gemini", "telemetry.log"))
	return err == nil
}

func (Provider) Collect(cfg provider.ProviderConfig, since time.Time) (provider.Summary, error) {
	return provider.Summary{
		Name:     "gemini",
		LimitPct: -1,
		Extra:    "coming soon",
	}, nil
}
