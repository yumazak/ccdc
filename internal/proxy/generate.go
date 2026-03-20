package proxy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var defaultDomains = []string{
	".github.com",
	".api.github.com",
	".raw.githubusercontent.com",
	".registry.npmjs.org",
	".npm.pkg.github.com",
	".rubygems.org",
	".bundler.io",
	".pypi.org",
	".files.pythonhosted.org",
	".claude.ai",
	".platform.claude.com",
	".api.anthropic.com",
	".statsig.anthropic.com",
	".sentry.io",
}

func GenerateSquidConf(projectDir string, extraDomains []string) error {
	proxyDir := filepath.Join(projectDir, ".devcontainer", "proxy")
	if err := os.MkdirAll(proxyDir, 0o755); err != nil {
		return fmt.Errorf("failed to create proxy directory: %w", err)
	}

	domains := append([]string{}, defaultDomains...)
	for _, d := range extraDomains {
		if !strings.HasPrefix(d, ".") {
			d = "." + d
		}
		domains = append(domains, d)
	}

	var b strings.Builder
	b.WriteString("# Allowlist\n")
	b.WriteString("acl allowlist dstdomain")
	for _, d := range domains {
		fmt.Fprintf(&b, " %s", d)
	}
	b.WriteString("\n\n")
	b.WriteString("# Allow HTTPS (CONNECT) to allowlisted domains\n")
	b.WriteString("http_access allow CONNECT allowlist\n\n")
	b.WriteString("# Allow HTTP to allowlisted domains\n")
	b.WriteString("http_access allow allowlist\n\n")
	b.WriteString("# Deny everything else\n")
	b.WriteString("http_access deny all\n\n")
	b.WriteString("http_port 3128\n")

	return os.WriteFile(filepath.Join(proxyDir, "squid.conf"), []byte(b.String()), 0o644)
}

func GenerateProxyDockerfile(projectDir string) error {
	proxyDir := filepath.Join(projectDir, ".devcontainer", "proxy")
	if err := os.MkdirAll(proxyDir, 0o755); err != nil {
		return fmt.Errorf("failed to create proxy directory: %w", err)
	}

	content := `FROM ubuntu/squid:latest

RUN apt-get update && apt-get install -y dnsmasq && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 3128 53/udp

ENTRYPOINT ["/entrypoint.sh"]
`
	return os.WriteFile(filepath.Join(proxyDir, "Dockerfile"), []byte(content), 0o644)
}

func GenerateProxyEntrypoint(projectDir string) error {
	content := `#!/bin/sh
dnsmasq --no-daemon --server=8.8.8.8 --server=8.8.4.4 --listen-address=0.0.0.0 --bind-interfaces &
exec squid -N -f /etc/squid/squid.conf
`
	path := filepath.Join(projectDir, ".devcontainer", "proxy", "entrypoint.sh")
	return os.WriteFile(path, []byte(content), 0o755)
}

func GenerateDevDockerfile(projectDir string, docker bool) error {
	devDir := filepath.Join(projectDir, ".devcontainer", "dev")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		return fmt.Errorf("failed to create dev directory: %w", err)
	}

	var b strings.Builder
	b.WriteString(`FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    git \
    curl \
    ca-certificates \
    sudo \
    && rm -rf /var/lib/apt/lists/*
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

# Create ccdc wrapper command
RUN CLAUDE_BIN="/home/ccdc/.local/bin/claude" && \
    printf '#!/bin/sh\nexec %s --dangerously-skip-permissions "$@"\n' "$CLAUDE_BIN" > /usr/local/bin/ccdc && \
    chmod +x /usr/local/bin/ccdc

# Copy /etc/claude/ to ~/.claude/ on bash login
RUN echo 'mkdir -p ~/.claude && for item in /etc/claude/*; do [ -e "$item" ] && cp -r "$item" ~/.claude/$(basename "$item"); done' >> /home/ccdc/.bashrc

WORKDIR /workspace
`)

	return os.WriteFile(filepath.Join(devDir, "Dockerfile"), []byte(b.String()), 0o644)
}

func GenerateCompose(projectDir string, docker bool) error {
	var b strings.Builder

	b.WriteString(`services:
  proxy:
    build:
      context: proxy
      dockerfile: Dockerfile
    volumes:
      - ./proxy/squid.conf:/etc/squid/squid.conf:ro
    networks:
      restricted:
        ipv4_address: 172.28.0.10
      external:
    healthcheck:
      test: ["CMD", "squid", "-k", "check", "-f", "/etc/squid/squid.conf"]
      interval: 10s
      timeout: 3s
      retries: 5
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

	b.WriteString(`
  dev:
    build:
      context: dev
      dockerfile: Dockerfile
    command: sleep infinity
    user: ccdc
    dns:
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
      - no_proxy=localhost,127.0.0.1,socket-proxy
`)

	if docker {
		b.WriteString("      - DOCKER_HOST=tcp://socket-proxy:2375\n")
	}

	b.WriteString(`    depends_on:
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
