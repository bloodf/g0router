import * as React from "react";
import { Globe } from "lucide-react";

import { apiFetch } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Modal } from "./modal";

export interface LocaleEntry {
  code: string;
  flag: string;
  label: string;
}

export const DEFAULT_LOCALES: LocaleEntry[] = [
  { code: "en", flag: "🇬🇧", label: "English" },
  { code: "zh-CN", flag: "🇨🇳", label: "简体中文" },
  { code: "ja", flag: "🇯🇵", label: "日本語" },
  { code: "pt-BR", flag: "🇧🇷", label: "Português (Brasil)" },
  { code: "ko", flag: "🇰🇷", label: "한국어" },
  { code: "es", flag: "🇪🇸", label: "Español" },
  { code: "de", flag: "🇩🇪", label: "Deutsch" },
  { code: "fr", flag: "🇫🇷", label: "Français" },
  { code: "ru", flag: "🇷🇺", label: "Русский" },
  { code: "it", flag: "🇮🇹", label: "Italiano" },
];

export interface LanguageSwitcherProps {
  locales?: LocaleEntry[];
  current?: string;
  onChange?: (code: string) => void;
  defaultOpen?: boolean;
}

function LanguageSwitcher({
  locales = DEFAULT_LOCALES,
  current = "en",
  onChange,
  defaultOpen = false,
}: LanguageSwitcherProps) {
  const [open, setOpen] = React.useState(defaultOpen);
  const active = locales.find((locale) => locale.code === current) ?? locales[0];

  async function pick(code: string) {
    await apiFetch("/api/locale", {
      method: "POST",
      body: JSON.stringify({ locale: code }),
    });
    onChange?.(code);
    setOpen(false);
  }

  return (
    <>
      <button
        type="button"
        aria-haspopup="dialog"
        aria-label={`Language: ${active.label}`}
        onClick={() => setOpen(true)}
        className="inline-flex size-8 items-center justify-center rounded-md border border-border text-foreground transition-colors hover:bg-muted"
      >
        <span aria-hidden="true">{active.flag}</span>
        <Globe className="sr-only size-4" />
      </button>

      <Modal open={open} onClose={() => setOpen(false)} title="Select language">
        <div className="grid grid-cols-3 gap-2">
          {locales.map((locale) => (
            <button
              key={locale.code}
              type="button"
              data-testid="locale-option"
              aria-label={`${locale.label} (${locale.code})`}
              onClick={() => pick(locale.code)}
              className={cn(
                "flex flex-col items-center gap-1 rounded-md border p-2 text-xs transition-colors hover:bg-muted",
                locale.code === current
                  ? "border-primary"
                  : "border-border"
              )}
            >
              <span aria-hidden="true" className="text-xl">
                {locale.flag}
              </span>
              <span>{locale.code}</span>
            </button>
          ))}
        </div>
      </Modal>
    </>
  );
}

export { LanguageSwitcher };
