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
StartLimitInterval=60s
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

### Build + Run

```bash
# Build
docker build -t g0router .

# Run
docker run -d \
  --name g0router \
  --restart unless-stopped \
  -p 20128:20128 \
  -v g0router-data:/data \
  -e JWT_SECRET=$(openssl rand -hex 32) \
  -e API_KEY_SECRET=$(openssl rand -hex 32) \
  -e REQUIRE_API_KEY=true \
  g0router
```

### Docker Compose (docker-compose.yml)

```yaml
services:
  g0router:
    build: .
    image: g0router:latest
    container_name: g0router
    restart: unless-stopped
    ports:
      - "20128:20128"
    volumes:
      - g0router-data:/data
    environment:
      PORT: "20128"
      DATA_DIR: "/data"
      JWT_SECRET: "${JWT_SECRET}"
      API_KEY_SECRET: "${API_KEY_SECRET}"
      REQUIRE_API_KEY: "true"
      RTK_ENABLED: "true"
    healthcheck:
      test: ["/g0router", "healthcheck"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s

volumes:
  g0router-data:
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
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o g0router ./cmd/g0router

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go-builder /app/g0router /g0router
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
