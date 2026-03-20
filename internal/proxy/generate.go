package proxy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func projectName(projectDir string) string {
	return filepath.Base(projectDir)
}

func GenerateEnforcer(projectDir string) error {
	proxyDir := filepath.Join(projectDir, ".ccdc", "proxy")
	if err := os.MkdirAll(proxyDir, 0o755); err != nil {
		return fmt.Errorf("failed to create proxy directory: %w", err)
	}

	content := `"""
ccdc network policy enforcer for mitmproxy
Edit RULES to customize access control. Hot-reloads on save.
"""
from mitmproxy import http, ctx

# Domain allowlist with optional method/path restrictions
# Format:
#   "domain": "allow_all"                    - all methods/paths allowed
#   "domain": [{"method": "GET"}]            - GET only, all paths
#   "domain": [{"method": "GET", "path": "/api/*"}]  - GET on specific path
#   "domain": [{"method": "POST", "path": "/exact"}] - POST on exact path
#
# Path matching:
#   "/foo/*"  - prefix match (anything starting with /foo/)
#   "/foo"    - exact match
RULES = {
    # GitHub - GET only by default. Add your repo to allow push/pull:
    # {"method": "POST", "path": "/YourOrg/YourRepo.git/git-upload-pack"},
    # {"method": "POST", "path": "/YourOrg/YourRepo.git/git-receive-pack"},
    "github.com": [
        {"method": "GET"},
    ],
    "api.github.com": [
        {"method": "GET"},
        {"method": "POST", "path": "/graphql"},
    ],
    "raw.githubusercontent.com": "allow_all",

    # Package registries
    "registry.npmjs.org": "allow_all",
    "npm.pkg.github.com": "allow_all",
    "rubygems.org": "allow_all",
    "bundler.io": "allow_all",
    "pypi.org": "allow_all",
    "files.pythonhosted.org": "allow_all",

    # Claude Code
    "claude.ai": "allow_all",
    "platform.claude.com": "allow_all",
    "api.anthropic.com": "allow_all",
    "statsig.anthropic.com": "allow_all",
    "sentry.io": "allow_all",
}


def _match_path(pattern, path):
    if "/**" in pattern:
        prefix = pattern.split("/**")[0]
        suffix = pattern.split("/**")[1]
        return path.startswith(prefix) and path.endswith(suffix)
    if pattern.endswith("/*"):
        return path.startswith(pattern[:-1])
    return path == pattern


def request(flow: http.HTTPFlow):
    host = flow.request.pretty_host

    rule = RULES.get(host)
    if rule is None:
        flow.response = http.Response.make(403, b"Blocked: domain not in allowlist")
        ctx.log.warn(f"DENY {flow.request.method} {host}{flow.request.path}")
        return

    if rule == "allow_all":
        ctx.log.info(f"ALLOW {flow.request.method} {host}{flow.request.path}")
        return

    for entry in rule:
        if entry["method"] != flow.request.method:
            continue
        if "path" not in entry:
            ctx.log.info(f"ALLOW {flow.request.method} {host}{flow.request.path}")
            return
        if _match_path(entry["path"], flow.request.path):
            ctx.log.info(f"ALLOW {flow.request.method} {host}{flow.request.path}")
            return

    flow.response = http.Response.make(403, b"Blocked: method/path not allowed")
    ctx.log.warn(f"DENY {flow.request.method} {host}{flow.request.path}")
`

	return os.WriteFile(filepath.Join(proxyDir, "enforcer.py"), []byte(content), 0o644)
}

func GenerateDevDockerfile(projectDir string, docker bool, joy bool) error {
	devDir := filepath.Join(projectDir, ".ccdc", "dev")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		return fmt.Errorf("failed to create dev directory: %w", err)
	}

	name := projectName(projectDir)

	var b strings.Builder
	b.WriteString(`FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    git \
    curl \
    ca-certificates \
    sudo \
    gpg \
    && rm -rf /var/lib/apt/lists/* \
    && curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" > /etc/apt/sources.list.d/github-cli.list \
    && apt-get update && apt-get install -y gh && rm -rf /var/lib/apt/lists/*
`)

	if docker {
		b.WriteString(`
# Install Docker CLI only (daemon runs on host via socket-proxy)
RUN curl -fsSL https://get.docker.com | sh
`)
	}

	b.WriteString(`
# Create ccdc user
RUN useradd -m -s /bin/bash ccdc
`)

	if docker {
		b.WriteString("RUN usermod -aG docker ccdc\n")
	}

	b.WriteString(`
# Install Claude Code
RUN su - ccdc -c 'curl -fsSL https://claude.ai/install.sh | bash'

# Add claude to PATH and create ccdc wrapper
RUN ln -s /home/ccdc/.local/bin/claude /usr/local/bin/claude && \
    printf '#!/bin/sh\nexec /home/ccdc/.local/bin/claude --dangerously-skip-permissions "$@"\n' > /usr/local/bin/ccdc && \
    chmod +x /usr/local/bin/ccdc

# Trust mitmproxy CA certificate at login
RUN echo 'if [ -f /etc/mitmproxy/mitmproxy-ca-cert.pem ]; then sudo cp /etc/mitmproxy/mitmproxy-ca-cert.pem /usr/local/share/ca-certificates/mitmproxy.crt && sudo update-ca-certificates 2>/dev/null; fi' >> /home/ccdc/.bashrc

# Allow ccdc to run update-ca-certificates without password
RUN echo 'ccdc ALL=(ALL) NOPASSWD: /usr/sbin/update-ca-certificates, /usr/bin/cp' >> /etc/sudoers.d/ccdc


# Copy host gitconfig on login (if not already present)
RUN echo '[ -f /etc/gitconfig.host ] && [ ! -f ~/.gitconfig ] && cp /etc/gitconfig.host ~/.gitconfig' >> /home/ccdc/.bashrc

# Copy /etc/claude/ to ~/.claude/ on bash login
RUN echo 'mkdir -p ~/.claude && for item in /etc/claude/*; do [ -e "$item" ] && cp -r "$item" ~/.claude/$(basename "$item"); done' >> /home/ccdc/.bashrc
`)

	b.WriteString(`
# Install mise and project tools
RUN su - ccdc -c 'curl https://mise.run | sh'
ENV PATH="/home/ccdc/.local/bin:${PATH}"
COPY .mise.toml /home/ccdc/.config/mise/config.toml
RUN chown -R ccdc:ccdc /home/ccdc/.config/mise
RUN su - ccdc -c 'mise trust ~/.config/mise/config.toml && mise install'
ENV PATH="/home/ccdc/.local/share/mise/shims:${PATH}"
`)

	if joy {
		b.WriteString(`
# Install joy plugin
RUN su - ccdc -c 'claude plugin marketplace add https://github.com/yumazak/joy && claude plugin install joy-hooks@joy'
`)
	}

	fmt.Fprintf(&b, "\nWORKDIR /%s\n", name)

	return os.WriteFile(filepath.Join(devDir, "Dockerfile"), []byte(b.String()), 0o644)
}

func GenerateMiseToml(projectDir string) error {
	path := filepath.Join(projectDir, ".ccdc", "dev", ".mise.toml")
	if _, err := os.Stat(path); err == nil {
		return nil // already exists, don't overwrite
	}

	content := `# Add project-specific tools here
# See https://mise.jdx.dev/ for available tools
#
# [tools]
# node = "24.13.1"
# pnpm = "10.29.3"
# ruby = "3.3.6"
# python = "3.12"
`
	return os.WriteFile(path, []byte(content), 0o644)
}

func buildNoProxy(withDocker bool, withJoy bool) string {
	noProxy := "localhost,127.0.0.1,proxy"
	if withDocker {
		noProxy += ",socket-proxy"
	}
	if withJoy {
		noProxy += ",joy-proxy"
	}
	return noProxy
}

func GenerateCompose(projectDir string, docker bool, joy bool) error {
	name := projectName(projectDir)

	var b strings.Builder

	fmt.Fprintf(&b, "name: ccdc-%s\n\n", name)

	b.WriteString(`services:
  proxy:
    image: mitmproxy/mitmproxy:latest
    command: mitmdump --listen-port 3128 -s /rules/enforcer.py
    volumes:
      - ./proxy/enforcer.py:/rules/enforcer.py:ro
      - mitmproxy-certs:/home/mitmproxy/.mitmproxy
    extra_hosts:
      - "host.docker.internal:host-gateway"
    networks:
      - restricted
      - external
`)

	if docker {
		b.WriteString(`
  socket-proxy:
    image: tecnativa/docker-socket-proxy
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      CONTAINERS: 1
      EXEC: 1
      ALLOW_START: 0
      ALLOW_STOP: 0
      ALLOW_RESTARTS: 0
      IMAGES: 0
      VOLUMES: 0
      NETWORKS: 0
      BUILD: 0
      AUTH: 0
      SECRETS: 0
      SWARM: 0
      POST: 1
    networks:
      - restricted
`)
	}

	if joy {
		b.WriteString(`
  joy-proxy:
    image: caddy:latest
    command: caddy reverse-proxy --from :50055 --to host.docker.internal:50055
    extra_hosts:
      - "host.docker.internal:host-gateway"
    networks:
      - restricted
      - external
`)
	}

	fmt.Fprintf(&b, `
  dev:
    build:
      context: dev
      dockerfile: Dockerfile
    command: sleep infinity
    user: ccdc
    volumes:
      - ..:%s
      - ~/.claude/skills:/etc/claude/skills:ro
      - ~/.claude/agents:/etc/claude/agents:ro
      - ~/.claude/commands:/etc/claude/commands:ro
      - ~/.claude/CLAUDE.md:/etc/claude/CLAUDE.md:ro
      - ~/.gitconfig:/etc/gitconfig.host:ro
      - mitmproxy-certs:/etc/mitmproxy:ro
    working_dir: %s
    environment:
      - http_proxy=http://proxy:3128
      - https_proxy=http://proxy:3128
      - HTTP_PROXY=http://proxy:3128
      - HTTPS_PROXY=http://proxy:3128
      - no_proxy=%s
      - SSL_CERT_FILE=/etc/mitmproxy/mitmproxy-ca-cert.pem
      - NODE_EXTRA_CA_CERTS=/etc/mitmproxy/mitmproxy-ca-cert.pem
      - REQUESTS_CA_BUNDLE=/etc/mitmproxy/mitmproxy-ca-cert.pem
      - GIT_SSL_CAINFO=/etc/mitmproxy/mitmproxy-ca-cert.pem
`, "/"+name, "/"+name, buildNoProxy(docker, joy))

	if docker {
		b.WriteString("      - DOCKER_HOST=tcp://socket-proxy:2375\n")
	}
	if joy {
		b.WriteString("      - JOY_URL=http://joy-proxy:50055/hooks\n")
	}

	b.WriteString(`    depends_on:
      - proxy
    networks:
      - restricted

volumes:
  mitmproxy-certs:

networks:
  restricted:
    driver: bridge
    internal: true
  external:
    driver: bridge
`)

	path := filepath.Join(projectDir, ".ccdc", "compose.yaml")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
