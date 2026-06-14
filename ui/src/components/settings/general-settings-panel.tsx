import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { Toggle } from "@/components/ui/toggle";
import { useTheme } from "@/hooks/use-theme";
import type { Theme } from "@/stores/theme";
import { useNotificationStore } from "@/stores/notification";

const THEME_OPTIONS = [
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
  { value: "system", label: "System" },
];

export interface GeneralSettingsPanelProps {
  // initialSettings seeds the panel for SSR/unit tests; when omitted the panel
  // fetches from /api/settings on mount.
  initialSettings?: Record<string, unknown>;
}

// GeneralSettingsPanel (PAR-UI-097/098) manages the theme (via the FROZEN
// useTheme().setTheme, plan §1.4) and the require_login settings key, persisting
// the latter through the REAL PUT /api/settings flat map.
export function GeneralSettingsPanel({ initialSettings }: GeneralSettingsPanelProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const { theme, setTheme } = useTheme();
  const [requireLogin, setRequireLogin] = React.useState<boolean>(
    Boolean(initialSettings?.require_login ?? false)
  );
  const [loading, setLoading] = React.useState(initialSettings === undefined);
  const [saving, setSaving] = React.useState(false);

  React.useEffect(() => {
    if (initialSettings !== undefined) return;
    apiFetch<Record<string, unknown>>("/api/settings")
      .then((settings) => {
        setRequireLogin(Boolean(settings?.require_login ?? false));
        setLoading(false);
      })
      .catch(() => {
        setLoading(false);
        pushToast({ message: "Failed to load settings" });
      });
  }, [initialSettings, pushToast]);

  async function save() {
    setSaving(true);
    try {
      await apiFetch("/api/settings", {
        method: "PUT",
        body: JSON.stringify({ require_login: String(requireLogin) }),
      });
      pushToast({ message: "Settings saved" });
    } catch {
      pushToast({ message: "Failed to save settings" });
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>General</CardTitle>
      </CardHeader>
      <CardContent className="mt-4 flex flex-col gap-6">
        <div data-testid="theme-segmented" className="flex flex-col gap-2">
          <span className="text-sm font-medium text-foreground">Theme</span>
          <SegmentedControl
            options={THEME_OPTIONS}
            value={theme}
            onChange={(value) => setTheme(value as Theme)}
          />
        </div>

        <div className="flex items-center justify-between">
          <label htmlFor="require-login" className="text-sm font-medium text-foreground">
            Require login
          </label>
          <Toggle
            id="require-login"
            checked={requireLogin}
            onCheckedChange={setRequireLogin}
            aria-label="Require login"
          />
        </div>

        <div className="flex justify-end">
          <Button data-testid="save-general" variant="primary" loading={saving} onClick={save}>
            Save
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
