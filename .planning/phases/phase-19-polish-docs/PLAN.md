# Phase 19: Polish + Docs

**Phase:** 19  
**Goal:** Final QA, documentation updates, and deployment verification.  
**Requirements:** REL-02..06, UI-18  
**Estimated duration:** 3–4 days  
**Wave:** 6 — Hardening + Ship

---

## Why

A complete rewrite needs clear docs and verified deployment paths before it can be used.

---

## Scope

### In scope
- Final pass on all gates: `go test ./...`, `go vet ./...`, `npm run build`, `npx playwright test`.
- Update `ARCHITECTURE.md`, `CONFIG.md`, `DEPLOYMENT.md`.
- Update `AGENTS.md` with new conventions if needed.
- Verify Docker build and systemd service.
- Apply g0router branding consistently across the dashboard.
- Update `docs/WORKFLOW.md`.

### Out of scope
- New features not in the 19-phase plan.

---

## Verification

### Tests
1. All automated gates pass.
2. Docker image builds and runs.
3. systemd service file is valid.

### Manual verification
1. Deploy locally with Docker and run OpenAI SDK test.
2. Spot-check dashboard branding.

---

## Tasks

1. Run final gates and fix any failures.
2. Update documentation.
3. Verify Docker build.
4. Verify systemd service.
5. Apply final branding pass.
6. Commit and tag milestone completion.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Last-minute gate failures | Reserve buffer time; fix issues iteratively. |
| Docs become stale quickly | Commit to updating docs at each phase transition. |
