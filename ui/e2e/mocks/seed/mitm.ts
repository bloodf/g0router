import type { MitmTool } from "../../src/lib/types";

export function seedMitmStatus() {
  return {
    enabled: false,
    ca_cert: "-----BEGIN CERTIFICATE-----\nMIIBkTCB+wIJAJHGTVDE...\n-----END CERTIFICATE-----",
    tools: [
      { id: "mitm-1", name: "Request Inspector", enabled: true, dns_override: "localhost", status: "active" as const },
      { id: "mitm-2", name: "Response Modifier", enabled: false, dns_override: "", status: "inactive" as const },
    ] as MitmTool[],
  };
}
