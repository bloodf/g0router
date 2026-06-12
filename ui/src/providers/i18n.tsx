import React, { createContext, useContext, useEffect, useState } from "react";
import { useRouter } from "@tanstack/react-router";
import i18n from "../i18n";
import { LOCALES, type Locale } from "../i18n/locales";
import { apiFetch } from "../lib/api";

interface I18nContextValue {
  currentLocale: string;
  locales: Locale[];
  setLocale: (code: string) => Promise<void>;
}

const I18nContext = createContext<I18nContextValue>({
  currentLocale: "en",
  locales: LOCALES,
  setLocale: async () => {},
});

export const useI18n = () => useContext(I18nContext);

function readLocaleCookie(): string {
  const match = document.cookie
    .split(";")
    .find((c) => c.trim().startsWith("locale="));
  return match ? match.split("=")[1].trim() : "";
}

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [currentLocale, setCurrentLocale] = useState(() =>
    readLocaleCookie() || navigator.language.slice(0, 2) || "en"
  );
  const router = useRouter();

  useEffect(() => {
    i18n.changeLanguage(currentLocale);
  }, [currentLocale]);

  useEffect(() => {
    const unsub = router.subscribe("onResolved", () => {
      i18n.changeLanguage(currentLocale);
    });
    return unsub;
  }, [router, currentLocale]);

  const setLocale = async (code: string) => {
    await apiFetch("/api/locale", {
      method: "POST",
      body: JSON.stringify({ locale: code }),
    });
    setCurrentLocale(code);
    i18n.changeLanguage(code);
  };

  return (
    <I18nContext.Provider value={{ currentLocale, locales: LOCALES, setLocale }}>
      {children}
    </I18nContext.Provider>
  );
}
