# Risk Register — Phase 15

- **Binary supply-chain (Critical):** a tampered/poisoned `cloudflared` download executing on host. Mitigation: pinned version + per-OS/arch SHA-256 verified before chmod+exec; refuse on mismatch; HTTPS official URL only.
- **Tunnel exposes unauthed gateway (Critical):** an active tunnel publicly exposes a misconfigured/unauthenticated gateway. Mitigation: tunnels target the gateway whose `/v1/*` and `/api/*` auth already enforce; tunnel mutations require admin-session/bearer + audit.
- **Health-loop goroutine leaks (Major):** background loops outlive shutdown or spawn duplicates. Mitigation: single startup-owned context (no `init()`), loops + child processes cancelled on shutdown; tested for clean stop.
- **Secret leakage (Major):** tunnel tokens logged or returned in API payloads. Mitigation: `config_enc` encrypted at rest; never log CLI-printed tokens; responses omit decrypted config.
