import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useNotificationStore } from "@/stores/notification";

// PasswordPanel (PAR-UI-100) changes the operator password via the existing
// PUT /api/auth/password mock (plan §1.4; consumed from the registered auth
// handler — no real Go endpoint yet, §8 ESC-2). It validates the new/confirm
// match client-side before submitting.
export function PasswordPanel() {
  const pushToast = useNotificationStore((state) => state.push);
  const [current, setCurrent] = React.useState("");
  const [next, setNext] = React.useState("");
  const [confirm, setConfirm] = React.useState("");
  const [saving, setSaving] = React.useState(false);

  const mismatch = confirm.length > 0 && next !== confirm;

  async function submit() {
    if (!current || !next) {
      pushToast({ message: "Enter the current and new password" });
      return;
    }
    if (next !== confirm) {
      pushToast({ message: "New password and confirmation do not match" });
      return;
    }
    setSaving(true);
    try {
      await apiFetch("/api/auth/password", {
        method: "PUT",
        body: JSON.stringify({ current_password: current, new_password: next }),
      });
      pushToast({ message: "Password changed" });
      setCurrent("");
      setNext("");
      setConfirm("");
    } catch {
      pushToast({ message: "Failed to change the password" });
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Change Password</CardTitle>
      </CardHeader>
      <CardContent className="mt-4 flex flex-col gap-4">
        <Input
          data-testid="password-current"
          label="Current password"
          type="password"
          value={current}
          onChange={(event) => setCurrent(event.target.value)}
        />
        <Input
          data-testid="password-new"
          label="New password"
          type="password"
          value={next}
          onChange={(event) => setNext(event.target.value)}
        />
        <Input
          data-testid="password-confirm"
          label="Confirm new password"
          type="password"
          value={confirm}
          onChange={(event) => setConfirm(event.target.value)}
          error={mismatch ? "Passwords do not match" : undefined}
        />
        <div className="flex justify-end">
          <Button
            data-testid="password-submit"
            variant="primary"
            loading={saving}
            disabled={mismatch}
            onClick={submit}
          >
            Change password
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
