// Package gemini reads session JSON files from ~/.gemini/tmp/*/chats/.
// Each model response includes a tokens object with input/output/total counts.
package gemini

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/rdubar/llmstat/internal/provider"
)

type Provider struct{}

func (Provider) Name() string { return "gemini" }

func (Provider) Detect() bool {
	home, _ := os.UserHomeDir()
	_, err := os.Stat(filepath.Join(home, ".gemini", "installation_id"))
	return err == nil
}

func (Provider) Collect(cfg provider.ProviderConfig, since time.Time) (provider.Summary, error) {
	home, _ := os.UserHomeDir()
	chatsGlob := filepath.Join(home, ".gemini", "tmp", "*", "chats", "session-*.json")

	files, err := filepath.Glob(chatsGlob)
	if err != nil || len(files) == 0 {
		return provider.Summary{Name: "gemini", LimitPct: -1}, nil
	}

	var tokensTotal, rate5 int64
	var sessions int

	now := time.Now()
	win5 := now.Add(-5 * time.Minute)

	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil || info.ModTime().Before(since) {
			continue
		}

		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}

		var session struct {
			Messages []struct {
				Type      string    `json:"type"`
				Timestamp time.Time `json:"timestamp"`
				Tokens    *struct {
					Total int64 `json:"total"`
				} `json:"tokens"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		sessionHasTokens := false
		for _, msg := range session.Messages {
			if msg.Tokens == nil || msg.Timestamp.IsZero() || msg.Timestamp.Before(since) {
				continue
			}
			tokensTotal += msg.Tokens.Total
			sessionHasTokens = true
			if msg.Timestamp.After(win5) {
				rate5 += msg.Tokens.Total
			}
		}
		if sessionHasTokens {
			sessions++
		}
	}

	return provider.Summary{
		Name:        "gemini",
		TokensToday: tokensTotal,
		RatePer5Min: rate5,
		Sessions:    sessions,
		LimitPct:    -1,
	}, nil
}
