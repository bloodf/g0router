# Deployment Guide

## Quick Start (Development)

```bash
make build
./g0router serve --port 20128 --data-dir ~/.g0router
```

## systemd Service (Production Linux)

### Automatic Install

```bash
# System-wide (requires root)
sudo ./g0router install

# Creates:
#   /usr/local/bin/g0router               (binary)
#   /etc/systemd/system/g0router.service  (unit)
#   /etc/default/g0router                 (env config)
#   /var/lib/g0router/                    (data dir)
#   g0router system user                  (runs as non-root)

# User-level (no root required)
./g0router install --user
# Installs to ~/.config/systemd/user/g0router.service
# Data at ~/.g0router/
```

### Manual Install

```bash
# 1. Copy binary
sudo cp g0router /usr/local/bin/g0router

# 2. Create system user
sudo useradd --system --no-create-home --shell /usr/sbin/nologin g0router

# 3. Create data directory
sudo mkdir -p /var/lib/g0router
sudo chown g0router:g0router /var/lib/g0router

# 4. Install unit + env file
sudo cp deploy/g0router.service /etc/systemd/system/
sudo cp deploy/g0router.default /etc/default/g0router

# 5. Enable and start
sudo systemctl daemon-reload
sudo systemctl enable --now g0router

# 6. Verify
sudo systemctl status g0router
sudo journalctl -u g0router -f
```

### Uninstall

```bash
sudo g0router uninstall
# Stops service, removes unit + binary. Data in /var/lib/g0router/ preserved.

g0router uninstall --user
# Stops user service and preserves ~/.g0router/.
```

### systemd Unit (deploy/g0router.service)

```ini
[Unit]
Description=g0router LLM Gateway
Documentation=https://github.com/bloodf/g0router
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=g0router
Group=g0router
EnvironmentFile=-/etc/default/g0router
ExecStart=/usr/local/bin/g0router serve
Restart=on-failure
RestartSec=5s
StartLimitIntervalSec=60s
StartLimitBurst=5

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/g0router
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true
MemoryDenyWriteExecute=true
LockPersonality=true

StandardOutput=journal
StandardError=journal
SyslogIdentifier=g0router

[Install]
WantedBy=multi-user.target
```

### Environment (deploy/g0router.default)

```bash
# /etc/default/g0router
PORT=20128
DATA_DIR=/var/lib/g0router
JWT_SECRET=                         # Generate: openssl rand -hex 32
API_KEY_SECRET=                     # Generate: openssl rand -hex 32
REQUIRE_API_KEY=true
ENABLE_REQUEST_LOGS=false
RTK_ENABLED=true
CAVEMAN_ENABLED=false
CAVEMAN_LEVEL=full
# HTTPS_PROXY=http://proxy:8080
```

---

## Docker

Docker runs as a non-root user and persists SQLite data in `/data`. The image
seeds `/data` with non-root ownership so a new named volume is writable on first
boot.

Generate the API-key secret before starting Docker and keep it stable across
restarts. `JWT_SECRET` is accepted by the config loader for compatibility but
the current dashboard/control-plane auth path uses gateway API keys, not JWT
sessions.

```bash
export API_KEY_SECRET="$(openssl rand -hex 32)"
```

`API_KEY_SECRET` hashes gateway API keys. Those API keys gate `/v1/*` inference
and `/api/*` dashboard/control-plane routes when `REQUIRE_API_KEY=true`.

### Build + Run

```bash
# Build
docker build -t g0router .

# Run
docker run -d \
  --name g0router \
  --restart unless-stopped \
  -p 127.0.0.1:20128:20128 \
  -v g0router-data:/data \
  -e API_KEY_SECRET="${API_KEY_SECRET}" \
  -e BIND_ADDRESS=0.0.0.0 \
  -e REQUIRE_API_KEY=true \
  g0router
```

Create the first gateway/control-plane API key with the same `API_KEY_SECRET`
value used by the running container:

```bash
docker run --rm \
  -v g0router-data:/data \
  -e DATA_DIR=/data \
  -e API_KEY_SECRET="${API_KEY_SECRET}" \
  g0router keys add default
```

Use the raw `g0r_...` key printed by that command as `Authorization: Bearer ...`
for `/v1/*` clients and as the dashboard control-plane key.

### Docker Compose (docker-compose.yml)

```yaml
services:
  g0router:
    build: .
    image: g0router:latest
    container_name: g0router
    restart: unless-stopped
    ports:
      - "127.0.0.1:20128:20128"
    volumes:
      - g0router-data:/data
    environment:
      PORT: "20128"
      BIND_ADDRESS: "0.0.0.0"
      DATA_DIR: "/data"
      JWT_SECRET: "${JWT_SECRET:-}"
      API_KEY_SECRET: "${API_KEY_SECRET:?API_KEY_SECRET is required for docker-compose API keys}"
      REQUIRE_API_KEY: "true"
      RTK_ENABLED: "true"
    healthcheck:
      test: ["CMD", "/g0router", "healthcheck"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s

volumes:
  g0router-data:
```

Start compose with the API-key secret exported:

```bash
API_KEY_SECRET="${API_KEY_SECRET}" docker compose up -d
```

### Dockerfile

```dockerfile
FROM node:22-alpine AS ui-builder
WORKDIR /app/ui
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ ./
RUN npm run build

FROM golang:1.26-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /app/ui/dist ./ui/dist
RUN mkdir -p /docker-data && touch /docker-data/.keep
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o g0router ./cmd/g0router

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go-builder /app/g0router /g0router
COPY --from=go-builder --chown=65532:65532 /docker-data/ /data/
VOLUME ["/data"]
EXPOSE 20128
ENV DATA_DIR=/data
ENV PORT=20128
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["/g0router", "healthcheck"]
ENTRYPOINT ["/g0router"]
CMD ["serve"]
```

---

## Logs

```bash
# systemd
journalctl -u g0router -f
journalctl -u g0router --since today
journalctl -u g0router -p err

# Docker
docker logs -f g0router
```

## Health Check

```bash
g0router healthcheck          # CLI (exit 0 = healthy)
curl localhost:20128/healthz  # HTTP
```

## MCP Manual Verification

Run these checks after deployment when MCP gateway support is enabled. They are intentionally local-first; each step should leave secrets out of command output.

### Remote HTTP MCP

```bash
g0router mcp add atlassian-a \
  --server-key atlassian \
  --launch-type http \
  --transport streamable-http \
  --url https://mcp.atlassian.com/mcp \
  --account-label account-a

g0router mcp auth start atlassian-a \
  --authorization-url https://auth.example/authorize \
  --resource https://mcp.atlassian.com \
  --redirect-url http://localhost:20128/api/mcp/oauth/callback

g0router mcp auth complete atlassian-a "http://localhost:20128/api/mcp/oauth/callback?code=...&state=..."
g0router mcp accounts atlassian-a
g0router mcp tools atlassian-a
```

Expected result: account labels and compact tool names are shown, but access tokens, refresh tokens, headers, and secret environment values are not printed.

### npx MCP

```bash
g0router mcp add expo \
  --server-key expo \
  --launch-type npx \
  --transport stdio \
  --command @expo/mcp \
  --arg --stdio
g0router mcp list
```

Expected launch shape: `npx --yes @expo/mcp --stdio`. The launcher builds an argv list directly and does not interpolate through a shell.

### Docker MCP

```bash
docker version
g0router mcp add docker-search \
  --server-key docker-search \
  --launch-type docker \
  --transport stdio \
  --command mcp/search:latest \
  --env TOKEN=secret
g0router mcp list
```

Expected launch shape: `docker run --rm -i -e TOKEN mcp/search:latest`. If `docker version` fails, skip Docker runtime verification and record that Docker or its daemon was unavailable.

## Upgrade

```bash
# systemd
sudo systemctl stop g0router
sudo cp g0router-new /usr/local/bin/g0router
sudo systemctl start g0router

# Docker
docker compose pull && docker compose up -d
```

## Reverse Proxy (nginx)

```nginx
server {
    listen 443 ssl http2;
    server_name llm.example.com;

    ssl_certificate /etc/letsencrypt/live/llm.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/llm.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:20128;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 600s;
    }
}
```
