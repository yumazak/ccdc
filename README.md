# ccdc

CLI tool that generates a [Dev Container Feature](https://containers.dev/implementors/features/) for running [Claude Code](https://docs.anthropic.com/en/docs/claude-code) with `--dangerously-skip-permissions` in isolated container environments.

> **Disclaimer:** This is an unofficial community tool and is not affiliated with or endorsed by Anthropic.

## Prerequisites

- macOS
- [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- [Dev Container CLI](https://github.com/devcontainers/cli) (`npm install -g @devcontainers/cli`)

## Install

```bash
# mise
mise use -g github:yumazak/ccdc@latest

# go install
go install github.com/yumazak/ccdc@latest
```

Binaries are also available on [GitHub Releases](https://github.com/yumazak/ccdc/releases).

## Usage

### 1. Generate the feature

```bash
cd your-project
ccdc init
```

This creates `.devcontainer/ccdc/` with the feature files.

### 2. Configure devcontainer.json

Add the feature and set the remote user in your `devcontainer.json`:

```jsonc
{
  "features": {
    "./ccdc": {}
  },
  "remoteUser": "ccdc"
}
```

### 3. Start the container

```bash
devcontainer up --workspace-folder .
devcontainer exec --workspace-folder . bash
```

### 4. Use Claude Code

Inside the container, run:

```bash
ccdc
```

This is a wrapper for `claude --dangerously-skip-permissions`.

## Generated Files

```
.devcontainer/ccdc/
├── devcontainer-feature.json   # Feature definition (mounts, postStartCommand)
└── install.sh                  # Creates ccdc user, installs Claude Code
```

## Configuration

### Host settings sync

The following files/directories from `~/.claude/` are automatically mounted into the container and copied at startup:

- `CLAUDE.md`
- `agents/`
- `commands/`
- `skills/`
- `projects/`

Only items that exist on the host are mounted. Changes inside the container do not affect the host.

## Security

### `--dangerously-skip-permissions`

This tool uses `--dangerously-skip-permissions`, which disables all permission prompts in Claude Code. **Only use this inside containers.** The generated feature creates a dedicated non-root `ccdc` user to mitigate risk, but Claude Code has full access to everything inside the container.

### Secrets and .env files

**Any file or environment variable accessible inside the container is also accessible to Claude Code.** There is no way to prevent this when using `--dangerously-skip-permissions`.

Recommendations:

- **Never put production secrets in the container.** Use development-only API keys with minimal permissions.
- **Use `.claudeignore`** to prevent Claude Code from intentionally reading sensitive files (note: this is not a hard security boundary).
- **Set usage limits and alerts** on any API keys used in the container.
- Consider tools like [dotenvx](https://dotenvx.com/) to avoid storing plaintext secrets on disk.

## License

[MIT](LICENSE)
