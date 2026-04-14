# wt

Personal CLI tool — one install, every repo gets opinionated shell integration
and hook-driven automation.

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/wtchronos/wt-cli/main/install.sh | sh
```

## What It Does

| Command | Purpose |
|---|---|
| `wt init` | Create `.wt.toml` + install git-hook dispatchers |
| `wt shell init <shell>` | Emit `eval`-able shell integration (bash/zsh/fish) |
| `wt hook run <event>` | Called by git hooks — runs user-defined commands |
| `wt prompt` | Print prompt segment for current project |
| `wt aliases --load/--unload` | Emit alias / unalias lines for the shell |
| `wt completion <shell>` | Generate shell completions |
| `wt version` | Print version info |

## Config (`.wt.toml`)

Place in any git repo root. Walks ancestors to find nearest config.

```toml
[project]
name = "myapp"

[prompt]
segment = "[{{.Project.Name}}]"

[hooks.git]
post-checkout = ["./scripts/sync.sh"]
post-merge     = ["make deps"]

[hooks.enter]
commands = ["echo entering {{.Project.Name}}"]

[hooks.leave]
commands = []

[aliases]
serve = "go run ./cmd/server"
tt    = "go test ./..."
```

## Shell Setup

```bash
# Zsh (most common)
eval "$(wt shell init zsh)" >> ~/.zshrc

# Bash
eval "$(wt shell init bash)" >> ~/.bashrc

# Fish
wt shell init fish | source
```

## License

MIT © Warren Trepp
