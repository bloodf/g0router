import { createRootRoute, Outlet } from '@tanstack/react-router'

// Phase 1 placeholder root route. The TanStackRouterVite plugin will scan
// this file (and the rest of ui/src/routes/) and regenerate routeTree.gen.ts
// during `npm run build`. Real routes land in later phases.
export const Route = createRootRoute({
  component: () => <Outlet />,
})
