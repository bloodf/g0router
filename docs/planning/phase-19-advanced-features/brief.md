# Brief

**Problem:** g0router lacks semantic caching, self-update, a streaming WebSocket chat channel, a MITM inspection proxy, locale persistence, and a skills catalog. These are the final advanced features for the gateway control plane and the highest-risk surface in the stage (updater self-replace, MITM CA handling).

**Success criteria:**
- Semantic cache (`internal/semcache`) serves cached chat completions via Go-side cosine (exact-key first, 0.95 threshold), inert when no embedding connection exists, gated by `semantic_cache` flag.
- Auto-updater (`internal/update`) verifies `checksums.txt` and stages a swap at `DATA_DIR/update/g0router.new` only on explicit user action.
- WebSocket chat endpoint streams `delta`/`done`/`error` frames over protocol v1, gated by `websocket_chat` flag; MITM proxy (`internal/mitm`) mints leaf certs from a persisted ECDSA P-256 CA, gated by `mitm_proxy` flag.
- Version, locale (get/set), and skills catalog endpoints return the `{data, error}` envelope with snake_case fields.

**Non-goals:**
- WebRTC (deferred — WebSocket only this stage).
- 33-locale i18n translations (backend stores locale preference only; only `en` + `pt-BR` ship complete).
- Landing page / skills page UI (Lovable's job; backend ships only the catalog endpoint).

**Constraints:** Direct push to main. All JSON snake_case + `{data, error}` envelope. Feature flags `semantic_cache`/`websocket_chat`/`mitm_proxy` seeded `enabled=0`. DDD-lite layering (handlers thin, domain owns logic, store is persistence-only). No automatic `/etc/hosts` editing; `ca.key` mode 0600. Security review mandatory (PROCESS §7). Coverage ≥ 95.0%.

**Verification:** Per-phase gate green (test/vet/build/-race), coverage ≥ 95.0%, mandatory security pass recorded in phase `## Outcome`.

**QA criteria:**
```yaml
qa_skip: null
scenarios:
  - name: semcache exact + cosine hit path returns cached response
    method: api
  - name: updater staged-swap verifies checksum and writes g0router.new
    method: runtime-required
  - name: WebSocket chat protocol v1 streams delta/done/error
    method: runtime-required
  - name: manual_smoke - version/locale/skills/mitm endpoints return envelope shapes
    method: manual_smoke
```

**Linked artifacts:** architect-plan: ./architect-plan.md; orchestration: ./orchestration.jsonl
