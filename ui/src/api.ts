export type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
};

export type ListResponse<T> = {
  data: T[];
};

export type ProviderMatrixEntry = {
  id: string;
  auth_types?: string[] | null;
  oauth_provider?: string;
  refresh: boolean;
  registered_adapter: boolean;
  public_inference: boolean;
  direct_dispatch: boolean;
  inference: boolean;
  streaming: boolean;
  model_catalog: boolean;
  list_models: boolean;
  quota: boolean;
  public_status: string;
  notes?: string;
};

export type ProviderModel = {
  id: string;
  object: string;
  created: number;
  owned_by: string;
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

export type ConnectionResponse = {
  ID: string;
  Provider: string;
  Name: string;
  AuthType: string;
  ExpiresAt?: number | null;
  IsActive: boolean;
  ProviderSpecificData?: Record<string, unknown>;
  AccountID?: string | null;
  Email?: string | null;
  UnavailableUntil?: number | null;
  BackoffLevel: number;
  ModelLocks?: Record<string, number>;
  NeedsReauth: boolean;
  LastRefreshError?: string | null;
  CreatedAt: string;
  UpdatedAt: string;
};

export type CreateConnectionRequest = {
  provider: string;
  name: string;
  auth_type: "api_key" | "oauth" | "noauth";
  api_key?: string;
  account_id?: string;
  is_active: boolean;
};

export type UpdateConnectionRequest = {
  provider: string;
  name: string;
  auth_type: string;
  expires_at?: number | null;
  is_active: boolean;
  provider_specific_data?: Record<string, unknown>;
  account_id?: string | null;
  email?: string | null;
  unavailable_until?: number | null;
  backoff_level: number;
  model_locks?: Record<string, number>;
};

export type ConnectionTestResponse = {
  ok: boolean;
  provider: string;
  name: string;
};

export type ProviderOAuthStartResponse = {
  provider: string;
  auth_url?: string;
  session_id?: string;
  user_code?: string;
  verification?: string;
  expires_in?: number;
  poll_interval?: number;
};

export type ProviderOAuthConnectionResponse = {
  id: string;
  provider: string;
  name: string;
  auth_type: string;
  expires_at?: number | null;
  scopes?: string[];
};

export type ProviderOAuthPollResponse = {
  status: "pending" | "complete" | "slow_down" | "expired" | "denied";
  connection?: ProviderOAuthConnectionResponse;
};

export type APIKeyResponse = {
  ID: string;
  Name: string;
  Prefix: string;
  IsActive: boolean;
  LastUsedAt?: string | null;
  CreatedAt: string;
};

export type CreateAPIKeyResponse = {
  key: APIKeyResponse;
  raw: string;
};

export type SettingsResponse = {
  require_api_key: boolean;
  rtk_enabled: boolean;
  caveman_enabled: boolean;
  caveman_level: string;
  enable_request_logs: boolean;
  proxy_url: string;
  data_dir: string;
  log_retention_days: number;
  allowed_sources: string[];
};

export type UsageLogRecord = {
  id: number;
  request_id: string;
  timestamp: string;
  provider: string;
  model: string;
  connection_id?: string | null;
  auth_type: string;
  input_tokens?: number | null;
  output_tokens?: number | null;
  cache_read_tokens?: number | null;
  cache_write_tokens?: number | null;
  total_tokens?: number | null;
  cost_usd?: number | null;
  latency_ms?: number | null;
  status_code?: number | null;
  error?: string | null;
  source_format?: string | null;
  target_format?: string | null;
  rtk_enabled?: boolean | null;
  rtk_bytes_saved?: number | null;
  caveman_enabled?: boolean | null;
  combo_name?: string | null;
  api_key_id?: string | null;
  api_key_name?: string | null;
  client_tool?: string | null;
  connection_name?: string | null;
  connection_provider?: string | null;
  account_email?: string | null;
};

export type UsageQuery = {
  api_key_id?: string;
  auth_type?: string;
};

export type UsageListResponse = {
  object: "list";
  data: UsageLogRecord[];
  limit: number;
  offset: number;
  total: number;
};

export type LogQuery = {
  limit?: number;
  offset?: number;
  provider?: string;
  model?: string;
  auth_type?: string;
  source_format?: string;
  status_class?: string;
  search?: string;
  start?: string;
  end?: string;
};

export type UsageSummaryResponse = {
  request_count: number;
  total_tokens: number;
  total_cost_usd: number;
};

export type QuotaResponse = {
  Provider: string;
  Limit: number;
  Used: number;
  Remaining: number;
  Unlimited?: boolean;
  Unit?: string;
};

export type ComboStepResponse = {
  provider: string;
  model: string;
};

export type ComboStrategy = "fallback" | "round_robin" | "least_used" | "auto";

export type ComboResponse = {
  ID: string;
  Name: string;
  Steps: ComboStepResponse[];
  Strategy: ComboStrategy;
  IsActive: boolean;
  CreatedAt: string;
  UpdatedAt: string;
};

export type ModelAliasResponse = {
  Alias: string;
  Provider: string;
  Model: string;
};

export type PricingOverrideResponse = {
  Provider: string;
  Model: string;
  InputCostPerToken: number;
  OutputCostPerToken: number;
};

export type MCPManifestTool = {
  name: string;
  description?: string;
  input_schema?: unknown;
};

export type MCPManifest = {
  tools?: MCPManifestTool[];
};

export type MCPInstanceResponse = {
  ID: string;
  Name: string;
  ServerKey: string;
  LaunchType: string;
  Transport: string;
  Command?: string | null;
  Args?: string[];
  URL?: string | null;
  Headers?: Record<string, string>;
  Env?: Record<string, string>;
  CWD?: string | null;
  AccountLabel?: string | null;
  IsActive: boolean;
  HealthStatus: string;
  LastHealthCheck?: string | null;
  ToolManifest?: MCPManifest | null;
  ManifestUpdatedAt?: string | null;
  CreatedAt: string;
  UpdatedAt: string;
};

export type MCPClientResponse = {
  ID: string;
  Name: string;
  Transport: string;
  Command?: string | null;
  Args?: string[];
  URL?: string | null;
  Env?: Record<string, string>;
  IsActive: boolean;
  HealthStatus: string;
  LastHealthCheck?: string | null;
  ToolManifest?: MCPManifest | null;
  ManifestUpdatedAt?: string | null;
  CreatedAt: string;
};

export type MCPOAuthAccountResponse = {
  id: string;
  instance_id: string;
  account_label: string;
  subject?: string;
  email?: string;
  issuer?: string;
  resource_uri?: string;
  scopes?: string[];
  expires_at?: string;
  created_at?: string;
  updated_at?: string;
};

export type MCPToolResponse = {
  type: string;
  function: {
    name: string;
    description?: string;
    parameters?: unknown;
  };
};

export type MCPToolExecuteResponse = {
  content: unknown;
};

export type LoadStatus = "idle" | "loading" | "success" | "empty" | "error" | "auth-expired";

export type AsyncState<T> =
  | { status: "idle" | "loading" }
  | { status: "success"; data: T }
  | { status: "empty"; data: T }
  | { status: "error"; error: ApiError }
  | { status: "auth-expired"; error: ApiError };

export class ApiError extends Error {
  readonly authExpired: boolean;
  readonly body: unknown;
  readonly status: number;

  constructor(status: number, message: string, body: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
    this.authExpired = status === 401 || status === 403;
  }
}

const controlPlaneKeyStorageKey = "g0router.controlPlaneKey";

const apiPaths = {
  providers: "/api/providers",
  connections: "/api/connections",
  apiKeys: "/api/keys",
  aliases: "/api/aliases",
  pricing: "/api/pricing",
  usage: "/api/usage",
  usageSummary: "/api/usage/summary",
  logs: "/api/logs",
  combos: "/api/combos",
  mcpClients: "/api/mcp/clients",
  mcpServers: "/api/mcp/instances",
  mcpTools: "/api/mcp/tools",
  settings: "/api/settings"
} as const;

export function getControlPlaneKey() {
  try {
    return globalThis.localStorage?.getItem(controlPlaneKeyStorageKey) ?? "";
  } catch {
    return "";
  }
}

export function saveControlPlaneKey(key: string) {
  const trimmed = key.trim();
  try {
    if (trimmed) {
      globalThis.localStorage?.setItem(controlPlaneKeyStorageKey, trimmed);
    } else {
      globalThis.localStorage?.removeItem(controlPlaneKeyStorageKey);
    }
  } catch {
    // Storage can be unavailable in restricted browser contexts.
  }
}

export function clearControlPlaneKey() {
  try {
    globalThis.localStorage?.removeItem(controlPlaneKeyStorageKey);
  } catch {
    // Storage can be unavailable in restricted browser contexts.
  }
}

export function getProvidersPath() {
  return apiPaths.providers;
}

export function getProviderModelsPath(provider: string) {
  return `${apiPaths.providers}/${encodeURIComponent(provider)}/models`;
}

export function getConnectionsPath() {
  return apiPaths.connections;
}

export function getProviderOAuthAuthorizePath(provider: string) {
  return `/api/oauth/${encodeURIComponent(provider)}/authorize`;
}

export function getProviderOAuthExchangePath(provider: string) {
  return `/api/oauth/${encodeURIComponent(provider)}/exchange`;
}

export function getProviderOAuthPollPath(provider: string, sessionID: string) {
  return `/api/oauth/${encodeURIComponent(provider)}/poll?session_id=${encodeURIComponent(sessionID)}`;
}

export function getApiKeysPath() {
  return apiPaths.apiKeys;
}

export function getAliasesPath() {
  return apiPaths.aliases;
}

export function getPricingPath() {
  return apiPaths.pricing;
}

export function getUsagePath() {
  return apiPaths.usage;
}

export function buildUsagePath(query: UsageQuery = {}): string {
  const params = new URLSearchParams();
  const append = (key: string, value: string | undefined) => {
    const trimmed = value?.trim();
    if (trimmed) {
      params.set(key, trimmed);
    }
  };
  append("api_key_id", query.api_key_id);
  append("auth_type", query.auth_type);
  const qs = params.toString();
  return qs ? `${apiPaths.usage}?${qs}` : apiPaths.usage;
}

export function getUsageSummaryPath() {
  return apiPaths.usageSummary;
}

export function getQuotaPath(provider: string) {
  return `${apiPaths.usage}/quota/${encodeURIComponent(provider)}`;
}

export function getLogsPath() {
  return apiPaths.logs;
}

export function buildLogsPath(query: LogQuery = {}): string {
  const params = new URLSearchParams();
  const append = (key: string, value: string | undefined) => {
    const trimmed = value?.trim();
    if (trimmed) {
      params.set(key, trimmed);
    }
  };
  if (query.limit != null) {
    params.set("limit", String(query.limit));
  }
  if (query.offset != null) {
    params.set("offset", String(query.offset));
  }
  append("provider", query.provider);
  append("model", query.model);
  append("auth_type", query.auth_type);
  append("source_format", query.source_format);
  append("status_class", query.status_class);
  append("search", query.search);
  append("start", query.start);
  append("end", query.end);
  const qs = params.toString();
  return qs ? `${apiPaths.logs}?${qs}` : apiPaths.logs;
}

export function getCombosPath() {
  return apiPaths.combos;
}

export function getMcpClientsPath() {
  return apiPaths.mcpClients;
}

export function getMcpServersPath() {
  return apiPaths.mcpServers;
}

export function getMcpAccountsPath(instanceID: string) {
  return `${apiPaths.mcpServers}/${encodeURIComponent(instanceID)}/accounts`;
}

export function getMcpToolsPath() {
  return apiPaths.mcpTools;
}

export function getSettingsPath() {
  return apiPaths.settings;
}

export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const optionHeaders = options.headers as Record<string, string> | undefined;
  const headers = {
    "Content-Type": "application/json",
    ...optionHeaders
  } as Record<string, string>;
  const savedKey = getControlPlaneKey();
  if (savedKey && !headers.Authorization && !headers["X-API-Key"]) {
    headers.Authorization = `Bearer ${savedKey}`;
  }

  const response = await fetch(path, {
    ...options,
    credentials: options.credentials ?? "same-origin",
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body)
  });

  const payload = await readResponsePayload(response);
  if (!response.ok) {
    throw new ApiError(response.status, errorMessage(response, payload), payload);
  }

  return payload as T;
}

export async function apiList<T>(path: string, options?: RequestOptions): Promise<T[]> {
  const response = await apiFetch<ListResponse<T>>(path, options);
  return response.data ?? [];
}

export function isAuthExpiredError(error: unknown): error is ApiError {
  return error instanceof ApiError && error.authExpired;
}

export function asyncSuccess<T>(data: T): AsyncState<T> {
  if (Array.isArray(data) && data.length === 0) {
    return { status: "empty", data };
  }
  return { status: "success", data };
}

export function asyncError<T>(error: ApiError): AsyncState<T> {
  return error.authExpired ? { status: "auth-expired", error } : { status: "error", error };
}

export function listProviders() {
  return apiList<ProviderMatrixEntry>(getProvidersPath());
}

export function listProviderModels(provider: string) {
  return apiList<ProviderModel>(getProviderModelsPath(provider));
}

export function listConnections() {
  return apiList<ConnectionResponse>(getConnectionsPath());
}

export function createConnection(request: CreateConnectionRequest) {
  return apiFetch<ConnectionResponse>(getConnectionsPath(), { method: "POST", body: request });
}

export function updateConnection(connection: ConnectionResponse) {
  const body: UpdateConnectionRequest = {
    provider: connection.Provider,
    name: connection.Name,
    auth_type: connection.AuthType,
    expires_at: connection.ExpiresAt ?? null,
    is_active: connection.IsActive,
    provider_specific_data: redactConnectionMetadata(connection.ProviderSpecificData ?? {}),
    account_id: connection.AccountID ?? null,
    email: connection.Email ?? null,
    unavailable_until: connection.UnavailableUntil ?? null,
    backoff_level: connection.BackoffLevel,
    model_locks: connection.ModelLocks ?? {}
  };
  return apiFetch<ConnectionResponse>(`${getConnectionsPath()}/${encodeURIComponent(connection.ID)}`, { method: "PUT", body });
}

export function deleteConnection(id: string) {
  return apiFetch<void>(`${getConnectionsPath()}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function testConnection(id: string) {
  return apiFetch<ConnectionTestResponse>(`${getConnectionsPath()}/${encodeURIComponent(id)}/test`, { method: "POST" });
}

export function startProviderOAuth(provider: string, accountLabel: string) {
  return apiFetch<ProviderOAuthStartResponse>(getProviderOAuthAuthorizePath(provider), {
    method: "POST",
    body: { account_label: accountLabel }
  });
}

export function exchangeProviderOAuth(provider: string, state: string, code: string) {
  return apiFetch<ProviderOAuthConnectionResponse>(getProviderOAuthExchangePath(provider), {
    method: "POST",
    body: { state, code }
  });
}

export function pollProviderOAuth(provider: string, sessionID: string) {
  return apiFetch<ProviderOAuthPollResponse>(getProviderOAuthPollPath(provider, sessionID), { method: "GET" });
}

export function listAPIKeys() {
  return apiList<APIKeyResponse>(getApiKeysPath());
}

export function createAPIKey(name: string) {
  return apiFetch<CreateAPIKeyResponse>(getApiKeysPath(), { method: "POST", body: { name } });
}

export function deleteAPIKey(id: string) {
  return apiFetch<void>(`${getApiKeysPath()}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function listAliases() {
  return apiList<ModelAliasResponse>(getAliasesPath());
}

export function createAlias(alias: string, provider: string, model: string) {
  return apiFetch<ModelAliasResponse>(getAliasesPath(), { method: "POST", body: { alias, provider, model } });
}

export function updateAlias(alias: string, provider: string, model: string) {
  return apiFetch<ModelAliasResponse>(`${getAliasesPath()}/${encodeURIComponent(alias)}`, {
    method: "PUT",
    body: { provider, model }
  });
}

export function deleteAlias(alias: string) {
  return apiFetch<void>(`${getAliasesPath()}/${encodeURIComponent(alias)}`, { method: "DELETE" });
}

export function listPricingOverrides() {
  return apiList<PricingOverrideResponse>(getPricingPath());
}

export function createPricingOverride(
  provider: string,
  model: string,
  inputCostPerToken: number,
  outputCostPerToken: number
) {
  return apiFetch<PricingOverrideResponse>(getPricingPath(), {
    method: "POST",
    body: {
      provider,
      model,
      input_cost_per_token: inputCostPerToken,
      output_cost_per_token: outputCostPerToken
    }
  });
}

export function deletePricingOverride(provider: string, model: string) {
  return apiFetch<void>(`${getPricingPath()}/${encodeURIComponent(provider)}/${encodeURIComponent(model)}`, { method: "DELETE" });
}

export function updatePricingOverride(
  provider: string,
  model: string,
  inputCostPerToken: number,
  outputCostPerToken: number
) {
  return apiFetch<PricingOverrideResponse>(`${getPricingPath()}/${encodeURIComponent(provider)}/${encodeURIComponent(model)}`, {
    method: "PUT",
    body: {
      input_cost_per_token: inputCostPerToken,
      output_cost_per_token: outputCostPerToken
    }
  });
}

export function getSettings() {
  return apiFetch<SettingsResponse>(getSettingsPath());
}

export function updateSettings(settings: SettingsResponse) {
  return apiFetch<SettingsResponse>(getSettingsPath(), { method: "PUT", body: settings });
}

export function listUsage(query: UsageQuery = {}) {
  return apiFetch<UsageListResponse>(buildUsagePath(query));
}

export function getUsageSummary() {
  return apiFetch<UsageSummaryResponse>(getUsageSummaryPath());
}

export function getQuota(provider: string) {
  return apiFetch<QuotaResponse>(getQuotaPath(provider));
}

export function listLogs() {
  return apiFetch<UsageListResponse>(getLogsPath());
}

export function listCombos() {
  return apiList<ComboResponse>(getCombosPath());
}

export function createCombo(name: string, steps: ComboStepResponse[], isActive: boolean, strategy: ComboStrategy = "fallback") {
  return apiFetch<ComboResponse>(getCombosPath(), { method: "POST", body: { name, steps, is_active: isActive, strategy } });
}

export function updateCombo(id: string, name: string, steps: ComboStepResponse[], isActive: boolean, strategy: ComboStrategy = "fallback") {
  return apiFetch<ComboResponse>(`${getCombosPath()}/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: { name, steps, is_active: isActive, strategy }
  });
}

export function deleteCombo(id: string) {
  return apiFetch<void>(`${getCombosPath()}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function listMCPClients() {
  return apiList<MCPClientResponse>(getMcpClientsPath());
}

export function listMCPInstances() {
  return apiList<MCPInstanceResponse>(getMcpServersPath());
}

export function deleteMCPInstance(id: string) {
  return apiFetch<void>(`${getMcpServersPath()}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function listMCPAccounts(instanceID: string) {
  return apiList<MCPOAuthAccountResponse>(getMcpAccountsPath(instanceID));
}

export function listMCPTools() {
  return apiList<MCPToolResponse>(getMcpToolsPath());
}

export function completeMCPOAuth(instanceID: string, callbackURL: string) {
  return apiFetch<MCPOAuthAccountResponse>(`${getMcpServersPath()}/${encodeURIComponent(instanceID)}/oauth/complete`, {
    method: "POST",
    body: { callback_url: callbackURL }
  });
}

export function executeMCPTool(name: string, args: unknown, allowedTools: string[] = []) {
  return apiFetch<MCPToolExecuteResponse>(`${getMcpToolsPath()}/${encodeURIComponent(name)}/execute`, {
    method: "POST",
    body: { arguments: args, allowed_tools: allowedTools }
  });
}

async function readResponsePayload(response: Response): Promise<unknown> {
  if (response.status === 204) {
    return undefined;
  }

  const text = await response.text();
  if (text === "") {
    return undefined;
  }

  const contentType = response.headers.get("Content-Type") ?? "";
  if (contentType.includes("json")) {
    return JSON.parse(text);
  }

  return text;
}

function errorMessage(response: Response, payload: unknown): string {
  if (payload && typeof payload === "object") {
    const body = payload as { error?: unknown; message?: unknown };
    if (typeof body.error === "string" && body.error !== "") {
      return body.error;
    }
    if (typeof body.message === "string" && body.message !== "") {
      return body.message;
    }
  }
  return response.statusText || `request failed: ${response.status}`;
}

function redactConnectionMetadata(values: Record<string, unknown>): Record<string, unknown> {
  const redacted: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(values)) {
    if (isSecretMetadataKey(key)) {
      continue;
    }
    if (value && typeof value === "object" && !Array.isArray(value)) {
      redacted[key] = redactConnectionMetadata(value as Record<string, unknown>);
      continue;
    }
    redacted[key] = value;
  }
  return redacted;
}

function isSecretMetadataKey(key: string) {
  const normalized = key.toLowerCase();
  return ["token", "secret", "key", "authorization", "password"].some((marker) => normalized.includes(marker));
}
