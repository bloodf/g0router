export type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
};

export type ConnectionStatus = "connected" | "degraded" | "disconnected";

export type ProviderConnection = {
  id: string;
  provider: string;
  account: string;
  status: ConnectionStatus;
  models: number;
  lastCheck: string;
};

export type UsageRecord = {
  id: string;
  route: string;
  provider: string;
  model: string;
  tokens: number;
  costUsd: number;
  latencyMs: number;
  status: number;
};

export type QuotaSnapshot = {
  provider: string;
  used: number;
  limit: number;
  resetAt: string;
};

export type ComboRoute = {
  name: string;
  strategy: string;
  providers: string[];
};

const apiPaths = {
  connections: "/api/connections",
  apiKeys: "/api/keys",
  usage: "/api/usage",
  quota: "/api/quota",
  combos: "/api/combos",
  mcpServers: "/api/mcp/servers",
  settings: "/api/settings"
} as const;

export function getConnectionsPath() {
  return apiPaths.connections;
}

export function getApiKeysPath() {
  return apiPaths.apiKeys;
}

export function getUsagePath() {
  return apiPaths.usage;
}

export function getQuotaPath() {
  return apiPaths.quota;
}

export function getCombosPath() {
  return apiPaths.combos;
}

export function getMcpServersPath() {
  return apiPaths.mcpServers;
}

export function getSettingsPath() {
  return apiPaths.settings;
}

export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const response = await fetch(path, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options.headers
    },
    body: options.body === undefined ? undefined : JSON.stringify(options.body)
  });

  if (!response.ok) {
    throw new Error(`request failed: ${response.status}`);
  }

  return response.json() as Promise<T>;
}
