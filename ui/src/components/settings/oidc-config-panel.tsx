import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useNotificationStore } from "@/stores/notification";

export interface OidcConfigPanelProps {
  // initialSettings seeds the panel for SSR/unit tests; when omitted the panel
  // fetches from /api/settings on mount.
  initialSettings?: Record<string, unknown>;
}

// OidcConfigPanel (PAR-UI-099) persists the oidc_* keys via the REAL flat
// PUT /api/settings map and tests via the REAL POST /api/auth/oidc/test
// (not exercised against a live IdP under e2e, plan §1.2/§8 ESC-4). The client
// secret is never echoed back into a value attribute on reload.
export function OidcConfigPanel({ initialSettings }: OidcConfigPanelProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [issuerUrl, setIssuerUrl] = React.useState(
    String(initialSettings?.oidc_issuer_url ?? "")
  );
  const [clientId, setClientId] = React.useState(
    String(initialSettings?.oidc_client_id ?? "")
  );
  const [clientSecret, setClientSecret] = React.useState("");
  const [redirectUri, setRedirectUri] = React.useState(
    String(initialSettings?.oidc_redirect_uri ?? "")
  );
  const [scopes, setScopes] = React.useState(
    String(initialSettings?.oidc_scopes ?? "")
  );
  const [saving, setSaving] = React.useState(false);
  const [testing, setTesting] = React.useState(false);

  React.useEffect(() => {
    if (initialSettings !== undefined) return;
    apiFetch<Record<string, unknown>>("/api/settings")
      .then((settings) => {
        setIssuerUrl(String(settings?.oidc_issuer_url ?? ""));
        setClientId(String(settings?.oidc_client_id ?? ""));
        setRedirectUri(String(settings?.oidc_redirect_uri ?? ""));
        setScopes(String(settings?.oidc_scopes ?? ""));
      })
      .catch(() => {
        /* tolerate absent OIDC keys */
      });
  }, [initialSettings]);

  async function save() {
    setSaving(true);
    try {
      const body: Record<string, string> = {
        oidc_issuer_url: issuerUrl,
        oidc_client_id: clientId,
        oidc_redirect_uri: redirectUri,
        oidc_scopes: scopes,
      };
      // Only persist the secret when the operator typed a new one.
      if (clientSecret) body.oidc_client_secret = clientSecret;
      await apiFetch("/api/settings", { method: "PUT", body: JSON.stringify(body) });
      pushToast({ message: "OIDC settings saved" });
    } catch {
      pushToast({ message: "Failed to save OIDC settings" });
    } finally {
      setSaving(false);
    }
  }

  async function test() {
    setTesting(true);
    try {
      await apiFetch("/api/auth/oidc/test", {
        method: "POST",
        body: JSON.stringify({
          issuer_url: issuerUrl,
          client_id: clientId,
          client_secret: clientSecret,
          redirect_uri: redirectUri,
          scopes,
        }),
      });
      pushToast({ message: "OIDC connection OK" });
    } catch {
      pushToast({ message: "OIDC test failed" });
    } finally {
      setTesting(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>OIDC / SSO</CardTitle>
      </CardHeader>
      <CardContent className="mt-4 flex flex-col gap-4">
        <Input
          data-testid="oidc-issuer-url"
          label="Issuer URL"
          value={issuerUrl}
          onChange={(event) => setIssuerUrl(event.target.value)}
          placeholder="https://idp.example.com"
        />
        <Input
          data-testid="oidc-client-id"
          label="Client ID"
          value={clientId}
          onChange={(event) => setClientId(event.target.value)}
        />
        <Input
          data-testid="oidc-client-secret"
          label="Client Secret"
          type="password"
          value={clientSecret}
          onChange={(event) => setClientSecret(event.target.value)}
          placeholder="••••••••"
        />
        <Input
          data-testid="oidc-redirect-uri"
          label="Redirect URI"
          value={redirectUri}
          onChange={(event) => setRedirectUri(event.target.value)}
        />
        <Input
          data-testid="oidc-scopes"
          label="Scopes"
          value={scopes}
          onChange={(event) => setScopes(event.target.value)}
          placeholder="openid email profile"
        />
        <div className="flex justify-end gap-2">
          <Button data-testid="oidc-test" variant="outline" loading={testing} onClick={test}>
            Test
          </Button>
          <Button data-testid="oidc-save" variant="primary" loading={saving} onClick={save}>
            Save
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
