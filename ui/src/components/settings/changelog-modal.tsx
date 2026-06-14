import * as React from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Modal } from "@/components/ui/modal";
import { apiFetch } from "@/lib/api";

export interface ChangelogModalProps {
  open: boolean;
  onClose: () => void;
}

// ChangelogModal (PAR-UI-056) fetches the changelog source from the mock-route
// /api/version/changelog (no outbound network in tests, plan §1.7b) and renders
// it with the already-installed react-markdown (no new dependency, §8 ESC-7).
export function ChangelogModal({ open, onClose }: ChangelogModalProps) {
  const [changelog, setChangelog] = React.useState("");
  const [loading, setLoading] = React.useState(false);

  React.useEffect(() => {
    if (!open) return;
    let active = true;
    setLoading(true);
    apiFetch<{ changelog: string }>("/api/version/changelog")
      .then((data) => {
        if (active) setChangelog(data?.changelog ?? "");
      })
      .catch(() => {
        if (active) setChangelog("Failed to load the changelog.");
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [open]);

  return (
    <Modal open={open} onClose={onClose} title="Changelog" size="lg">
      <div data-testid="changelog-modal" className="max-h-[60vh] overflow-y-auto">
        {loading ? (
          <p className="text-sm text-muted-foreground">Loading changelog…</p>
        ) : (
          <div className="prose prose-sm max-w-none text-sm text-foreground">
            <Markdown remarkPlugins={[remarkGfm]}>{changelog}</Markdown>
          </div>
        )}
      </div>
    </Modal>
  );
}
