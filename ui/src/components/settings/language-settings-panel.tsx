import * as React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Select } from "@/components/ui/select";
import { useI18n } from "@/providers/i18n";

// LanguageSettingsPanel lists the FROZEN i18n LOCALES and changes the active
// locale via the FROZEN useI18n().setLocale action (plan §1.3/§1.4).
export function LanguageSettingsPanel() {
  const { currentLocale, locales, setLocale } = useI18n();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Language</CardTitle>
      </CardHeader>
      <CardContent className="mt-4">
        <Select
          data-testid="language-select"
          label="Language"
          value={currentLocale}
          onChange={(event) => {
            void setLocale(event.target.value);
          }}
          options={locales.map((locale) => ({
            value: locale.code,
            label: locale.name,
          }))}
        />
      </CardContent>
    </Card>
  );
}
