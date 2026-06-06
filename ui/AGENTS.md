# g0router - UI

React 19 dashboard embedded into the Go binary at build time (`ui/dist/` committed).

## Stack
- React 19 + TypeScript + Vite 8, Tailwind v4, Vitest (unit), Playwright (`ui/e2e/`)
- Dev: `npm run dev` (port 5173, proxies `/api` to the Go server on 8080)
- Build: `npm run build` → `ui/dist/` (commit the dist output)

## Key Conventions
- API client lives in `ui/src/api.ts` — fully typed; all responses use the `{data, error}` envelope with snake_case fields.
- Every page component has a colocated `.test.tsx`; UI test suite must stay green (`npm test`).

Fill this in as the track is built out.
