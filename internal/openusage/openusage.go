// Package openusage queries the OpenUsage daemon (if running) to enrich
// provider summaries with real server-side rate limit data.
package openusage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rdubar/llmstat/internal/provider"
)

// providerKeyMap maps llmstat provider names to OpenUsage snapshot keys.
var providerKeyMap = map[string]string{
	"claude": "claude-code",
	"codex":  "codex-cli",
	"gemini": "gemini-cli",
}

type metric struct {
	Limit     *float64 `json:"limit"`
	Remaining *float64 `json:"remaining"`
	Used      *float64 `json:"used"`
	Unit      string   `json:"unit"`
	Window    string   `json:"window"`
}

type snapshot struct {
	Metrics map[string]metric  `json:"metrics"`
	Resets  map[string]string  `json:"resets"`
}

type readModelResponse struct {
	Snapshots map[string]snapshot `json:"snapshots"`
}

func socketPath() string {
	if base := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); base != "" {
		return filepath.Join(base, "openusage", "telemetry.sock")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "openusage", "telemetry.sock")
}

func newClient(sock string) *http.Client {
	dialer := &net.Dialer{Timeout: 1 * time.Second}
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return dialer.DialContext(ctx, "unix", sock)
			},
			DisableKeepAlives: true,
		},
		Timeout: 3 * time.Second,
	}
}

// Enrich queries the OpenUsage daemon and fills in LimitPct/LimitSource/LimitLabel
// for any provider where server-side rate limit data is available.
// Silently no-ops if the daemon is not running.
func Enrich(summaries []provider.Summary) {
	sock := socketPath()
	if _, err := os.Stat(sock); err != nil {
		return // daemon not installed or not running
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := newClient(sock)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix/v1/read-model",
		bytes.NewReader([]byte("{}")))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var result readModelResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return
	}

	for i := range summaries {
		ouKey, ok := providerKeyMap[summaries[i].Name]
		if !ok {
			continue
		}
		snap, ok := result.Snapshots[ouKey]
		if !ok {
			continue
		}
		applyLimits(&summaries[i], snap)
	}
}

func applyLimits(s *provider.Summary, snap snapshot) {
	// Prefer 5h primary window; fall back to 7d secondary.
	for _, key := range []string{"rate_limit_primary", "rate_limit_secondary"} {
		m, ok := snap.Metrics[key]
		if !ok || m.Used == nil || m.Limit == nil || *m.Limit <= 0 {
			continue
		}
		pct := *m.Used / *m.Limit // 0.0–1.0 equivalent (values are 0–100)
		s.LimitPct = pct / 100.0
		s.LimitSource = "ou"
		label := fmt.Sprintf("%s window [ou]", m.Window)
		if resetStr, ok := snap.Resets[key]; ok {
			if t, err := time.Parse(time.RFC3339, resetStr); err == nil {
				label += " ↺ " + t.Local().Format("15:04")
			}
		}
		s.LimitLabel = label
		return
	}
}
