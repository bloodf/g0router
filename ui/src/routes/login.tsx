import * as React from "react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { LoginForm } from "@/components/auth/login-form";
import {
  getAuthStatus,
  loginWithPassword,
  startOidc,
  LoginError,
  type AuthMode,
} from "@/lib/auth";
import { useUserStore } from "@/stores/user";
import { useNotificationStore } from "@/stores/notification";

export const Route = createFileRoute("/login")({
  component: LoginPage,
});

function LoginPage() {
  const navigate = useNavigate();
  const setUser = useUserStore((state) => state.setUser);
  const setToken = useUserStore((state) => state.setToken);
  const pushToast = useNotificationStore((state) => state.push);

  const [authMode, setAuthMode] = React.useState<AuthMode>("password");
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | undefined>(undefined);
  const [retryAfter, setRetryAfter] = React.useState(0);

  // GET /api/auth/status on mount drives which auth methods render (PAR-UI-067).
  React.useEffect(() => {
    let active = true;
    getAuthStatus().then((status) => {
      if (active) setAuthMode(status.auth_mode);
    });
    return () => {
      active = false;
    };
  }, []);

  // Rate-limit countdown (PAR-UI-065): tick down each second, re-enable at 0.
  React.useEffect(() => {
    if (retryAfter <= 0) return;
    const timer = setInterval(() => {
      setRetryAfter((current) => (current <= 1 ? 0 : current - 1));
    }, 1000);
    return () => clearInterval(timer);
  }, [retryAfter]);

  async function handleSubmit(username: string, password: string) {
    setLoading(true);
    setError(undefined);
    try {
      const { token, user } = await loginWithPassword(username, password);
      setToken(token);
      setUser(user);
      navigate({ to: "/dashboard" });
    } catch (err) {
      if (err instanceof LoginError) {
        if (err.status === 429 && err.retryAfter && err.retryAfter > 0) {
          setRetryAfter(err.retryAfter);
        }
        setError(err.message);
        pushToast({ message: err.message });
      } else {
        const message = "login failed";
        setError(message);
        pushToast({ message });
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-[60vh] items-center justify-center p-4">
      <LoginForm
        authMode={authMode}
        loading={loading}
        error={error}
        retryAfter={retryAfter}
        onSubmit={handleSubmit}
        onOidc={startOidc}
      />
    </div>
  );
}
