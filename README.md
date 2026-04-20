<h3 align="center">wt</h3>
<p align="center">Per-repo hooks, shell integration, project-aware prompt, and operator event bridge — one install for every repo.</p>

<p align="center">
  <a href="https://warrencommand.dev"><img src="https://img.shields.io/badge/warrencommand.dev-0a1220?style=for-the-badge&logoColor=white" alt="Website" /></a>
  <a href="https://github.com/wtchronos/wt-cli/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=for-the-badge" alt="MIT License" /></a>
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
</p>

---

### What this is

A single Go binary that turns any git repo into a `wt`-aware project via a `.wt.toml` file. Shell integration auto-loads project aliases + env when you `cd` in, git hook dispatchers fire declared commands on lifecycle events, and an operator bridge emits structured events to a remote surface (Cortix) with local JSONL queue + replay for offline reliability.

Built for one person's workflow across ~20 repos on Mac + VPS. Public because the shape is general.

---

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/wtchronos/wt-cli/main/install.sh | sh
```

Or build from source:

```bash
go install github.com/wtchronos/wt-cli@latest
```

## Commands

| Command | Purpose |
|---|---|
| `wt init` | Create `.wt.toml` + install git hook dispatchers |
| `wt status` | Project overview — git state, hooks, aliases, scripts, env |
| `wt run <script>` | Run a named script from `[scripts]` with project env |
| `wt deploy [target]` | Deploy via `[scripts] deploy` + emit operator events |
| `wt health` | Check project + operator health |
| `wt emit <type> <msg>` | Send events to operator surface (Cortix) |
| `wt events` | Show local event log |
| `wt sync` | Flush queued events to operator surface |
| `wt env show/export` | Display or inject project environment variables |
| `wt shell init <shell>` | Emit eval-able shell integration (bash/zsh/fish) |
| `wt hook run <event>` | Run project hooks for a git/lifecycle event |
| `wt prompt` | Print colored prompt segment |
| `wt aliases --load/--unload` | Project-scoped shell aliases |
| `wt completion <shell>` | Shell completions |
| `wt agent` | Query Cortix for active services and agent status |
| `wt log [-n 10] [-f] [-s source]` | Tail unified ops log (events, ops, audit) |
| `wt intent <desc> [-p P0/P1/P2]` | Submit intent to Cortix intent bridge |
| `wt version` | Version info |

## Config (`.wt.toml`)

Place in any git repo root. `wt` walks ancestors to find the nearest config.

```toml
[project]
name = "kairos-w"

[prompt]
segment = '{{cyan (printf "[%s] " .Project.Name)}}'

[hooks.git]
pre-commit = ["uv run ruff check ."]
post-checkout = ["./scripts/sync.sh"]
post-merge = ["uv run pip install -r requirements.txt"]

[hooks.enter]
commands = ["echo entering {{.Project.Name}}"]

[hooks.leave]
commands = []

[aliases]
t = "PYTHONPATH=. uv run pytest tests/ -q"
lint = "uv run ruff check . --fix"
deploy = "bash scripts/deploy-rotation-fix.sh"

[env]
PYTHONPATH = "."
KAIROS_ENV = "development"

[scripts]
test = "PYTHONPATH=. uv run pytest tests/ -q"
lint = "uv run ruff check . --fix"
deploy = "bash scripts/deploy-rotation-fix.sh"
health = "bash scripts/regression-guard.sh"

[operator]
cortix_url = "https://command.warrencommand.dev"
tags = ["active", "python"]
```

## Shell Setup

```bash
# Zsh — add to ~/.zshrc
eval "$(wt shell init zsh)"

# Bash — add to ~/.bashrc
eval "$(wt shell init bash)"

# Fish
wt shell init fish | source
```

This gives you:
- Auto-load/unload aliases when you `cd` into/out of a wt project
- Auto-inject `[env]` variables per project
- `wtr` shortcut for `wt run`
- Project-aware prompt segment

## Operator Integration

The `[operator]` block connects your project to the operator surface (Cortix).

```bash
# Emit events
wt emit deploy "shipped v2.1"
wt emit test "142 tests passing" --meta branch=main,coverage=87

# Check health (project + operator)
wt health

# Flush queued events
wt sync

# Deploy with event emission
wt deploy
```

Events are structured JSON with source, type, project, tags, and metadata.
When Cortix is unreachable, events queue locally to `.wt/events.jsonl`
and replay on `wt sync`.

## License

MIT
