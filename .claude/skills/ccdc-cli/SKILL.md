---
name: ccdc-cli
description: Guide agents through using the ccdc CLI for generating and managing sandboxed Claude Code environments with mitmproxy network control. Covers init, build, connect, enforcer customization, and troubleshooting. Trigger keywords - ccdc, ccdc init, sandbox, enforcer, mitmproxy, compose, network proxy, sandboxed claude, container setup, mise tools.
---

# ccdc CLI

Guide agents through using the `ccdc` CLI to generate and manage sandboxed Claude Code environments with L7 network proxy control.

## Overview

`ccdc` is a Go CLI tool that generates Docker-based sandboxed environments for running Claude Code with `--dangerously-skip-permissions`. It provides:

- **L7 network control** via mitmproxy with a Python-based enforcer script
- **Docker isolation** via `internal: true` networks (proxy-only egress)
- **Tool management** via mise (node, ruby, python, etc.)
- **Host settings sync** (`~/.claude/skills`, `agents`, `commands`, `CLAUDE.md`)
- **Optional joy notifications** via Caddy reverse proxy

**Companion skill**: For creating or modifying `enforcer.py` rules (domain allowlists, method/path restrictions), use the `generate-enforcer-policy` skill.

## Prerequisites

- macOS (Darwin arm64)
- Docker Desktop
- `ccdc` installed (`mise use -g github:yumazak/ccdc@latest` or `go install github.com/yumazak/ccdc@latest`)

---

## Workflow 1: Getting Started

### Step 1: Initialize the project

```bash
cd your-project
ccdc init
```

This generates the following files:

```
.ccdc/
├── proxy/
│   └── enforcer.py          # mitmproxy L7 network policy
├── dev/
│   ├── Dockerfile           # Claude Code container image
│   └── .mise.toml           # Project-specific tool versions
└── compose.yaml             # Docker Compose orchestration
```

### Step 2: Configure project tools

Edit `.ccdc/dev/.mise.toml` to specify required tools:

```toml
[tools]
node = "24.13.1"
pnpm = "10.29.3"
# ruby = "3.3.6"
# python = "3.12"
```

### Step 3: Configure network policy

Edit `.ccdc/proxy/enforcer.py` to customize allowed domains and methods. See the `generate-enforcer-policy` skill for guidance.

### Step 4: Build and start

```bash
docker compose -f .ccdc/compose.yaml up -d --build
```

### Step 5: Connect

```bash
docker compose -f .ccdc/compose.yaml exec dev bash
```

### Step 6: First-time setup (inside container)

```bash
gh auth login          # GitHub authentication
ccdc                   # Start Claude Code (alias for claude --dangerously-skip-permissions)
```

---

## Workflow 2: Init with Options

### Add joy notifications

```bash
ccdc init --joy
```

Adds a `joy-proxy` service (Caddy reverse proxy) that forwards notifications to `host.docker.internal:50055`. Installs the joy plugin into Claude Code.

---

## Workflow 3: Day-to-Day Usage

### Start the environment

```bash
docker compose -f .ccdc/compose.yaml up -d
```

### Connect

```bash
docker compose -f .ccdc/compose.yaml exec dev bash
```

### Stop

```bash
docker compose -f .ccdc/compose.yaml down
```

### Rebuild after Dockerfile or mise changes

```bash
docker compose -f .ccdc/compose.yaml build && docker compose -f .ccdc/compose.yaml up -d
```

Use `--no-cache` only if Docker is caching stale layers despite Dockerfile changes.

### View proxy logs (denied/allowed requests)

```bash
docker compose -f .ccdc/compose.yaml logs proxy -f
```

Look for `DENY` and `ALLOW` lines to debug network access issues.

---

## Architecture

```
┌─ restricted network (internal: true) ─────────────┐
│                                                     │
│  dev container                                      │
│  - Claude Code (ccdc wrapper)                       │
│  - mise-managed tools (node, pnpm, etc.)            │
│  - http_proxy=http://proxy:3128                     │
│  - All egress goes through proxy                    │
│                                                     │
│  proxy container (mitmproxy)                        │
│  - enforcer.py controls domain/method/path access   │
│  - Connected to both restricted + external networks │
│                                                     │
│  [optional] joy-proxy (caddy)                       │
│  - Notification forwarding to host                  │
│                                                     │
└─────────────────────────────────────────────────────┘
         │
    external network (proxy + joy-proxy only)
```

### Key design decisions

- **`internal: true` network**: The dev container cannot reach the internet directly. All traffic must go through the mitmproxy proxy.
- **mitmproxy for L7**: Enables per-domain, per-method, per-path access control with Python scripting.
- **TLS termination**: mitmproxy terminates TLS, inspects HTTP traffic, then re-encrypts. CA certificates are distributed via a shared Docker volume.
- **enforcer.py is mounted read-only**: The dev container cannot modify its own network policy. Edit from the host for hot-reload.
- **mise shims in PATH**: Tools are available via `ENV PATH` with shims, so they work in git hooks and non-interactive shells.
- **`gh auth login` for git**: No PATs stored. Users authenticate interactively once per container lifecycle.

---

## Generated Files Reference

### `.ccdc/proxy/enforcer.py`

mitmproxy addon script that enforces network policy. Rules are defined in the `RULES` dict:

```python
RULES = {
    "domain": "allow_all",                              # All methods/paths
    "domain": [{"method": "GET"}],                      # GET only
    "domain": [{"method": "POST", "path": "/api/*"}],   # POST on prefix
}
```

Path matching:
- `/foo/*` — prefix match (anything starting with `/foo/`)
- `/foo/**suffix` — prefix + suffix match
- `/foo` — exact match

Hot-reloads on save (mitmproxy watches the file).

### `.ccdc/dev/Dockerfile`

Ubuntu 24.04 based image with:
- git, curl, gh CLI
- ccdc user (non-root)
- Claude Code + `ccdc` wrapper
- mitmproxy CA trust via `.bashrc`
- Host `.gitconfig` copy on first login
- Host `~/.claude/` settings sync on login
- mise + project tools from `.mise.toml`
- [Optional] joy plugin

### `.ccdc/dev/.mise.toml`

mise configuration for project tools. Preserved across `ccdc init` re-runs (won't overwrite existing file).

### `.ccdc/compose.yaml`

Docker Compose file with services:
- `proxy` (mitmproxy) — always present
- `joy-proxy` (caddy) — with `--joy`
- `dev` — the development container

Project name: `ccdc-<directory-name>`

---

## Troubleshooting

### `node: command not found` inside container

mise shims may not be in PATH. Check:
```bash
echo $PATH | tr ':' '\n' | grep mise
```
Should include `/home/ccdc/.local/share/mise/shims`. If missing, rebuild the image.

### `mise ERROR No version is set for shim`

The `.mise.toml` was placed as a local config, not global. It should be at `~/.config/mise/config.toml` inside the container (handled by the Dockerfile's `COPY` instruction).

### Git hooks fail (`npx: not found`)

Same as above — mise shims must be in PATH for non-interactive shells.

### `DENY` in proxy logs for a needed domain

Edit `.ccdc/proxy/enforcer.py` on the host to add the domain. The change hot-reloads automatically.

### Git push/pull fails (403 from proxy)

By default, `github.com` is GET-only. Add POST rules for your repo's git operations:
```python
"github.com": [
    {"method": "GET"},
    {"method": "POST", "path": "/YourOrg/YourRepo.git/**"},
],
```

### TLS certificate errors

The mitmproxy CA certificate should be trusted on container startup via `.bashrc`. If it fails:
```bash
sudo cp /etc/mitmproxy/mitmproxy-ca-cert.pem /usr/local/share/ca-certificates/mitmproxy.crt
sudo update-ca-certificates
```

### Container can't resolve DNS

Expected behavior — the `internal: true` network has no external DNS. DNS resolution happens through the HTTP proxy. Ensure `http_proxy` / `https_proxy` environment variables are set.

---

## Quick Reference

| Task | Command |
|------|---------|
| Initialize project | `ccdc init [--joy]` |
| Build and start | `docker compose -f .ccdc/compose.yaml up -d --build` |
| Connect to container | `docker compose -f .ccdc/compose.yaml exec dev bash` |
| View proxy logs | `docker compose -f .ccdc/compose.yaml logs proxy -f` |
| Stop environment | `docker compose -f .ccdc/compose.yaml down` |
| Rebuild image | `docker compose -f .ccdc/compose.yaml build` |
| Start Claude Code | `ccdc` (inside container) |
| GitHub auth | `gh auth login` (inside container) |
| Edit network policy | Edit `.ccdc/proxy/enforcer.py` (from host) |
| Edit project tools | Edit `.ccdc/dev/.mise.toml` then rebuild |

## Companion Skills

| Skill | When to use |
|-------|------------|
| `generate-enforcer-policy` | Creating or modifying `enforcer.py` rules from natural-language requirements |
