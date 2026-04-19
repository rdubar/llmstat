package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rdubar/llmstat/internal/config"
	"github.com/rdubar/llmstat/internal/display"
	"github.com/rdubar/llmstat/internal/provider"
	"github.com/rdubar/llmstat/internal/provider/claude"
	"github.com/rdubar/llmstat/internal/provider/codex"
	"github.com/rdubar/llmstat/internal/provider/cursor"
	"github.com/rdubar/llmstat/internal/provider/gemini"
)

var allProviders = []provider.Provider{
	claude.Provider{},
	codex.Provider{},
	gemini.Provider{},
	cursor.Provider{},
}

func main() {
	var (
		doSetup  = flag.Bool("setup", false, "Run interactive setup wizard")
		weekly   = flag.Bool("w", false, "Show this week's usage instead of today")
		jsonOut  = flag.Bool("json", false, "Output JSON")
		cfgPath  = flag.String("config", config.DefaultPath(), "Config file path")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: llmstat [flags] [provider]\n\n")
		fmt.Fprintf(os.Stderr, "  provider    Optional: show detail for one provider (e.g. llmstat claude)\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *doSetup {
		if err := config.RunSetup(*cfgPath); err != nil {
			fmt.Fprintln(os.Stderr, "setup error:", err)
			os.Exit(1)
		}
		return
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	// Determine time window
	now := time.Now().UTC()
	var since time.Time
	if *weekly {
		since = now.AddDate(0, 0, -7)
	} else {
		since = now.Truncate(24 * time.Hour) // start of today UTC
	}

	// Filter to a single provider if positional arg given
	target := flag.Arg(0)

	// Collect stale warnings
	var warnings []string
	for _, name := range []string{"claude", "codex", "gemini"} {
		tier := tierFor(cfg, name)
		if w := config.StaleWarning(name, tier); w != "" {
			warnings = append(warnings, w)
		}
	}

	// Run providers
	var summaries []provider.Summary
	for _, p := range allProviders {
		if target != "" && p.Name() != target {
			continue
		}
		if !p.Detect() {
			continue
		}
		pcfg := providerCfg(cfg, p.Name())
		s, _ := p.Collect(pcfg, since)
		summaries = append(summaries, s)
	}

	if len(summaries) == 0 {
		if target != "" {
			fmt.Fprintf(os.Stderr, "provider %q not detected on this machine\n", target)
		} else {
			fmt.Fprintln(os.Stderr, "no AI tools detected — run `llmstat --setup` to configure")
		}
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(summaries)
		return
	}

	display.PrintWarnings(warnings)
	display.Render(summaries)
}

func tierFor(cfg config.Config, name string) string {
	switch name {
	case "claude":
		return cfg.Claude.Tier
	case "codex":
		return cfg.Codex.Tier
	case "gemini":
		return cfg.Gemini.Tier
	case "cursor":
		return cfg.Cursor.Tier
	}
	return ""
}

func providerCfg(cfg config.Config, name string) provider.ProviderConfig {
	switch name {
	case "claude":
		return provider.ProviderConfig{
			Tier:           cfg.Claude.Tier,
			DailyBudgetUSD: cfg.Claude.DailyBudgetUSD,
			LogPath:        cfg.Claude.LogPath,
		}
	case "codex":
		return provider.ProviderConfig{Tier: cfg.Codex.Tier}
	case "gemini":
		return provider.ProviderConfig{Tier: cfg.Gemini.Tier}
	case "cursor":
		return provider.ProviderConfig{Tier: cfg.Cursor.Tier}
	}
	return provider.ProviderConfig{}
}
