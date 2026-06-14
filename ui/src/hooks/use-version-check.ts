import * as React from "react";
import { apiFetch } from "@/lib/api";
import { useSettingsStore } from "@/stores/settings";

// VersionInfo mirrors the GET /api/version DTO (internal/admin/version.go):
// {version, build_date, update_available, latest_version} (plan §1.5).
interface VersionInfo {
  version: string;
  build_date: string;
  update_available: boolean;
  latest_version: string;
}

export interface UseVersionCheckResult {
  version: string;
  buildDate: string;
  updateAvailable: boolean;
  latestVersion: string;
  loading: boolean;
}

// useVersionCheck (PAR-UI-021/102) fetches /api/version on mount and, when an
// update is available, calls the FROZEN settingsStore.setUpdateInfo action (the
// sanctioned bridge that lights the frozen sidebar update-badge, plan §1.6). It
// returns the version metadata for the settings about-block to display. It does
// NOT edit the sidebar or the settings store definition.
export function useVersionCheck(): UseVersionCheckResult {
  const [version, setVersion] = React.useState("");
  const [buildDate, setBuildDate] = React.useState("");
  const [updateAvailable, setUpdateAvailable] = React.useState(false);
  const [latestVersion, setLatestVersion] = React.useState("");
  const [loading, setLoading] = React.useState(true);

  React.useEffect(() => {
    let active = true;
    apiFetch<VersionInfo>("/api/version")
      .then((info) => {
        if (!active || !info) return;
        setVersion(info.version ?? "");
        setBuildDate(info.build_date ?? "");
        setUpdateAvailable(Boolean(info.update_available));
        setLatestVersion(info.latest_version ?? "");
        if (info.update_available && info.latest_version) {
          useSettingsStore
            .getState()
            .setUpdateInfo(true, info.latest_version);
        }
      })
      .catch(() => {
        // A failed version check is non-fatal; leave defaults.
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  return { version, buildDate, updateAvailable, latestVersion, loading };
}
