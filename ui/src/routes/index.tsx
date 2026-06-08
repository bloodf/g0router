import { createFileRoute, redirect } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api/client";
import type { AuthStatus } from "@/lib/types";

export const Route = createFileRoute("/")({
  beforeLoad: async () => {
    try {
      const s = await apiFetch<AuthStatus>("/api/auth/status", { silent: true });
      if (!s.has_users) throw redirect({ to: "/setup" });
      if (s.require_login && !s.authenticated) throw redirect({ to: "/login" });
      throw redirect({ to: "/dashboard" });
    } catch (e) {
      if ((e as any)?.options?.to) throw e; // propagate redirect
      throw redirect({ to: "/login" });
    }
  },
  component: () => null,
});
