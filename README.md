# llmstat

> **Experimental** — this is an early-stage utility. See [what it does and doesn't do](#what-it-does-and-doesnt-do) before relying on any numbers.

A local activity pulse for your AI coding tools.

```
claude  ░░░░░░░░░░  │ 4.8M tok  224k/5min  3 sessions
codex   ░░░░░░░░░░  │ 5.4M tok  3 sessions
gemini  ░░░░░░░░░░  │ 7.8k tok  1 session
cursor  ░░░░░░░░░░  │ enterprise · no local usage data
```

Reads local data files directly — no API calls, no accounts, works offline.

**Supported tools:** Claude (Anthropic), Codex (OpenAI), Gemini CLI, Cursor

---

## What it does and doesn't do

### What it does

- Tells you **roughly how much** you've been using each AI tool today, this week, or this month
- Shows your current **token burn rate** (tokens in the last 5 minutes) as a proxy for current activity
- Counts **sessions** started in the period
- Works **offline**, with no accounts or API keys required
- Outputs **JSON** for scripting

### What it doesn't do

**It cannot tell you how close you are to your usage limit.** This is the most important thing to understand. Anthropic, OpenAI, and other providers enforce limits server-side and do not expose remaining quota in any local file. There is no reliable way to show a "X% of limit used" bar for Claude Code — we tried, and the numbers were consistently misleading. The progress bar is only shown where a verified limit is known.

**Token counts are approximate**, not billing-accurate. Local logs were not designed for accounting:
- Claude logs contain duplicate records that must be deduplicated by message ID
- Codex stores cumulative thread totals, not per-message events
- Cache reads (which can dwarf actual input/output tokens) are excluded from counts since they don't appear to count against limits the same way
- Counts are per-device — if you use the same tool on multiple machines, each machine only sees its own activity

**If you need accurate cross-machine usage data**, see [OpenUsage](#see-also), which takes a daemon + API approach instead.

---

## Install

Requires [Go 1.21+](https://go.dev/doc/install).

```sh
go install github.com/rdubar/llmstat/cmd/llmstat@latest
```

Make sure `$GOPATH/bin` is on your PATH (one-time setup if not already done):

```sh
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc && source ~/.zshrc
```

Then run setup to configure your tier:

```sh
llmstat --setup
```

## Usage

```sh
llmstat            # today's usage for all detected tools
llmstat -w         # this week's usage
llmstat -m         # this month's usage
llmstat claude     # single provider detail
llmstat --json     # machine-readable JSON output
llmstat -u         # upgrade to latest version
llmstat -v         # version and build info
llmstat --credits  # credits and provenance
```

## Upgrade

```sh
llmstat --upgrade   # or: llmstat -u
```

## How it works

| Tool   | Data source |
|--------|-------------|
| Claude | `~/.claude/projects/**/*.jsonl` (tokens), `~/.claude/sessions/` (session count) |
| Codex  | `~/.codex/state_5.sqlite` |
| Gemini | `~/.gemini/tmp/*/chats/session-*.json` — token counts per response |
| Cursor | `~/Library/Application Support/Cursor/User/globalStorage/state.vscdb` — tier only |

Each provider is auto-detected. If the data file/directory exists, it appears in output.

## Config

Config lives at `~/.config/llmstat/config.toml`. Running `--setup` creates it interactively. You can also edit it directly:

```toml
[claude]
tier = "pro"   # pro | max5x | max20x

[codex]
tier = "plus"
```

## Known data quality issues

**Claude — duplicate log records**
The JSONL logs emit multiple records per API call (one per content block — thinking block,
then text response), each carrying the full usage payload. llmstat deduplicates by
`message.id`. Without this, totals are roughly 2× inflated.

**Claude — no limit bar**
Anthropic publishes no token limits for Claude Code CLI. Community estimates derived from
the claude.ai web interface are empirically wrong for Claude Code — heavy users don't hit
limits at those figures. The bar is omitted rather than showing a misleading number.

**Claude — cache reads excluded from counts**
`cache_read_input_tokens` can be 20–30× larger than actual input+output tokens (large
contexts are re-read on every request). These are excluded from displayed totals since
they don't appear to count against rolling limits the same way.

**Codex — cumulative token totals**
`~/.codex/state_5.sqlite` stores a running `tokens_used` per thread, not per-message
events. Only threads *created* within the window are counted — threads started before
the window but still active are missed.

**Gemini — session file filtering**
Token counts are filtered by per-message timestamp, which is accurate. Files with no
timestamps fall back to file modification time.

**Cursor — no usage data**
Subscription tier is detected locally; actual usage is cloud-side only.

**All providers — per-device only**
Counts reflect this machine only. No cross-machine aggregation.

## Future directions

The most meaningful improvement would be provider API integration for cross-machine
totals and verified limit data:

| Provider | API availability | Notes |
|----------|-----------------|-------|
| **OpenAI (Codex)** | Yes — `GET /v1/usage` | Daily token counts by model; requires API key |
| **Anthropic (Claude)** | Not yet public | No programmatic usage endpoint for subscriptions |
| **Gemini** | Partial — Vertex AI only | Free tier has no usage API |
| **Cursor** | No | No public API |

## See also

[OpenUsage](https://github.com/janekbaraniewski/openusage) — runs a local daemon and pulls
usage from provider APIs (17+ providers). Responds instantly (pre-computed), aggregates
across machines, and shows rolling window data. A better choice if you need accuracy over
simplicity.

## Contributing

Contributions welcome — bug reports, tier data corrections, and new provider implementations especially. Please open an issue before starting significant work so we can align on approach.

## Credits

Built and maintained by [Roger Dubar](https://github.com/rdubar), with development assistance from Claude (Anthropic) and Codex (OpenAI).

With thanks to [Alphapet Ventures](https://alpha.pet).

## License

MIT — see [LICENSE](LICENSE).
