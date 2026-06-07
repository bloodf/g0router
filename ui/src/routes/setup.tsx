import { createFileRoute, useNavigate, redirect } from "@tanstack/react-router";
import { useState } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useSetup } from "@/lib/auth";
import { apiFetch } from "@/lib/api/client";
import type { AuthStatus } from "@/lib/mocks/types";
import { toast } from "sonner";

export const Route = createFileRoute("/setup")({
  beforeLoad: async () => {
    const s = await apiFetch<AuthStatus>("/api/auth/status", { silent: true });
    if (s.has_users) throw redirect({ to: "/login" });
  },
  component: SetupPage,
});

function SetupPage() {
  const [username, setUsername] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const setup = useSetup();
  const navigate = useNavigate();

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (password !== confirm) {
      toast.error("Passwords don't match");
      return;
    }
    try {
      await setup.mutateAsync({ username, password, display_name: displayName });
      toast.success("Admin account created");
      navigate({ to: "/dashboard" });
    } catch {}
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
          <h1 className="text-2xl font-semibold text-center">First-run setup</h1>
          <p className="text-sm text-text-muted text-center mt-1">
            Create the initial admin account
          </p>
          <form onSubmit={submit} className="mt-6 space-y-4">
            <div>
              <Label>Username</Label>
              <Input value={username} onChange={(e) => setUsername(e.target.value)} required />
            </div>
            <div>
              <Label>Display name (optional)</Label>
              <Input value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
            </div>
            <div>
              <Label>Password</Label>
              <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
            </div>
            <div>
              <Label>Confirm password</Label>
              <Input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} required />
            </div>
            <Button type="submit" className="w-full btn-cta" disabled={setup.isPending}>
              {setup.isPending ? "Creating…" : "Create admin"}
            </Button>
          </form>
        </Card>
      </div>
    </div>
  );
}
