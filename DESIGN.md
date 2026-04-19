# llmstat — Design Document

> Binary: `llmstat` (alias: `aiu`) · Language: Go · Status: pre-build design

---

## Problem

Every AI coding tool tracks its own usage in isolation. There is no single command
that tells you how much you have used across all your AI tools today, and how close
you are to your limits. You only find out you have hit a limit when something stops
working.

---

## What llmstat does

Reads local files left behind by AI tools — logs, SQLite databases, telemetry — and
prints a one-line summary per active provider showing usage and how close you are to
your limit or budget.

```
$ llmstat
claude  ████████░░  78% of Max plan est.  │ 42.3k tok  $0.18  1.2k/5min ↑
codex   ███░░░░░░░  31% of Plus plan est. │ 18.1k tok         7 threads
gemini  █░░░░░░░░░  12% of quota (live)   │  8.4k tok
cursor  ████░░░░░░  41% of Pro plan est.  │ (session data only — experimental)
```

Flags:
- `llmstat` — today's summary (default)
- `llmstat -w` — this week
- `llmstat -v` — verbose: one line per session/thread under each provider
- `llmstat claude` — positional arg for single provider drill-down
- `llmstat --json` — structured output for scripting

Shell alias for those who want short: `alias aiu=llmstat`

---

## What llmstat does NOT do

- No proxy, no API interception, no code instrumentation
- No network requests (Gemini `/stats` is a local subprocess call, not an API)
- No web UI, no server process, no daemon
- No writing to any AI tool's data files
- No account credentials stored or used

Everything is read-only from local files that already exist.

---

## Differentiators

Compared to existing tools (Tokscale, LiteLLM, Helicone, CodeBurn):

| | llmstat | Proxy tools | Tokscale |
|--|--|--|--|
| Setup | `llmstat setup` — done | Proxy config required | Manual |
| Network required | Never | Always | Sometimes |
| Limit awareness | Yes (tier + budget) | Cost only | No |
| Distribution | Single binary | Docker / pip | Cargo |
| Cross-platform | Mac + Linux + Pi | Varies | Varies |

---

## Language: Go

- Single binary — `scp` to Pi, done; no Python, no venv, no deps on target machine
- Cross-compile in one command: `GOOS=linux GOARCH=arm64 go build`
- Fast startup — matters for a quick status check at the shell prompt
- SQLite via `modernc.org/sqlite` — pure Go, no cgo, cross-compiles cleanly
- Terminal output via `github.com/charmbracelet/lipgloss` for bars and colour

---

## Providers — v1 scope

Four providers in v1. Others parked; the interface makes adding them a single new file.

### Claude Code
- **Source**: `~/.claude/projects/**/*.jsonl`
- **Data**: tokens (input/output/cache), cost, timestamps
- **Limit metric**: 5-hour rolling window token count (matches Claude's actual rate limit)
- **Notes**: richest local data of any provider; logic ported from rogkit `clu`

### OpenAI Codex
- **Source**: `~/.codex/state_5.sqlite`
- **Data**: per-thread token counts, timestamps, working directory
- **Limit metric**: tokens per 5-hour window (OpenAI moved to token-based pricing
  April 2026; older accounts may still see message-based estimates in the app)
- **Notes**: schema is not officially documented; treat as best-effort; log a clear
  warning if schema doesn't match rather than crashing

### Gemini CLI
- **Source**: `~/.gemini/telemetry.log` for token data
- **Quota**: invoke `gemini /stats` as a subprocess and parse stdout for live quota
- **Fallback**: if `gemini` is not in PATH or subprocess fails, use telemetry.log
  only and show "quota unknown" instead of a limit bar
- **Notes**: only v1 provider with live quota data; bar labelled "(live)"

### Cursor *(experimental)*
- **Source macOS**: `~/Library/Application Support/Cursor/User/globalStorage/state.vscdb`
- **Source Linux**: `~/.config/Cursor/User/globalStorage/state.vscdb`
- **Data**: conversation history and usage tracking in SQLite `ItemTable`
- **Limit metric**: credit pool is $-based so bar shows session count vs 7-day avg
  if token counts are not present in the schema
- **Notes**: schema is reverse-engineered and may break across Cursor versions;
  any error silently shows "unavailable" rather than crashing; labelled experimental

### Parked (post-v1)
- **Aider** — `.aider.chat.history.md`; no token counts locally, session data only
- **Copilot, Windsurf, Amazon Q** — no meaningful local data found

---

## Setup

First run experience. Providers are auto-detected from their data paths; setup only
asks about providers whose data is present on the current machine.

```
$ llmstat setup

llmstat setup
─────────────────────────────────────────
Detected: Claude Code, Codex, Gemini CLI

Claude subscription  [free / pro / max / enter to skip]: max
Codex subscription   [free / plus / pro / enter to skip]: plus
Gemini subscription  [free / advanced / enter to skip]:

Wrote ~/.config/llmstat/config.toml
Run `llmstat` to see your usage.
```

Rules:
- Only detected providers are shown
- Blank / enter = skip; provider still appears in output without a limit bar
- Config dir is created if it doesn't exist
- Re-running `llmstat setup` re-prompts all detected providers; existing answers
  shown as defaults so the user can just press enter to keep them
- Any field can be edited manually in the config file afterwards

---

## Limit / budget awareness

No provider reliably exposes remaining quota in local files. Four layers, in order
of preference:

1. **Live** — Gemini only; parsed from `gemini /stats` subprocess output
2. **Tier lookup** — user declares plan in setup; limits from bundled `tiers.toml`
3. **User override** — explicit daily budget in config; overrides tier
4. **7-day average** — no tier configured; bar shows today vs rolling average,
   no percentage, arrow indicates direction

Each limit type uses its natural metric — token window for Claude, request window
for Codex, prompts/day for Gemini — and the display label is explicit:

```
claude  ████████░░  78% of 5hr window (Max est.)
claude  ████████░░  78% of $5.00/day budget
gemini  █░░░░░░░░░  12% of daily quota (live)
codex   ███░░░░░░░  ↑ above 7-day avg
```

### tiers.toml

Bundled into the binary via `go:embed`. Each entry carries `last_verified`.
`llmstat` warns at startup if any configured tier entry is older than
`TIER_STALE_DAYS = 90`. Tiers are updated by cutting a new release — no network
fetch, no auto-update.

```toml
version = "1"

[claude.pro]
last_verified = "2026-04-19"
window_hours = 5
tokens_per_window = 88000        # approximate; soft limit
cost_usd_per_day = 3.33          # $100/month ÷ 30

[claude.max]
last_verified = "2026-04-19"
window_hours = 5
tokens_per_window = 220000       # approximate; soft limit
cost_usd_per_day = 16.67         # $500/month ÷ 30

# Codex moved to token-based pricing April 2026; limits are per 5-hour window.
# Free tier limits are not publicly documented by OpenAI — no entry until confirmed.
# Message estimates (shown in app for Plus/Pro) convert roughly to token ranges:
#   Plus:    20–100 msgs/5h on GPT-5.4  ≈ low-mid token range
#   Pro 5x:  100–500 msgs/5h
#   Pro 20x: 400–2000 msgs/5h
# Until OpenAI publishes token limits, Codex shows usage only — no limit bar.

[codex.plus]
last_verified = "2026-04-19"
window_hours = 5
notes = "Token limits not yet published; usage displayed without limit bar"

[codex.pro]
last_verified = "2026-04-19"
window_hours = 5
notes = "Token limits not yet published; usage displayed without limit bar"

[gemini.free]
last_verified = "2026-04-19"
prompts_per_day = 100

[gemini.advanced]
last_verified = "2026-04-19"
prompts_per_day = 300
```

**Note**: Codex token limits are not yet published by OpenAI (April 2026). Codex
will show usage data without a limit bar until OpenAI documents token thresholds.
Monitor: https://developers.openai.com/codex/pricing

---

## Configuration

`~/.config/llmstat/config.toml` — written by `llmstat setup`, editable by hand.

```toml
[claude]
tier = "max"
# daily_budget_usd = 5.00         # overrides tier
# log_path = "~/.claude/projects" # override if non-standard

[codex]
tier = "plus"

[gemini]
tier = "advanced"
# enabled = false                  # suppress a detected provider

[cursor]
# enabled = false
```

Providers are auto-detected from their data paths. `enabled = false` suppresses
a provider even if its files are present.

---

## Error handling

Errors in one provider must never affect others. Each provider runs independently;
on any error the line shows `[unavailable: <reason>]` and llmstat exits 0.

| Scenario | Behaviour |
|----------|-----------|
| Data path missing | Provider silently skipped (not installed) |
| SQLite locked or unreadable | `[unavailable: db locked]` |
| Malformed JSONL entry | Skip that entry, continue |
| Gemini subprocess not found | Fall back to telemetry.log, no quota bar |
| Gemini subprocess output unparseable | Show telemetry data, no quota bar |
| Unknown Cursor schema | `[unavailable: schema changed]` |
| Config file missing | Run as if no tiers configured; suggest `llmstat setup` |

---

## Render model

All providers are collected first, then rendered together. This is required for
column alignment — provider name, bar, percentage, and data columns must all be
right-padded to the widest value across all active providers before any output
is written.

```
collect all → compute column widths → render all
```

---

## Architecture

```
cmd/llmstat/
  main.go                # flags, detect, collect, render

internal/
  config/
    config.go            # load ~/.config/llmstat/config.toml
    tiers.go             # load + query embedded tiers.toml; TIER_STALE_DAYS = 90
    setup.go             # interactive setup wizard
  provider/
    provider.go          # Provider interface + Summary struct
    claude/claude.go
    codex/codex.go
    gemini/gemini.go
    cursor/cursor.go     # experimental
  display/
    bar.go               # bar renderer, colour, column alignment
    table.go             # verbose / drill-down view

tiers/
  tiers.toml             # embedded via go:embed
```

### Provider interface

```go
type Summary struct {
    Name        string
    TokensToday int64
    CostUSD     float64
    RatePer5Min float64
    Sessions    int
    LimitPct    float64    // 0.0–1.0; -1 = unknown
    LimitSource string     // "live", "tier", "budget", "avg", ""
    LimitLabel  string     // human label e.g. "5hr window (Max est.)"
    Extra       string     // provider-specific note shown after │
    Err         error      // non-nil shows [unavailable: ...]
}

type Provider interface {
    Name()   string
    Detect() bool          // false if data path absent — skip silently
    Collect(cfg ProviderConfig, since time.Time) (Summary, error)
}
```

---

## Display

Bar: 10 cells, `█` filled, `░` empty.
Colour: green < 60%, yellow 60–85%, red > 85%.
Percentage and label only when `LimitSource` is non-empty.
Plain-text fallback (no ANSI) when stdout is not a TTY.

```
claude  ████████░░  78% of 5hr window (Max est.)  │ 42.3k tok  $0.18  1.2k/5min ↑
codex   ███░░░░░░░  31% of Plus plan est.          │ 18.1k tok         7 threads
gemini  █░░░░░░░░░  12% of daily quota (live)      │  8.4k tok
cursor  ████░░░░░░  ↑ above 7-day avg              │ (experimental)
```

---

## Build and distribution

```sh
# local (Mac)
go build -o llmstat ./cmd/llmstat

# Pi (ARM64 Linux)
GOOS=linux GOARCH=arm64 go build -o llmstat-linux-arm64 ./cmd/llmstat

# M3 Mac
GOOS=darwin GOARCH=arm64 go build -o llmstat-darwin-arm64 ./cmd/llmstat
```

Add `./scripts/build_all.sh` once stable.

---

## Milestones

1. **Scaffold** — Go module, Provider interface, config loader, tiers loader
2. **Claude provider** — port logic from rogkit `clu`; reference implementation
3. **Display layer** — bar renderer, column alignment, colour, plain-text fallback, JSON
4. **Setup wizard** — `llmstat setup`; detects providers, writes config
5. **Codex provider** — SQLite reader via `modernc.org/sqlite`
6. **Tiers system** — lookup, stale-tier warning, bundled tiers.toml
7. **Gemini provider** — telemetry log + subprocess `/stats` with fallback
8. **Cursor provider** — SQLite, experimental, graceful schema-mismatch handling
9. **Polish** — `-w` weekly, positional provider arg, README, build script

---

## Decisions made

- **Binary name**: `llmstat`; `alias aiu=llmstat` for those who want short
- **v1 providers**: Claude, Codex, Gemini, Cursor (experimental)
- **Single-provider drill-down**: positional arg (`llmstat claude`), not a flag
- **Cursor**: included in v1 as experimental; any error shows "unavailable", never crashes
- **Gemini /stats**: subprocess, not an API — falls back gracefully if unavailable
- **tiers.toml**: bundled only, no network fetch; new release updates tier data
- **Render model**: collect-all-then-render for column alignment
- **Error model**: per-provider isolation; one bad provider never affects others
- **DESIGN.md**: no personal config — safe to commit to a public repo
