import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { apiFetch } from "@/lib/api";

interface DonateInfo {
  title: string;
  message: string;
  links: { label: string; url: string }[];
}

export interface DonateModalProps {
  open: boolean;
  onClose: () => void;
}

// DonateModal (PAR-UI-055) fetches the donate source from the mock-route
// /api/version/donate (no outbound network in tests, plan §1.7b) and renders the
// donation info. Mounted from the w6-j settings about-block (no frozen edit).
export function DonateModal({ open, onClose }: DonateModalProps) {
  const [info, setInfo] = React.useState<DonateInfo | null>(null);
  const [loading, setLoading] = React.useState(false);

  React.useEffect(() => {
    if (!open) return;
    let active = true;
    setLoading(true);
    apiFetch<DonateInfo>("/api/version/donate")
      .then((data) => {
        if (active) setInfo(data ?? null);
      })
      .catch(() => {
        if (active) setInfo(null);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [open]);

  return (
    <Modal open={open} onClose={onClose} title="Support g0router">
      <div data-testid="donate-modal" className="flex flex-col gap-3">
        {loading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : (
          <>
            <p className="text-sm font-medium text-foreground">
              {info?.title ?? "Support g0router"}
            </p>
            <p className="text-sm text-muted-foreground">
              {info?.message ?? "Donate to support development."}
            </p>
            <ul className="flex flex-col gap-2">
              {(info?.links ?? []).map((link) => (
                <li key={link.url}>
                  <a
                    href={link.url}
                    target="_blank"
                    rel="noreferrer"
                    className="text-sm text-primary underline"
                  >
                    {link.label}
                  </a>
                </li>
              ))}
            </ul>
          </>
        )}
      </div>
    </Modal>
  );
}
