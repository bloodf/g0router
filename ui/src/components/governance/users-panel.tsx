import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Modal } from "@/components/ui/modal";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { useNotificationStore } from "@/stores/notification";
import type { User } from "@/lib/types";

type ApiFetch = typeof apiFetch;

// changePassword is the pure PUT seam (chat-window streamChatCompletion
// precedent) so the PAR-UI-132 change-password call can be unit-tested with a
// stubbed apiFetch without simulating typed input in JSDOM.
export async function changePassword(
  current: string,
  next: string,
  fetchImpl: ApiFetch = apiFetch
): Promise<void> {
  await fetchImpl("/api/auth/password", {
    method: "PUT",
    body: JSON.stringify({ current_password: current, new_password: next }),
  });
}

export interface UsersPanelProps {
  // Test seam (general-settings-panel precedent): when provided, the panel
  // renders these rows without an initial fetch. The page mounts it without the
  // prop so it loads from /api/auth/users.
  initialUsers?: User[];
}

// UsersPanel surfaces the in-app user-management subset of PAR-UI-132 (plan
// §1.5). It CONSUMES the w6-c-owned auth.ts mock routes unchanged:
// GET/POST /api/auth/users, DELETE /api/auth/users/{id}, PUT /api/auth/password.
// Variant-HAVE against that mock; no Go user-management endpoints exist yet
// (§8 ESCALATION-2).
function UsersPanel({ initialUsers }: UsersPanelProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [users, setUsers] = React.useState<User[]>(initialUsers ?? []);
  const [loading, setLoading] = React.useState(!initialUsers);
  const [creating, setCreating] = React.useState(false);
  const [deleting, setDeleting] = React.useState<User | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);

  // New-user form state.
  const [newUsername, setNewUsername] = React.useState("");
  const [newDisplayName, setNewDisplayName] = React.useState("");
  const [newRole, setNewRole] = React.useState("user");
  const [newPassword, setNewPassword] = React.useState("");
  const [createBusy, setCreateBusy] = React.useState(false);

  // Change-password form state.
  const [currentPassword, setCurrentPassword] = React.useState("");
  const [changePasswordValue, setChangePasswordValue] = React.useState("");
  const [pwBusy, setPwBusy] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<User[]>("/api/auth/users")
      .then((rows) => {
        setUsers(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setUsers([]);
        setLoading(false);
        pushToast({ message: "Failed to load users" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    if (initialUsers) return;
    load();
  }, [initialUsers, load]);

  async function createUser() {
    setCreateBusy(true);
    try {
      await apiFetch("/api/auth/users", {
        method: "POST",
        body: JSON.stringify({
          username: newUsername,
          display_name: newDisplayName || newUsername,
          role: newRole,
          password: newPassword,
        }),
      });
      pushToast({ message: "User created" });
      setCreating(false);
      setNewUsername("");
      setNewDisplayName("");
      setNewRole("user");
      setNewPassword("");
      load();
    } catch {
      pushToast({ message: "Failed to create the user" });
    } finally {
      setCreateBusy(false);
    }
  }

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/auth/users/${deleting.id}`, { method: "DELETE" });
      setUsers((prev) => prev.filter((u) => u.id !== deleting.id));
      pushToast({ message: "User deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the user" });
    } finally {
      setDeleteBusy(false);
    }
  }

  async function submitPassword() {
    setPwBusy(true);
    try {
      await changePassword(currentPassword, changePasswordValue);
      pushToast({ message: "Password changed" });
      setCurrentPassword("");
      setChangePasswordValue("");
    } catch {
      pushToast({ message: "Failed to change the password" });
    } finally {
      setPwBusy(false);
    }
  }

  return (
    <section className="flex flex-col gap-4">
      <header className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">Users</h2>
        <Button
          data-testid="user-new"
          variant="primary"
          size="sm"
          onClick={() => setCreating(true)}
        >
          New user
        </Button>
      </header>

      {loading ? (
        <p className="text-sm text-muted-foreground">Loading users…</p>
      ) : users.length === 0 ? (
        <p className="text-sm text-muted-foreground">No users yet.</p>
      ) : (
        <div className="flex flex-col gap-2">
          {users.map((user) => (
            <div
              key={user.id}
              data-testid="user-row"
              className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
            >
              <div>
                <p className="text-sm font-medium text-foreground">{user.username}</p>
                <p className="text-xs text-muted-foreground">{user.display_name}</p>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant="neutral" size="sm">
                  {user.role}
                </Badge>
                <Button
                  data-testid="user-delete"
                  variant="danger"
                  size="sm"
                  onClick={() => setDeleting(user)}
                >
                  Delete
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="flex flex-col gap-3 rounded-lg border border-border px-4 py-3">
        <h3 className="text-sm font-semibold text-foreground">Change password</h3>
        <Input
          aria-label="Current password"
          type="password"
          placeholder="Current password"
          value={currentPassword}
          onChange={(event) => setCurrentPassword(event.target.value)}
        />
        <Input
          aria-label="New password"
          type="password"
          placeholder="New password"
          value={changePasswordValue}
          onChange={(event) => setChangePasswordValue(event.target.value)}
        />
        <div className="flex justify-end">
          <Button
            data-testid="user-change-password"
            variant="primary"
            size="sm"
            loading={pwBusy}
            onClick={submitPassword}
          >
            Change password
          </Button>
        </div>
      </div>

      <Modal
        open={creating}
        onClose={() => setCreating(false)}
        title="New user"
      >
        <div className="flex flex-col gap-4">
          <Input
            id="user-username"
            label="Username"
            value={newUsername}
            onChange={(event) => setNewUsername(event.target.value)}
          />
          <Input
            id="user-display-name"
            label="Display name"
            value={newDisplayName}
            onChange={(event) => setNewDisplayName(event.target.value)}
          />
          <Select
            id="user-role"
            label="Role"
            value={newRole}
            onChange={(event) => setNewRole(event.target.value)}
            options={[
              { value: "user", label: "User" },
              { value: "admin", label: "Admin" },
            ]}
          />
          <Input
            id="user-password"
            label="Password"
            type="password"
            value={newPassword}
            onChange={(event) => setNewPassword(event.target.value)}
          />
          <div className="flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setCreating(false)}>
              Cancel
            </Button>
            <Button
              data-testid="user-save"
              variant="primary"
              loading={createBusy}
              onClick={createUser}
            >
              Save
            </Button>
          </div>
        </div>
      </Modal>

      <ConfirmModal
        open={deleting !== null}
        title="Delete user"
        message={`Delete "${deleting?.username ?? ""}"? This cannot be undone.`}
        confirmLabel="Delete"
        cancelLabel="Cancel"
        variant="danger"
        loading={deleteBusy}
        onConfirm={confirmDelete}
        onCancel={() => setDeleting(null)}
      />
    </section>
  );
}

export { UsersPanel };
