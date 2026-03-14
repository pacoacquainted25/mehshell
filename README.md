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
- Runtime version detection for Node.js, Python, and Go when marker files are present.
- Cloud context support for Kubernetes and AWS profiles.
- Environment detection for Conda and virtualenv.
- Command duration (3s+), time, and exit code coloring.
- Right-aligned segments on the first line.
- Nerd Font icons.

## Install

### Binary releases

```bash
curl -sL https://github.com/blackflame007/mehshell/releases/latest/download/mehshell_linux_amd64.tar.gz | tar xz && mv mehshell ~/.local/bin/
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

## Requirements

- Go 1.22+ (build only)
- Nerd Font
- zsh

## License

MIT
