import type { ApiKey } from "../../src/lib/types";

// The mock seed mirrors the REAL Go apiKeyDTO (internal/admin/apikeys.go:11-17):
// {id, key, name, machine_id, is_active, created_at}. The frozen UI ApiKey type's
// extra fields (prefix/scopes/...) are display-optional, so the seed is cast to it
// (plan §1.4 / §8 ESC-2: mocks mirror reality).
export function seedKeys(): ApiKey[] {
  return [
    {
      id: "key-1",
      key: "sk-g0def-1234567890abcdef",
      name: "Default Key",
      machine_id: "machine-default",
      is_active: true,
      created_at: new Date(Date.now() - 86400000 * 7).toISOString(),
    } as unknown as ApiKey,
    {
      id: "key-2",
      key: "sk-g0stg-0987654321zyxwvu",
      name: "Staging Key",
      machine_id: "machine-staging",
      is_active: true,
      created_at: new Date(Date.now() - 86400000 * 3).toISOString(),
    } as unknown as ApiKey,
  ];
}
