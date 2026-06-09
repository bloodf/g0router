# Phase 1: Scaffolding

**Phase:** 01  
**Goal:** Delete old code and set up the new directory structure so CI passes an empty build.  
**Requirements:** TEST-07  
**Estimated duration:** 1–2 days  
**Wave:** 1 — Foundation

---

## Why

The current codebase has drifted too far from the target architecture. This phase creates a clean foundation for the 9router+BiFrost port by removing old feature code and establishing the new package layout.

---

## Scope

### In scope
- Delete `api/`, `internal/`, and `ui/src/` old feature code.
- Preserve `cmd/g0router/`, `embed.go`, `go.mod`, `Dockerfile`, `deploy/`, `ui/package.json` + build toolchain, `ui/public/providers/`, project metadata.
- Create new `internal/` package layout:
  - `internal/schemas/`
  - `internal/server/`
  - `internal/api/`
  - `internal/admin/`
  - `internal/providers/`
  - `internal/inference/`
  - `internal/catalog/`
  - `internal/governance/`
  - `internal/auth/`
  - `internal/store/`
  - `internal/logging/`
  - `internal/mcp/`
  - `internal/config/`
  - `internal/platform/`
- Create placeholder `_test.go` files in each new package.
- Clean `go.mod` dependencies.
- Update `embed.go` to reference the new UI build path.
- Ensure `go test ./...`, `go vet ./...`, and `npm run build` pass.

### Out of scope
- Implementing any provider or API handler.
- Adding dashboard routes or components.
- Changing the UI build toolchain.

---

## Verification

### Tests
- Placeholder tests compile and pass.

### Manual verification
1. `go test ./...` passes.
2. `go vet ./...` passes.
3. `cd ui && npm run build` passes.
4. Binary builds with `go build ./cmd/g0router`.

---

## Tasks

1. Remove old directories (`api/`, `internal/`, `ui/src/`).
2. Create new directory structure.
3. Write minimal `package.go` or `doc.go` in each package.
4. Write placeholder `_test.go` files.
5. Update `embed.go` and `cmd/g0router/main.go` skeleton.
6. Run `go mod tidy`.
7. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Accidentally delete a file we need | Review delete list against design doc "retained skeleton" before executing. |
| `go mod tidy` removes needed indirect deps | Pin required indirect deps explicitly. |
| UI build breaks from deleted source | Create a minimal `ui/src/main.tsx` + `App.tsx` placeholder. |
