import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { AppShell } from "@/components/layout/AppShell";
import { apiFetch } from "@/lib/api/client";
import type { AuthStatus } from "@/lib/mocks/types";

export const Route = createFileRoute("/_app")({
  beforeLoad: async () => {
    const s = await apiFetch<AuthStatus>("/api/auth/status", { silent: true });
    if (!s.has_users) throw redirect({ to: "/setup" });
    if (s.require_login && !s.authenticated) throw redirect({ to: "/login" });
  },
  component: () => (
    <AppShell>
      <Outlet />
    </AppShell>
  ),
});
