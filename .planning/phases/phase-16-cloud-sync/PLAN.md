# Phase 16: Cloud Sync

**Phase:** 16  
**Goal:** Implement configuration export/import and cloud sync UI.  
**Requirements:** MGMT-13, UI-16, PLAT-14  
**Estimated duration:** 3–4 days  
**Wave:** 5 — 9router Features

---

## Why

Cloud sync lets users back up configuration and restore it across machines.

---

## Scope

### In scope
- `internal/admin/sync.go` — `/api/sync/export`, `/api/sync/import`.
- Encrypted configuration bundle format.
- Optional upstream cloud sync orchestration.
- Dashboard page:
  - `routes/_app.sync.tsx`

### Out of scope
- Managed cloud service backend (we orchestrate upload/download only).

---

## Verification

### Tests
1. Export returns a valid encrypted bundle.
2. Import restores configuration exactly.
3. Secrets in the bundle are encrypted.
4. Import validates bundle signature/version before applying.

### Manual verification
1. Export config, wipe database, import config, verify providers are restored.

---

## Tasks

1. Define sync bundle schema.
2. Implement export with encryption.
3. Implement import with validation.
4. Implement optional cloud sync HTTP client.
5. Implement dashboard page.
6. Write tests and E2E coverage.
7. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Bundle compatibility breaks | Version the bundle schema and support migration readers. |
| Secret leakage in export | Encrypt all sensitive fields; never include plaintext API keys. |
