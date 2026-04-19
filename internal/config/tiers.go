package config

import (
	_ "embed"
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
)

//go:embed tiers.toml
var tiersBytes []byte

const TierStaleDays = 90

type TierEntry struct {
	LastVerified    string  `toml:"last_verified"`
	WindowHours     int     `toml:"window_hours"`
	TokensPerWindow int64   `toml:"tokens_per_window"`
	CostUSDPerDay   float64 `toml:"cost_usd_per_day"`
	PromptsPerDay   int64   `toml:"prompts_per_day"`
	Notes           string  `toml:"notes"`
}

type tierFile struct {
	Version string                 `toml:"version"`
	Claude  map[string]TierEntry   `toml:"claude"`
	Codex   map[string]TierEntry   `toml:"codex"`
	Gemini  map[string]TierEntry   `toml:"gemini"`
}

var loadedTiers tierFile

func init() {
	if err := toml.Unmarshal(tiersBytes, &loadedTiers); err != nil {
		panic("llmstat: failed to parse embedded tiers.toml: " + err.Error())
	}
}

// LookupTier returns the TierEntry for the given provider and tier name.
func LookupTier(provider, tier string) (TierEntry, bool) {
	var m map[string]TierEntry
	switch provider {
	case "claude":
		m = loadedTiers.Claude
	case "codex":
		m = loadedTiers.Codex
	case "gemini":
		m = loadedTiers.Gemini
	default:
		return TierEntry{}, false
	}
	entry, ok := m[tier]
	return entry, ok
}

// StaleWarning returns a non-empty string if the tier data is older than TierStaleDays.
func StaleWarning(provider, tier string) string {
	entry, ok := LookupTier(provider, tier)
	if !ok || entry.LastVerified == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02", entry.LastVerified)
	if err != nil {
		return ""
	}
	if time.Since(t) > time.Duration(TierStaleDays)*24*time.Hour {
		return fmt.Sprintf(
			"warning: %s %q tier data was last verified %s (>%d days ago — consider updating llmstat)",
			provider, tier, entry.LastVerified, TierStaleDays,
		)
	}
	return ""
}
