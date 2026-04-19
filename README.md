# llmstat

Track your local AI tool usage in one line.

```
claude  ████████░░  78% of 5hr window  │ 42.0M tok  1.9M/5min
codex   ░░░░░░░░░░                     │ 5.4M tok  3 sessions
gemini  ░░░░░░░░░░                     │ 7.8k tok  1 sessions
cursor  ░░░░░░░░░░                     │ enterprise · no local usage data
```

Reads local data files directly — no API calls, no accounts, works offline.

**Supported tools:** Claude (Anthropic), Codex (OpenAI), Gemini CLI, Cursor

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

Then run setup to configure your tier (Claude Pro, Max, etc.):

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
| Claude | `~/.claude/projects/**/*.jsonl` |
| Codex  | `~/.codex/state_5.sqlite` |
| Gemini | `~/.gemini/tmp/*/chats/session-*.json` — token counts per response |
| Cursor | `~/Library/Application Support/Cursor/User/globalStorage/state.vscdb` — tier only (usage is cloud-side) |

Each provider is auto-detected. If the data file exists, it appears in output.

Tier limits (tokens per 5-hour window) are read from a bundled `tiers.toml` and calibrated periodically against published limits. Run `llmstat --setup` to set your tier. The bar turns yellow at 60% and red at 85%.

## Config

Config lives at `~/.config/llmstat/config.toml`. Running `--setup` creates it interactively. You can also edit it directly:

```toml
[claude]
tier = "max"

[codex]
tier = "plus"
```

## Contributing

Contributions welcome — bug reports, tier data corrections, and new provider implementations especially. Please open an issue before starting significant work so we can align on approach.

## Credits

Built and maintained by [Roger Dubar](https://github.com/rdubar), with development assistance from Claude (Anthropic) and Codex (OpenAI).

With thanks to [Alphapet Ventures](https://alpha.pet).

## License

MIT — see [LICENSE](LICENSE).
