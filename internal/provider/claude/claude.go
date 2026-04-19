package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rdubar/llmstat/internal/config"
	"github.com/rdubar/llmstat/internal/provider"
)

const rateWindowMinutes = 5

type Provider struct{}

func (Provider) Name() string { return "claude" }

func (Provider) Detect() bool {
	home, _ := os.UserHomeDir()
	_, err := os.Stat(filepath.Join(home, ".claude", "projects"))
	return err == nil
}

func (Provider) Collect(cfg provider.ProviderConfig, since time.Time) (provider.Summary, error) {
	logDir := cfg.LogPath
	if logDir == "" {
		home, _ := os.UserHomeDir()
		logDir = filepath.Join(home, ".claude", "projects")
	}

	now := time.Now().UTC()
	rateStart := now.Add(-rateWindowMinutes * time.Minute)

	var tokensToday, rateTokens int64
	var costUSD float64

	pattern := filepath.Join(logDir, "**", "*.jsonl")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return provider.Summary{Name: "claude", Err: err}, err
	}
	// filepath.Glob doesn't support **, so walk manually
	paths = nil
	_ = filepath.WalkDir(logDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".jsonl" {
			paths = append(paths, path)
		}
		return nil
	})

	for _, path := range paths {
		tokens, rate, cost, walkErr := parseJSONL(path, since, rateStart)
		if walkErr != nil {
			continue // skip unreadable files
		}
		tokensToday += tokens
		rateTokens += rate
		costUSD += cost
	}

	sum := provider.Summary{
		Name:        "claude",
		TokensToday: tokensToday,
		CostUSD:     costUSD,
		RatePer5Min: rateTokens,
	}

	// Apply tier limit if configured
	if cfg.Tier != "" {
		entry, ok := config.LookupTier("claude", cfg.Tier)
		if ok && entry.TokensPerWindow > 0 {
			// Use the 5-hour rolling window as the limit metric
			windowStart := now.Add(-time.Duration(entry.WindowHours) * time.Hour)
			var windowTokens int64
			for _, path := range paths {
				tokens, _, _, _ := parseJSONL(path, windowStart, windowStart)
				windowTokens += tokens
			}
			sum.LimitPct = float64(windowTokens) / float64(entry.TokensPerWindow)
			sum.LimitSource = "tier"
			sum.LimitLabel = fmt.Sprintf("%dhr window (%s est.)", entry.WindowHours, cfg.Tier)
		}
	}

	if cfg.DailyBudgetUSD > 0 {
		sum.LimitPct = costUSD / cfg.DailyBudgetUSD
		sum.LimitSource = "budget"
		sum.LimitLabel = fmt.Sprintf("$%.2f/day budget", cfg.DailyBudgetUSD)
	}

	if sum.LimitSource == "" {
		sum.LimitPct = -1
	}

	return sum, nil
}

type jsonlRecord struct {
	Timestamp string  `json:"timestamp"`
	CostUSD   float64 `json:"costUSD"`
	Message   struct {
		Usage *struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

type usageKey struct {
	tsSec int64
	in    int64
	out   int64
	read  int64
	write int64
}

func parseJSONL(path string, since, rateStart time.Time) (tokens, rateTokens int64, cost float64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, 0, err
	}
	defer f.Close()

	// Deduplicate repeated assistant events with identical usage payloads
	// (Claude logs emit the same usage object multiple times within milliseconds).
	seen := make(map[usageKey]bool)

	dec := json.NewDecoder(f)
	for dec.More() {
		var rec jsonlRecord
		if err := dec.Decode(&rec); err != nil {
			continue
		}
		if rec.Message.Usage == nil {
			continue
		}
		ts, err := time.Parse(time.RFC3339, rec.Timestamp)
		if err != nil {
			ts, err = time.Parse("2006-01-02T15:04:05.999Z", rec.Timestamp)
			if err != nil {
				continue
			}
		}
		if ts.Before(since) {
			continue
		}
		u := rec.Message.Usage
		key := usageKey{ts.Unix(), u.InputTokens, u.OutputTokens, u.CacheReadInputTokens, u.CacheCreationInputTokens}
		if seen[key] {
			continue
		}
		seen[key] = true
		t := u.InputTokens + u.OutputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
		tokens += t
		cost += rec.CostUSD
		if !ts.Before(rateStart) {
			rateTokens += t
		}
	}
	return tokens, rateTokens, cost, nil
}
