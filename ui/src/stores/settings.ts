import { create } from "zustand";
import { persist } from "zustand/middleware";

interface SettingsState {
  settings: Record<string, unknown>;
  updateAvailable: boolean;
  latestVersion: string | null;
  setSettings: (settings: Record<string, unknown>) => void;
  setUpdateInfo: (updateAvailable: boolean, latestVersion: string) => void;
}

export const useSettingsStore = create(
  persist<SettingsState>(
    (set) => ({
      settings: {},
      updateAvailable: false,
      latestVersion: null,
      setSettings: (settings) => set({ settings }),
      setUpdateInfo: (updateAvailable, latestVersion) =>
        set({ updateAvailable, latestVersion }),
    }),
    { name: "settings" }
  )
);
