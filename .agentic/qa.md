# QA Config

## Dev server
command: npm run dev --prefix ui
port: 5173

## URLs
local: http://localhost:5173
staging: <!-- optional: add staging/preview URL here -->

## Preferences
prefer: local

## Notes
- Backend dev server: `go run ./cmd/g0router` (default port 8080). The Vite dev
  server proxies `/api` to it — start both for full-stack QA.
- Embedded production UI is served by the Go binary itself; `make e2e-binary`
  exercises that path.
- Playwright E2E suite lives at `ui/e2e/` and targets the Vite dev server.

## Viewport canonical sizes
# mobile:  375x667
# tablet:  768x1024
# desktop: 1440x900
# Override per-scenario by setting `viewport: [mobile, tablet, desktop]` in a scenario block.
# Scenarios default to `[desktop]` when no viewport is declared.

## Visual baselines
# Perceptual diff baselines are stored at:
#   tests/visual-baselines/<scenario-id>/<viewport>.png
# First run with perceptual_diff_enabled: true saves baselines and returns INCONCLUSIVE.
# Subsequent runs compare against saved baselines using Playwright toHaveScreenshot.
# Commit baselines to source control after reviewing them.

## Accessibility
# qa-engineer runs axe-core at WCAG AA level by default for `accessibility` method scenarios.
# Default axe tags: [wcag2a, wcag2aa]
# To enforce AAA: set wcag_level: AAA in the scenario (adds wcag2aaa to axe tags).
# To override axe tags directly: set axe_tags: [wcag2a, wcag2aa] in the scenario.
