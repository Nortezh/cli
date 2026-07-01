# ntzh

> Command-line client for the [Nortezh](https://nortezh.com) deployment platform.

[![Go Reference](https://pkg.go.dev/badge/github.com/nortezh/cli.svg)](https://pkg.go.dev/github.com/nortezh/cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/nortezh/cli)](https://goreportcard.com/report/github.com/nortezh/cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

`ntzh` lets you list projects, ship new container images, roll back to a previous
revision, inspect deployment history, and manage routes/domains — straight from
your terminal or CI pipeline.

## Table of Contents

- [Installation](#installation)
- [Quick start](#quick-start)
- [Authentication](#authentication)
- [Usage](#usage)
- [Configuration](#configuration)
- [Scripting & CI](#scripting--ci)
- [Shell completion](#shell-completion)
- [Development](#development)
- [License](#license)

## Installation

### Homebrew (macOS / Linux)

```sh
brew install Nortezh/tap/ntzh
```

### Install script (macOS / Linux)

Downloads the right prebuilt binary for your OS/arch into `/usr/local/bin`:

```sh
curl -fsSL https://raw.githubusercontent.com/Nortezh/cli/main/install.sh | sh
```

Override the version or target directory with `VERSION=v0.5.0` and
`BINDIR=$HOME/.local/bin`.

### Prebuilt binaries

Download an archive for your platform from the
[releases page](https://github.com/Nortezh/cli/releases), extract it, and put
`ntzh` somewhere on your `PATH`.

### From source

```sh
go install github.com/nortezh/cli/cmd/ntzh@latest
```

This installs the `ntzh` binary to `$(go env GOPATH)/bin`. Make sure that
directory is on your `PATH`.

### AI coding agent skill (optional)

`ntzh` bundles a `SKILL.md` that teaches AI coding agents how to drive
this CLI (deploy recipe, flag shapes, where to look for more help).
One command installs it for every supported agent:

```sh
ntzh skill install                    # install for Claude Code and Codex
ntzh skill install --target=claude    # only Claude Code (also read by opencode)
ntzh skill install --target=codex     # only OpenAI Codex
ntzh skill install --force            # overwrite existing copies
```

Install locations:

| Target  | Path                                | Agents that read it      |
| ------- | ----------------------------------- | ------------------------ |
| claude  | `~/.claude/skills/ntzh/SKILL.md`    | Claude Code, opencode    |
| codex   | `~/.agents/skills/ntzh/SKILL.md`    | OpenAI Codex CLI         |

### From a clone

```sh
git clone https://github.com/nortezh/cli.git
cd cli
make install        # go install ./cmd/ntzh
# or
make build          # produces ./ntzh
```

Requires Go 1.26 or newer.

### Updating

`ntzh` can update itself in place:

```sh
ntzh upgrade            # check GitHub and install the latest release
ntzh upgrade --check    # only report whether a newer version exists
ntzh update             # alias for 'ntzh upgrade'
```

Replacing the binary requires write access to its directory. If `ntzh` lives in
a system path (e.g. `/usr/local/bin`), re-run with `sudo` or use the install
script above. Homebrew installs should be updated with `brew upgrade` instead.

## Quick start

```sh
ntzh login                                        # open the browser, sign in
ntzh project list
ntzh deployment list --project=acme
ntzh deployment deploy staging-bo \
  --project=acme \
  --image=ghcr.io/acme/api:v1.2.3 \
  --location=bkk-1
```

## Authentication

`ntzh` supports two credential types. Both are stored at
`~/.config/ntzh/credentials.json` with mode `0600`.

| Mode               | Command                                                                    | When to use                |
| ------------------ | -------------------------------------------------------------------------- | -------------------------- |
| **Browser**        | `ntzh login`                                                               | Interactive use on a laptop |
| **Service account**| `ntzh login --service-account=ci@acme.com --key-file=./key.txt`            | CI / headless environments  |

```sh
ntzh whoami         # show current identity
ntzh logout         # remove stored credentials
```

> **Heads up:** Browser tokens expire after **7 days** and are **not refreshed
> automatically**. Re-run `ntzh login` when prompted. Service-account
> credentials do not expire.

## Usage

### Projects

```sh
ntzh project list
ntzh project list --output=json
```

### Deployments

`--project` accepts a project name, slug (the `no` field), or internal ID.
`--location` (cluster ID) is auto-detected via `deployment.list` when omitted.

```sh
# List
ntzh deployment list --project=<project>

# Inspect one
ntzh deployment get <deployment> --project=<project> --location=<location>

# Ship a new image (optionally patch env, ports, replicas in the same revision)
ntzh deployment deploy <deployment> \
  --project=<project> \
  --image=<image> \
  --location=<location> \
  [--set-env KEY=VALUE ...] [--remove-env KEY ...] \
  [--port <n>] [--protocol <p>] [--internal=<bool>] \
  [--min-replica <n>] [--max-replica <n>]

# Roll back to a previous revision
ntzh deployment rollback <deployment> \
  --project=<project> \
  --to=<revision> \
  --location=<location>

# Revision history (newest first)
ntzh deployment revisions <deployment> --project=<project> --location=<location>
```

#### Example

```sh
ntzh deployment deploy staging-bo \
  --project=acme \
  --image=ghcr.io/acme/api:v1.2.3 \
  --location=bkk-1
```

Deployment list fields: `name`, `type`, `status`, `location`, `replicas`,
`last_deployed` (lowercased in TOON; uppercased columns in table mode).

`ntzh deployment get <name>` prints the current env (`env.KEY` keys in TOON,
`ENV:KEY` rows in table mode, or the `env` object under `--output=json`). Use `--set-env` / `--remove-env`
on `deploy` to patch it — each flag is optional, omitted flags leave the value
unchanged.

### Domains

```sh
ntzh domain list   --project=<project>
ntzh domain get    <domain> --project=<project>
ntzh domain create <domain> --project=<project> --location=<location> [--wildcard] [--cdn]
ntzh domain delete <domain> --project=<project>
```

### Routes

A route binds `(domain, path)` to a web-service deployment. The owning project
must already have the domain registered.

```sh
ntzh route list   --project=<project> [--search=<q>]
ntzh route get    --project=<project> --domain=<domain> --path=<path>
ntzh route create --project=<project> --domain=<domain> --path=<path> --target=<deployment> \
  [--location=<location>] [--rewrite-path=<expr>] [--skip-domain-verify]
ntzh route delete --project=<project> --domain=<domain> --path=<path>
```

`--target` accepts a deployment name (e.g. `api-prod`); the CLI prepends
`deployment://` for you. `--location` is auto-resolved from the target
deployment when omitted.

## Configuration

`ntzh` reads two files from `~/.config/ntzh/`:

```
~/.config/ntzh/config.json       # { "server": "https://api.nortezh.com" }
~/.config/ntzh/credentials.json  # 0600 — bearer or service_account
```

### Environment variables

| Variable          | Purpose                                                  |
| ----------------- | -------------------------------------------------------- |
| `NTZH_SERVER`     | Override the API server URL                              |
| `NTZH_PROJECT`    | Default `--project` for project-scoped commands          |
| `NTZH_LOCATION`   | Default `--location`, skips the `deployment.list` lookup |
| `NTZH_CONFIG_DIR` | Override `~/.config/ntzh`                                |

### Precedence

```
flag  >  env var  >  config file  >  default
```

## Scripting & CI

| Flag             | Purpose                                                                        |
| ---------------- | ------------------------------------------------------------------------------ |
| `--output=toon`  | Default: compact, agent-friendly output (lists end with a `count:` line)       |
| `--output=json`  | Emit raw structured responses (parse with `jq`)                                |
| `--output=table` | Aligned human-readable columns                                                 |
| `--debug`        | Log HTTP request/response to stderr (Authorization header is redacted)         |

```sh
# GitHub Actions example
- name: Deploy
  env:
    NTZH_PROJECT: acme
  run: |
    ntzh login \
      --service-account=ci@acme.com \
      --key-file=<(echo "$NTZH_KEY")
    ntzh deployment deploy api \
      --image=ghcr.io/acme/api:${{ github.sha }} \
      --location=bkk-1
```

All commands exit non-zero on failure; output and structured errors (an `error:`
line, plus a `help:` line with the fix when known) are written to stdout.

## Shell completion

`ntzh` ships completion scripts for `bash`, `zsh`, `fish`, and `powershell`
via `ntzh completion <shell>`.

### zsh

Quick test in the current shell:

```sh
source <(ntzh completion zsh)
compdef _ntzh ntzh
```

Persistent install — make sure `compinit` is enabled in your `~/.zshrc`:

```sh
autoload -Uz compinit && compinit
```

Then write the script to a directory on your `fpath`:

```sh
ntzh completion zsh > "${fpath[1]}/_ntzh"
# or, on Homebrew:
ntzh completion zsh > "$(brew --prefix)/share/zsh/site-functions/_ntzh"
```

Restart the shell (`exec zsh`) and tab-complete on `ntzh `.

### bash / fish / powershell

```sh
ntzh completion bash --help
ntzh completion fish --help
ntzh completion powershell --help
```

Each subcommand prints shell-specific install instructions.

## Development

```sh
make test       # go test ./...
make build      # build ./ntzh
make lint       # golangci-lint run
```

Project layout:

```
cmd/ntzh/          # main entrypoint
internal/api/      # arpc HTTP client + typed wrappers
internal/auth/     # credential store (bearer, service account)
internal/cli/      # cobra command tree
internal/config/   # config file + env resolution
internal/output/   # TOON, table & JSON printers
```

## License

[MIT](LICENSE)
