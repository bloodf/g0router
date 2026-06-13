import * as React from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { AuthMode } from "@/lib/auth";

export interface LoginFormProps {
  authMode: AuthMode;
  loading?: boolean;
  error?: string;
  /** Seconds remaining on a rate-limit lockout; > 0 disables submit. */
  retryAfter?: number;
  onSubmit: (username: string, password: string) => void;
  onOidc: () => void;
}

export function LoginForm({
  authMode,
  loading = false,
  error,
  retryAfter = 0,
  onSubmit,
  onOidc,
}: LoginFormProps) {
  const [username, setUsername] = React.useState("");
  const [password, setPassword] = React.useState("");

  const showOidc = authMode === "oidc" || authMode === "both";
  // Variant per plan §1.4: when auth_mode is "oidc", OIDC is the primary action
  // (Go still permits password unless OIDC is configured, so the password form
  // stays available as a recovery affordance).
  const oidcPrimary = authMode === "oidc";
  const locked = retryAfter > 0;

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (loading || locked) return;
    onSubmit(username, password);
  }

  return (
    <Card padding="lg" className="w-full max-w-sm">
      <CardHeader className="mb-4">
        <CardTitle className="text-xl">Sign in</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <Input
            id="username"
            name="username"
            label="Username"
            autoComplete="username"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
          />
          <Input
            id="password"
            name="password"
            type="password"
            label="Password"
            autoComplete="current-password"
            error={error}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
          <Button
            type="submit"
            variant={oidcPrimary ? "outline" : "primary"}
            loading={loading}
            disabled={locked}
          >
            {locked ? `Wait ${retryAfter}s` : "Sign in"}
          </Button>
        </form>

        {showOidc ? (
          <div className="mt-4">
            <Button
              type="button"
              data-testid="oidc-button"
              data-primary={oidcPrimary ? "true" : "false"}
              variant={oidcPrimary ? "primary" : "outline"}
              className="w-full"
              onClick={onOidc}
            >
              Sign in with OIDC
            </Button>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}
