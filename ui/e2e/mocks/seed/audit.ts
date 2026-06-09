import type { AuditLog } from "../../src/lib/types";

export function seedAuditLogs(): AuditLog[] {
  return [
    { id: "audit-1", timestamp: new Date(Date.now() - 3600000).toISOString(), actor: "admin", action: "create_key", target: "key-1", details: "Created Default Key" },
    { id: "audit-2", timestamp: new Date(Date.now() - 7200000).toISOString(), actor: "admin", action: "copy_key", target: "key-1" },
    { id: "audit-3", timestamp: new Date(Date.now() - 86400000).toISOString(), actor: "admin", action: "enable_key", target: "key-2" },
    { id: "audit-4", timestamp: new Date(Date.now() - 172800000).toISOString(), actor: "admin", action: "export_keys", target: "all", details: "Exported 2 keys" },
    { id: "audit-5", timestamp: new Date(Date.now() - 259200000).toISOString(), actor: "admin", action: "regenerate_key", target: "key-1" },
  ];
}
