// ui/src/lib/types.ts
//
// Type declarations for the 32 domain entity types referenced by the e2e mocks
// in ui/e2e/mocks/. Field shapes are derived from the seed files in
// ui/e2e/mocks/seed/ — see .planning/parity/plans/w0-d-recon.md for the
// extraction command and the full field-shape enumeration.
//
// Out of scope for Wave 0: API client wrapper, fetch calls, react-query hooks.
// These types are declared as opaque interfaces so the e2e mocks can import
// them; the eventual API client (Wave 6) will add the runtime layer.

export interface AlertChannel {
  id: number;
  name: string;
  channel_type: string;
  config: Record<string, unknown>;
  events: string[];
  is_active: boolean;
  created_at: string;
}

export interface Alias {
  id: string;
  alias: string;
  provider: string;
  model: string;
}

export interface ApiKey {
  id: string;
  name: string;
  prefix: string;
  full_key?: string;
  scopes: string[];
  rpm_limit?: number;
  tpm_limit?: number;
  daily_spend_cap?: number;
  is_active: boolean;
  created_at: string;
}

export interface AuditLog {
  id: string;
  timestamp: string;
  actor: string;
  action: string;
  target: string;
  details?: string;
}

export interface AuthStatus {
  require_login: boolean;
  has_users: boolean;
  authenticated: boolean;
  username: string;
  display_name: string;
  role: string;
}

export interface ChatSession {
  id: string;
  title: string;
  model: string;
  provider: string;
  messages: Array<{ role: string; content: string }>;
  created_at: string;
  updated_at: string;
}

export interface Combo {
  id: string;
  name: string;
  strategy: string;
  steps: Array<{ provider: string; model: string }>;
  is_active: boolean;
}

export interface Connection {
  id: string;
  provider: string;
  name: string;
  auth_type: string;
  is_active: boolean;
  models: string[];
  priority: number;
  needs_reauth: boolean;
}

export interface ConsoleLogEntry {
  timestamp: string;
  level: string;
  message: string;
}

export interface FeatureFlag {
  id: number;
  key: string;
  enabled: boolean;
  description: string;
  created_at: string;
}

export interface Guardrails {
  guardrails_enabled: boolean;
  guardrails_blocklist: string[];
  pii_redaction_enabled: boolean;
  pii_redaction_types: string[];
}

export interface McpClient {
  ID: string;
  Name: string;
  Transport: string;
  Command?: string;
  Args?: string[];
  Env?: Record<string, string>;
  URL?: string;
  IsActive: boolean;
  HealthStatus: string;
  CreatedAt: string;
}

export interface McpInstance {
  ID: string;
  Name: string;
  Transport: string;
  Command?: string;
  Args?: string[];
  IsActive: boolean;
  HealthStatus: string;
  CreatedAt: string;
}

export interface McpTool {
  type: string;
  function: {
    name: string;
    description: string;
    parameters: Record<string, unknown>;
  };
}

export interface McpToolGroup {
  id: number;
  name: string;
  tool_ids: string[];
  is_active: boolean;
  created_at: string;
}

export interface MitmTool {
  id: string;
  name: string;
  enabled: boolean;
  dns_override: string;
  status: "active" | "inactive";
}

export interface Model {
  id: string;
  provider: string;
  name: string;
  input_cost: number;
  output_cost: number;
  context_window: number;
  is_disabled: boolean;
  is_custom: boolean;
}

export interface ModelLimit {
  id: number;
  model: string;
  max_tokens: number;
  max_rpm: number;
  allowed_key_ids: string[];
  created_at: string;
}

export interface PricingOverride {
  id: string;
  provider: string;
  model: string;
  input_cost: number;
  output_cost: number;
}

export interface PromptTemplate {
  id: number;
  name: string;
  system_prompt: string;
  models: string[];
  is_active: boolean;
  created_at: string;
}

export interface Provider {
  id: string;
  name: string;
  display_name: string;
  description: string;
  auth_types: string[];
  capabilities: string[];
  connection_count: number;
  status: string;
}

export interface ProxyPool {
  id: string;
  name: string;
  protocol: string;
  host: string;
  port: number;
  username: string;
  is_active: boolean;
  last_check_at: string;
  last_check_status: string;
}

export interface Quota {
  connection_id: string;
  provider: string;
  connection_name: string;
  account_label?: string;
  plan: string;
  used: number;
  limit: number;
  unit: string;
  reset_at: string;
  is_active: boolean;
}

export interface RoutingRule {
  id: string;
  name: string;
  priority: number;
  cond_field: string;
  cond_operator: string;
  cond_value: string;
  target_provider: string;
  is_active: boolean;
  created_at: string;
}

export interface Settings {
  require_api_key: boolean;
  require_login: boolean;
  rtk_enabled: boolean;
  caveman_enabled: boolean;
  caveman_level: string;
  enable_request_logs: boolean;
  log_retention_days: number;
  cache_enabled: boolean;
  cache_ttl_seconds: number;
  proxy_url: string;
  notify_webhook_url: string;
  notify_on_reauth: boolean;
  allowed_sources: string[];
  tunnel_dashboard_access: boolean;
  theme: string;
  language: string;
}

export interface Skill {
  name: string;
  category: string;
  description: string;
  url: string;
}

export interface Team {
  id: string;
  name: string;
  budget_usd: number;
  budget_used_usd: number;
  budget_period: string;
  rate_limit_rpm: number;
}

// No field references observed in seed files; opaque for now.
export interface TrafficEvent {
  [key: string]: unknown;
}

export interface Tunnel {
  type: string;
  is_enabled: boolean;
  url: string;
  status: string;
}

export interface UsageLog {
  id: string;
  timestamp: string;
  provider: string;
  model: string;
  api_key_id: string;
  api_key_name: string;
  status: string;
  status_code: number;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  cost_usd: number;
  latency_ms: number;
  rtk_enabled: boolean;
  caveman_enabled: boolean;
}

export interface User {
  id: string;
  username: string;
  display_name: string;
  role: string;
  password?: string;
}

export interface VirtualKey {
  id: string;
  name: string;
  prefix: string;
  budget_usd: number;
  budget_used_usd: number;
  budget_period: string;
  rate_limit_rpm: number;
  is_active: boolean;
}
