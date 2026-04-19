package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
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
		doSetup   = flag.Bool("setup", false, "Run interactive setup wizard")
		doUpgrade bool
		doVersion bool
		doCredits bool
		weekly    = flag.Bool("w", false, "Show this week's usage instead of today")
		monthly   = flag.Bool("m", false, "Show this month's usage instead of today")
		jsonOut   = flag.Bool("json", false, "Output JSON")
		cfgPath   = flag.String("config", config.DefaultPath(), "Config file path")
	)
	flag.BoolVar(&doUpgrade, "upgrade", false, "Upgrade llmstat to the latest version")
	flag.BoolVar(&doUpgrade, "u", false, "Alias for --upgrade")
	flag.BoolVar(&doVersion, "version", false, "Show version and build info")
	flag.BoolVar(&doVersion, "v", false, "Alias for --version")
	flag.BoolVar(&doCredits, "credits", false, "Show credits")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: llmstat [flags] [provider]\n\n")
		fmt.Fprintf(os.Stderr, "  provider    Optional: show detail for one provider (e.g. llmstat claude)\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if doVersion {
		fmt.Println(versionInfo())
		return
	}

	if doCredits {
		fmt.Println("llmstat — https://github.com/rdubar/llmstat")
		fmt.Println("Built and maintained by Roger Dubar (https://github.com/rdubar)")
		fmt.Println("With thanks to Alphapet Ventures (https://alpha.pet)")
		fmt.Println("Development assistance: Claude (Anthropic), Codex (OpenAI)")
		fmt.Println("MIT License")
		return
	}

	if doUpgrade {
		cmd := exec.Command("go", "install", "github.com/rdubar/llmstat/cmd/llmstat@latest")
		cmd.Env = append(os.Environ(), "GOPROXY=direct", "GONOSUMDB=*")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "upgrade failed:", err)
			os.Exit(1)
		}
		fmt.Println("llmstat upgraded to latest.")
		return
	}

	if *doSetup {
		if err := config.RunSetup(*cfgPath); err != nil {
			fmt.Fprintln(os.Stderr, "setup error:", err)
			os.Exit(1)
		}
		return
	}

	if !config.Exists(*cfgPath) {
		fmt.Fprintln(os.Stderr, "No config found. Run `llmstat --setup` to configure your AI tools.")
		fmt.Fprintln(os.Stderr, "llmstat will still show any tools it can detect automatically.")
		fmt.Fprintln(os.Stderr)
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	// Determine time window using local time so "today" matches the user's clock.
	now := time.Now()
	var since time.Time
	var periodLabel string
	switch {
	case *monthly:
		since = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		periodLabel = since.Format("January 2006")
	case *weekly:
		since = now.AddDate(0, 0, -7)
		periodLabel = "since " + since.Format("2 Jan")
	}
	if periodLabel == "" {
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
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
		label := periodLabel
		if label == "" {
			label = "today"
		}
		for i := range summaries {
			summaries[i].Period = label
			summaries[i].Since = since
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(summaries)
		return
	}

	display.PrintWarnings(warnings)
	if periodLabel != "" {
		display.PrintPeriod(periodLabel)
	}
	display.Render(summaries)
}

func versionInfo() string {
	version := "(development)"
	built := ""

	info, ok := debug.ReadBuildInfo()
	if ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
		for _, s := range info.Settings {
			if s.Key == "vcs.time" {
				built = "  built " + s.Value
			}
		}
	}
	return "llmstat " + version + built
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
