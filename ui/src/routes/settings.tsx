import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { GeneralSettingsPanel } from "@/components/settings/general-settings-panel";
import { LanguageSettingsPanel } from "@/components/settings/language-settings-panel";
import { ChangelogModal } from "@/components/settings/changelog-modal";
import { DonateModal } from "@/components/settings/donate-modal";
import { useVersionCheck } from "@/hooks/use-version-check";

export const Route = createFileRoute("/settings")({
  component: SettingsPage,
});

// SettingsPage (PAR-UI-097-103/021/055/056) composes the operator preference
// panels (plan §1.3). The about-block drives the version-check hook (lighting the
// FROZEN sidebar update-badge via setUpdateInfo, §1.6) and mounts the changelog +
// donate modals (§1.7b) — all from this w6-j-owned surface, no frozen-file edit.
function SettingsPage() {
  const { version, buildDate } = useVersionCheck();
  const [changelogOpen, setChangelogOpen] = React.useState(false);
  const [donateOpen, setDonateOpen] = React.useState(false);

  return (
    <div className="flex flex-col gap-6">
      <header>
        <h1 className="text-2xl font-semibold text-foreground">Settings</h1>
      </header>

      <GeneralSettingsPanel />
      <LanguageSettingsPanel />

      <Card>
        <CardHeader>
          <CardTitle>About</CardTitle>
        </CardHeader>
        <CardContent className="mt-4 flex flex-col gap-3">
          <div data-testid="about-version" className="text-sm text-foreground">
            <span className="font-medium">Version: </span>
            <span>{version || "unknown"}</span>
            {buildDate ? (
              <span className="text-muted-foreground"> (built {buildDate})</span>
            ) : null}
          </div>
          <div className="flex gap-2">
            <Button
              data-testid="open-changelog"
              variant="outline"
              size="sm"
              onClick={() => setChangelogOpen(true)}
            >
              View changelog
            </Button>
            <Button
              data-testid="open-donate"
              variant="outline"
              size="sm"
              onClick={() => setDonateOpen(true)}
            >
              Donate
            </Button>
          </div>
        </CardContent>
      </Card>

      <ChangelogModal open={changelogOpen} onClose={() => setChangelogOpen(false)} />
      <DonateModal open={donateOpen} onClose={() => setDonateOpen(false)} />
    </div>
  );
}
