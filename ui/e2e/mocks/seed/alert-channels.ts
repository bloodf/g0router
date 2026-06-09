import type { AlertChannel } from "../../src/lib/types";

export function seedAlertChannels(): AlertChannel[] {
  return [
    { id: 1, name: "Webhook Alerts", channel_type: "webhook", config: { url: "https://hooks.example.com/g0router" }, events: ["quota_exceeded", "provider_error"], is_active: true, created_at: new Date(Date.now() - 86400000 * 10).toISOString() },
    { id: 2, name: "Discord Alerts", channel_type: "discord", config: { webhook_url: "https://discord.com/api/webhooks/xxx" }, events: ["provider_error"], is_active: false, created_at: new Date(Date.now() - 86400000 * 5).toISOString() },
  ];
}
