# claude-usage

> Know how much of your Claude.ai plan you've burned — from your terminal, shell prompt, or menu bar.

`claude-usage` is a tiny cross-platform CLI that reads the **Plan Usage** numbers from your Claude.ai account (Current session, Weekly all-models, Sonnet, Design) and surfaces them where you actually work: a one-shot `status` command, a live TUI, a shell-prompt segment, or a system tray icon.

<!-- TODO: replace with vhs/asciinema gif -->
<p align="center"><em>[demo gif goes here — record with vhs after Phase 1]</em></p>

Numbers are battery-style: the percentage and bar represent how
much capacity is **left** in each bucket (100% = untouched, 0% =
at the plan ceiling). Threshold colors are still decided from raw
usage so warn/alert fire at the same place they always did.

```text
$ claude-usage status
Session   █████████░░░░░░░░░░░  49%  (resets in 1h 5m)
Weekly    █████████████████░░░  87%  (resets Sun 3:00 PM)
Sonnet    ████████████████████ 100%
Design    ████████████████████ 100%
```

---

## Install

```bash
# macOS (Homebrew)
brew install tonydisco/tap/claude-usage

# Windows (Scoop)
scoop bucket add tonydisco https://github.com/tonydisco/scoop-bucket
scoop install claude-usage

# Linux / fallback (any OS)
curl -fsSL https://raw.githubusercontent.com/tonydisco/claude-usage/main/install.sh | bash
```

> Binaries are signed and published from GitHub Actions via `goreleaser`. Source: [`/.github/workflows/release.yml`](.github/workflows/release.yml).

> **Tray / daemon need a CGO build.** The release binaries above ship with `CGO_ENABLED=0` for clean cross-compilation, so `claude-usage tray` and `claude-usage daemon start` are stubs in those builds. For the working menu-bar icon and background daemon, install from source on macOS/Linux:
>
> ```bash
> go install github.com/tonydisco/claude-usage/cmd/claude-usage@latest
> ```
>
> Everything else (`status`, `watch`, `prompt`, `login`, etc.) works in both build modes.

## Quick start

```bash
# 1. Paste your claude.ai session cookie (stored in OS keychain, not on disk)
claude-usage login

# 2. Check status
claude-usage status

# 3. Embed in your zsh prompt
PROMPT='%~ $(claude-usage prompt) ❯ '
# → ~/code [51%/13%] ❯

# 4. (optional) Tray icon + background notifications
claude-usage daemon start
```

## Features

- **Cross-platform single binary** — macOS (Intel + Apple Silicon), Windows, Linux. No runtime to install.
- **Three frontends in one tool** — CLI, live TUI (`watch`), shell-prompt segment (`prompt`), and tray icon (`daemon`).
- **Tracks Claude.ai plan usage** (Pro / Max), not just Claude Code token costs.
- **Secure auth** — session cookie is stored in macOS Keychain / Windows Credential Manager via [`zalando/go-keyring`](https://github.com/zalando/go-keyring), never written to a plaintext config file.
- **Configurable thresholds & notifications** — warn at 80% / 95% by default, override in `~/.config/claude-usage/config.toml`.
- **oh-my-zsh plugin included** — drop into `$ZSH_CUSTOM/plugins/claude-usage` and add `claude-usage` to your `plugins=(...)`.

## How is this different from existing tools?

There are several great Claude-related usage trackers already. They mostly target **Claude Code's local JSONL files**. `claude-usage` targets the **Plan dashboard on claude.ai itself**, which is the number that actually decides whether you can keep chatting today.

| Tool | What it tracks | Stack | Platforms | UI surfaces |
|---|---|---|---|---|
| **claude-usage** (this) | claude.ai **plan** usage (session + weekly buckets) | Go, single binary | macOS · Windows · Linux | CLI · TUI · shell prompt · tray |
| [ccusage](https://github.com/ryoppippi/ccusage) | Claude **Code** token cost (local JSONL) | Node/TS, `npx` | Any with Node | CLI · statusline |
| [Claude-Code-Usage-Monitor](https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor) | Claude **Code** real-time, with ML predictions | Python | Any with Python | Terminal TUI |
| [Claude-Usage-Tracker](https://github.com/hamed-elfayome/Claude-Usage-Tracker) | claude.ai plan usage | Swift / SwiftUI | macOS only | Menu bar (native) |
| [Claude-Usage-Extension](https://github.com/lugia19/Claude-Usage-Extension) | claude.ai plan usage | Browser extension | Browser only | In-page overlay |

**Pick `claude-usage` if** you live in a terminal and want plan info in your shell prompt, or you need a cross-platform tray icon on Windows/Linux.

**Pick ccusage** if you mostly use Claude Code from the CLI and care about token cost per project.

**Pick Claude-Usage-Tracker** if you're macOS-only and want a polished native menu bar app.

## Commands

| Command | Purpose |
|---|---|
| `claude-usage login` | Paste session cookie, store it in OS keychain |
| `claude-usage status` | Print current usage once (colored when TTY) |
| `claude-usage watch` | Live TUI, refreshes every 60s, `r` to refresh manually |
| `claude-usage prompt` | Compact `[51%/13%]` for embedding in `PS1` |
| `claude-usage daemon start\|stop\|status` | Background process + tray icon |
| `claude-usage config <key> <value>` | Edit threshold / poll interval / theme |
| `claude-usage logout` | Remove credential from keychain |
| `claude-usage completion zsh\|bash\|fish` | Generate shell completion |

## Configuration

`~/.config/claude-usage/config.toml`:

```toml
poll_interval_seconds = 60
warn_threshold = 80   # show yellow when any bucket >= 80%
alert_threshold = 95  # show red + notify when any bucket >= 95%
notify = true
org_id = ""           # leave empty to auto-detect; set if you have multiple orgs
```

## How it works (and the honest caveat)

Anthropic doesn't publish an API for Claude.ai plan usage. This tool calls the **same internal endpoint** that the Settings → Usage page in your browser hits, authenticated with the session cookie you paste in via `login`.

**This means:**

- Anthropic may change the endpoint shape at any time. When that happens the tool will likely break until I ship a patch. Watch [Releases](../../releases) — patches usually land within 24–48h.
- Your session cookie is sensitive (it's a login). `claude-usage` stores it only in your OS keychain, and only the HTTPS request to `claude.ai` ever sees it.
- This is not affiliated with or endorsed by Anthropic.

The fetcher is isolated in [`internal/fetcher`](internal/fetcher) so endpoint patches are a single-file change.

## Roadmap

- [x] Phase 0 — Endpoint reconnaissance (see [`samples/usage-response.json`](samples/usage-response.json))
- [x] Phase 1 — Core CLI (`login`, `status`, `logout`)
- [x] Phase 2 — TUI (`watch`) + shell prompt + oh-my-zsh plugin
- [x] Phase 3 — Daemon + tray icon + threshold notifications
- [x] Phase 4 — Homebrew tap + install script *(Scoop bucket and Apple notarization pending)*

## Contributing

PRs welcome — especially for:
- **Endpoint patches** when claude.ai changes things (see `internal/fetcher/`)
- **Linux tray icon polish** (Wayland is the hard part)
- **Translations** of CLI output

Run the test suite with `go test ./...`. Mock data lives in `samples/`.

## License

MIT — see [LICENSE](LICENSE).
