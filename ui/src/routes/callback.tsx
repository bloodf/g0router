import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { relayOAuthCallback } from "@/lib/auth";

export const Route = createFileRoute("/callback")({
  component: CallbackPage,
});

type CallbackStatus = "processing" | "success" | "done" | "error" | "manual";

function CallbackPage() {
  const [status, setStatus] = React.useState<CallbackStatus>("processing");

  React.useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const code = params.get("code") ?? undefined;
    const state = params.get("state") ?? undefined;
    const error = params.get("error") ?? undefined;
    const errorDescription = params.get("error_description") ?? undefined;

    // No code and no error → nothing to relay; show the manual-copy fallback
    // (ports 9router callback/page.js manual state).
    if (!code && !error) {
      setStatus("manual");
      return;
    }

    relayOAuthCallback({
      code,
      state,
      error,
      error_description: errorDescription,
    });

    setStatus(error ? "error" : "success");

    // Auto-close the popup once the opener has been notified.
    const timer = setTimeout(() => {
      setStatus("done");
      try {
        window.close();
      } catch {
        // Some browsers refuse window.close() on non-script-opened tabs.
      }
    }, 1500);
    return () => clearTimeout(timer);
  }, []);

  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center gap-4 p-6 text-center">
      {status === "processing" || status === "success" || status === "done" ? (
        <>
          <span
            className="material-symbols-outlined animate-spin text-4xl text-primary"
            aria-hidden="true"
          >
            progress_activity
          </span>
          <p className="text-sm text-muted-foreground">
            {status === "success" || status === "done"
              ? "Authorization complete. You can close this window."
              : "Completing authorization..."}
          </p>
        </>
      ) : null}

      {status === "error" ? (
        <>
          <span
            className="material-symbols-outlined text-4xl text-destructive"
            aria-hidden="true"
          >
            error
          </span>
          <p className="text-sm text-muted-foreground">
            Authorization failed. You can close this window.
          </p>
        </>
      ) : null}

      {status === "manual" ? (
        <>
          <span
            className="material-symbols-outlined text-4xl text-muted-foreground"
            aria-hidden="true"
          >
            content_copy
          </span>
          <p className="text-sm font-medium text-foreground">Copy This URL</p>
          <p className="max-w-md break-all text-xs text-muted-foreground">
            No authorization code was found. Copy this page's URL and paste it
            back into the application that started the sign-in.
          </p>
          <code
            data-testid="callback-url"
            className="max-w-md break-all rounded-md bg-muted px-3 py-2 text-xs text-foreground"
          >
            {typeof window !== "undefined" ? window.location.href : ""}
          </code>
        </>
      ) : null}
    </div>
  );
}
