# mehshell

A fast, parallelized prompt engine for zsh written in Go.

## Prompt Example

```text
╭─  ~/project  main ✓  v22.11.0                  ⎈ minikube  3s  04:20:47 PM
╰─❯
```

## Features

- Single Go binary, ~6ms per prompt.
- All segments run as parallel goroutines.
- Git dirty check with 150ms timeout. It never blocks, even in huge repos.
- Zero-fork git branch detection. It reads .git/HEAD directly.
- OS icon auto-detection for Arch, Ubuntu, Fedora, Debian, NixOS, and macOS.
- Runtime version detection for Node.js, Python, Go, Rust, Ruby, and Java when marker files are present.
- Cloud context support for Kubernetes, AWS, Azure, and GCP profiles.
- Terraform workspace and Docker context detection.
- Environment detection for Conda and virtualenv.
- Battery level with charge state icons.
- Command duration (3s+), time, and exit code coloring.
- Vi mode indicator (swaps prompt char on keymap change).
- Transient prompt (collapses previous prompt on Enter).
- Instant prompt (caches last prompt for zero-latency shell startup).
- Right-aligned segments on the first line.
- Nerd Font icons.

## Install

### Homebrew (macOS & Linux)

```bash
brew tap blackflame007/tap
brew install mehshell
```

### AUR (Arch Linux)

```bash
yay -S mehshell-bin
```

### Binary releases

Download a prebuilt binary from [GitHub Releases](https://github.com/blackflame007/mehshell/releases/latest):

```bash
# Linux (x86_64)
curl -sL https://github.com/blackflame007/mehshell/releases/latest/download/mehshell_linux_amd64.tar.gz | tar xz && mv mehshell ~/.local/bin/

# Linux (ARM64)
curl -sL https://github.com/blackflame007/mehshell/releases/latest/download/mehshell_linux_arm64.tar.gz | tar xz && mv mehshell ~/.local/bin/

# macOS (Apple Silicon)
curl -sL https://github.com/blackflame007/mehshell/releases/latest/download/mehshell_darwin_arm64.tar.gz | tar xz && mv mehshell ~/.local/bin/

# macOS (Intel)
curl -sL https://github.com/blackflame007/mehshell/releases/latest/download/mehshell_darwin_amd64.tar.gz | tar xz && mv mehshell ~/.local/bin/
```

### From source

```bash
go install github.com/blackflame007/mehshell@latest
```

## Zsh Integration

Add this to your `.zshrc`:

```zsh
eval "$(mehshell init zsh)"
```

## Benchmarks

| Metric | p10k | mehshell |
|---|---|---|
| Shell startup | 870ms+ | 64ms |
| Between commands | 6000ms (with vcs_info) | 28ms |
| Prompt generation | ~45ms | 6ms |

## Feature Comparison with Powerlevel10k

mehshell is intentionally minimal. p10k is a full-featured theme engine. Pick the right tool for your workflow.

| Feature | mehshell | p10k |
|---|---|---|
| **Architecture** | Single Go binary | Zsh scripts + gitstatusd daemon |
| **Dependencies** | Zero (Go stdlib only) | gitstatus binary (downloaded) |
| **Prompt generation** | ~6ms | ~45ms |
| **Shell startup impact** | ~64ms | ~870ms+ |
| **Configuration** | Recompile | `~/.p10k.zsh` + config wizard |
| **Async rendering** | Goroutines (parallel) | Zsh workers + gitstatusd |
| **Git branch detection** | Zero-fork (reads `.git/HEAD`) | gitstatusd (libgit2) |
| **Git dirty check** | `git status` with 150ms timeout | Async, never blocks |
| **Instant prompt** | ✓ (cache-based) | ✓ |
| **Transient prompt** | ✓ | ✓ |
| **Vi mode** | ✓ (prompt char swap) | ✓ (full indicator) |
| **Custom segments** | Add in source | Public API (`p10k segment`) |
| **Total segments** | 20 | 67+ |

### Segment Coverage

| Segment | mehshell | p10k |
|---|---|---|
| OS icon | ✓ | ✓ |
| Directory | ✓ | ✓ (smart truncation) |
| Git | ✓ (branch + dirty) | ✓ (branch, ahead/behind, stash, conflicts) |
| Node.js | ✓ | ✓ (+ nvm, nodenv, package name) |
| Python | ✓ | ✓ (+ pyenv, poetry) |
| Go | ✓ | ✓ (+ goenv) |
| Rust | ✓ | ✓ |
| Ruby | ✓ | ✓ (+ rvm, chruby) |
| Java | ✓ | ✓ (+ jenv) |
| Conda | ✓ | ✓ |
| Virtualenv | ✓ | ✓ |
| Kubernetes | ✓ | ✓ (+ show-on-command) |
| Terraform | ✓ | ✓ |
| Docker | ✓ | ✓ |
| AWS | ✓ | ✓ (+ Elastic Beanstalk) |
| Azure | ✓ | ✓ |
| GCP | ✓ | ✓ |
| Battery | ✓ | ✓ |
| Duration | ✓ | ✓ |
| Time | ✓ | ✓ (+ date) |
| CPU / RAM | — | ✓ |
| Vi mode | ✓ (prompt char) | ✓ (full indicator) |

> **Note**: p10k is in limited maintenance mode — no new features are in development.

## Requirements

- Go 1.22+ (build only)
- Nerd Font
- zsh

## License

MIT
