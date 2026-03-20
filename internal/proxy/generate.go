package proxy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var defaultDomains = []string{
	"github.com",
	"api.github.com",
	"raw.githubusercontent.com",
	"registry.npmjs.org",
	"npm.pkg.github.com",
	"rubygems.org",
	"bundler.io",
	"pypi.org",
	"files.pythonhosted.org",
	"claude.ai",
	"platform.claude.com",
	"api.anthropic.com",
	"statsig.anthropic.com",
	"sentry.io",
	"registry-1.docker.io",
	"auth.docker.io",
	"production.cloudflare.docker.com",
	"ghcr.io",
	"pkg-containers.githubusercontent.com",
}

func GenerateCaddyfile(projectDir string, extraDomains []string) error {
	proxyDir := filepath.Join(projectDir, ".devcontainer", "proxy")
	if err := os.MkdirAll(proxyDir, 0o755); err != nil {
		return fmt.Errorf("failed to create proxy directory: %w", err)
	}

	domains := append([]string{}, defaultDomains...)
	domains = append(domains, extraDomains...)

	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString("\torder forward_proxy before respond\n")
	b.WriteString("}\n\n")
	b.WriteString(":3128 {\n")
	b.WriteString("\tforward_proxy {\n")
	b.WriteString("\t\tacl {\n")
	for _, d := range domains {
		fmt.Fprintf(&b, "\t\t\tallow %s\n", d)
	}
	b.WriteString("\t\t\tdeny all\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")

	path := filepath.Join(proxyDir, "Caddyfile")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func GenerateProxyDockerfile(projectDir string) error {
	content := `FROM caddy:builder AS builder
RUN xcaddy build --with github.com/caddyserver/forwardproxy=github.com/caddyserver/forwardproxy@caddy2

FROM caddy:latest
COPY --from=builder /usr/bin/caddy /usr/bin/caddy

RUN apk add --no-cache dnsmasq

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 3128 53/udp

ENTRYPOINT ["/entrypoint.sh"]
`
	path := filepath.Join(projectDir, ".devcontainer", "proxy", "Dockerfile")
	return os.WriteFile(path, []byte(content), 0o644)
}

func GenerateProxyEntrypoint(projectDir string) error {
	content := `#!/bin/sh
dnsmasq --no-daemon --server=8.8.8.8 --server=8.8.4.4 --listen-address=0.0.0.0 --bind-interfaces &
exec caddy run --config /etc/caddy/Caddyfile
`
	path := filepath.Join(projectDir, ".devcontainer", "proxy", "entrypoint.sh")
	return os.WriteFile(path, []byte(content), 0o755)
}

func GenerateDevDockerfile(projectDir string, dind bool) error {
	devDir := filepath.Join(projectDir, ".devcontainer", "dev")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		return fmt.Errorf("failed to create dev directory: %w", err)
	}

	var content string
	if dind {
		content = `FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    git \
    curl \
    ca-certificates \
    sudo \
    iptables \
    fuse-overlayfs \
    && rm -rf /var/lib/apt/lists/*

# Install Docker
RUN curl -fsSL https://get.docker.com | sh

# Create ccdc user with docker group access
RUN useradd -m -s /bin/bash ccdc && \
    usermod -aG docker ccdc

# Install Claude Code
RUN su - ccdc -c 'curl -fsSL https://claude.ai/install.sh | bash'

# Create ccdc wrapper command
RUN CLAUDE_BIN="/home/ccdc/.local/bin/claude" && \
    printf '#!/bin/sh\nexec %s --dangerously-skip-permissions "$@"\n' "$CLAUDE_BIN" > /usr/local/bin/ccdc && \
    chmod +x /usr/local/bin/ccdc

# Copy /etc/claude/ to ~/.claude/ on bash login
RUN echo 'mkdir -p ~/.claude && for item in /etc/claude/*; do [ -e "$item" ] && cp -r "$item" ~/.claude/$(basename "$item"); done' >> /home/ccdc/.bashrc

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

WORKDIR /workspace

ENTRYPOINT ["/entrypoint.sh"]
`
	} else {
		content = `FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    git \
    curl \
    ca-certificates \
    sudo \
    && rm -rf /var/lib/apt/lists/*

# Create ccdc user
RUN useradd -m -s /bin/bash ccdc

# Install Claude Code
RUN su - ccdc -c 'curl -fsSL https://claude.ai/install.sh | bash'

# Create ccdc wrapper command
RUN CLAUDE_BIN="/home/ccdc/.local/bin/claude" && \
    printf '#!/bin/sh\nexec %s --dangerously-skip-permissions "$@"\n' "$CLAUDE_BIN" > /usr/local/bin/ccdc && \
    chmod +x /usr/local/bin/ccdc

# Copy /etc/claude/ to ~/.claude/ on bash login
RUN echo 'mkdir -p ~/.claude && for item in /etc/claude/*; do [ -e "$item" ] && cp -r "$item" ~/.claude/$(basename "$item"); done' >> /home/ccdc/.bashrc

WORKDIR /workspace
`
	}
	return os.WriteFile(filepath.Join(devDir, "Dockerfile"), []byte(content), 0o644)
}

func GenerateDevEntrypoint(projectDir string) error {
	devDir := filepath.Join(projectDir, ".devcontainer", "dev")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		return fmt.Errorf("failed to create dev directory: %w", err)
	}

	content := `#!/bin/sh
# Start Docker daemon in background
dockerd --storage-driver=fuse-overlayfs &

# Wait for Docker daemon to be ready
while ! docker info >/dev/null 2>&1; do
    sleep 1
done

# Enable multi-platform support (amd64 on arm64 etc.)
docker run --rm --privileged multiarch/qemu-user-static --reset -p yes 2>/dev/null || true

# Execute CMD as ccdc user
exec su - ccdc -c "$*"
`
	return os.WriteFile(filepath.Join(devDir, "entrypoint.sh"), []byte(content), 0o755)
}

func GenerateCompose(projectDir string, dind bool) error {
	var b strings.Builder

	b.WriteString(`services:
  proxy:
    build:
      context: proxy
      dockerfile: Dockerfile
    volumes:
      - ./proxy/Caddyfile:/etc/caddy/Caddyfile:ro
    networks:
      restricted:
        ipv4_address: 172.28.0.10
      external:
    healthcheck:
      test: ["CMD", "caddy", "validate", "--config", "/etc/caddy/Caddyfile"]
      interval: 10s
      timeout: 3s
      retries: 5

  dev:
    build:
      context: dev
      dockerfile: Dockerfile
`)

	b.WriteString("    command: sleep infinity\n")
	if !dind {
		b.WriteString("    user: ccdc\n")
	}

	if dind {
		b.WriteString("    privileged: true\n")
		b.WriteString("    platform: linux/amd64\n")
	}

	b.WriteString(`    dns:
      - 172.28.0.10
    volumes:
      - ..:/workspace
      - ~/.claude/skills:/etc/claude/skills:ro
      - ~/.claude/agents:/etc/claude/agents:ro
      - ~/.claude/commands:/etc/claude/commands:ro
      - ~/.claude/CLAUDE.md:/etc/claude/CLAUDE.md:ro
    working_dir: /workspace
    environment:
      - GITHUB_TOKEN=${GITHUB_TOKEN}
      - GH_TOKEN=${GITHUB_TOKEN}
      - NODE_AUTH_TOKEN=${GITHUB_TOKEN}
      - http_proxy=http://proxy:3128
      - https_proxy=http://proxy:3128
      - HTTP_PROXY=http://proxy:3128
      - HTTPS_PROXY=http://proxy:3128
      - no_proxy=localhost,127.0.0.1
    depends_on:
      proxy:
        condition: service_healthy
    networks:
      - restricted

networks:
  restricted:
    driver: bridge
    internal: true
    ipam:
      config:
        - subnet: 172.28.0.0/24
  external:
    driver: bridge
`)

	path := filepath.Join(projectDir, ".devcontainer", "docker-compose.proxy.yml")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
