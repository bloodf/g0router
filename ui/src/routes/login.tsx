import { createFileRoute, useNavigate, redirect } from "@tanstack/react-router";
import { useState } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Icon } from "@/components/common/Icon";
import { useLogin } from "@/lib/auth";
import { apiFetch } from "@/lib/api/client";
import type { AuthStatus } from "@/lib/types";
import { toast } from "sonner";

export const Route = createFileRoute("/login")({
  beforeLoad: async () => {
    const s = await apiFetch<AuthStatus>("/api/auth/status", { silent: true });
    if (!s.has_users) throw redirect({ to: "/setup" });
    if (s.authenticated) throw redirect({ to: "/dashboard" });
  },
  component: LoginPage,
});

function LoginPage() {
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("123456");
  const [show, setShow] = useState(false);
  const login = useLogin();
  const navigate = useNavigate();

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await login.mutateAsync({ username, password });
      toast.success("Welcome back");
      navigate({ to: "/dashboard" });
    } catch (err: any) {
      console.error("[LOGIN] mutation failed:", err?.message || err);
      toast.error(err?.message || "Invalid credentials");
    }
  };

  return (
    <div className="min-h-screen bg-hero-gradient dot-grid-bg flex items-center justify-center px-4">
      <div className="w-full max-w-md">
        <div className="flex items-center justify-center mb-6">
          <div className="w-12 h-12 rounded-2xl bg-gradient-to-br from-brand-400 to-brand-600 flex items-center justify-center text-white font-bold text-lg shadow-warm">
            g0
          </div>
        </div>
        <Card className="card-elev border-border p-8">
          <h1 className="text-2xl font-semibold text-center">Welcome back</h1>
          <p className="text-sm text-text-muted text-center mt-1">
            Sign in to your g0router dashboard
          </p>

          <form onSubmit={submit} className="mt-6 space-y-4">
            <div>
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                autoFocus
              />
            </div>
            <div>
              <Label htmlFor="password">Password</Label>
              <div className="relative">
                <Input
                  id="password"
                  type={show ? "text" : "password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
                <button
                  type="button"
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-text-muted hover:text-foreground"
                  onClick={() => setShow((s) => !s)}
                  aria-label={show ? "Hide password" : "Show password"}
                >
                  <Icon name={show ? "visibility_off" : "visibility"} size={18} />
                </button>
              </div>
            </div>
            <Button type="submit" className="w-full btn-cta" disabled={login.isPending}>
              {login.isPending ? "Signing in…" : "Sign in"}
            </Button>
          </form>

          <div className="mt-6 p-3 bg-info/5 border border-info/20 rounded-lg text-xs text-text-muted">
            <div className="flex items-center gap-1.5 text-info font-medium mb-1">
              <Icon name="info" size={14} /> Demo credentials
            </div>
            Username <code className="font-mono">admin</code> / password{" "}
            <code className="font-mono">123456</code>. Change this password in Settings.
          </div>
        </Card>
        <p className="text-center text-xs text-text-muted mt-4">
          g0router · single-binary LLM gateway
        </p>
      </div>
    </div>
  );
}
