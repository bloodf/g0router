import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import { LOCALES } from "./locales";

const modules = import.meta.glob<{ default?: Record<string, string> }>(
  "./locales/*.json",
  { eager: true }
);

const resources: Record<string, { translation: Record<string, string> }> = {};
LOCALES.forEach(({ code }) => {
  const mod = modules[`./locales/${code}.json`];
  const translation = mod?.default ?? mod ?? {};
  resources[code] = { translation };
});

i18n.use(initReactI18next).init({
  resources,
  lng: "en",
  fallbackLng: "en",
  interpolation: { escapeValue: false },
});

export default i18n;
