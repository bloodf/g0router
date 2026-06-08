// Domain types — snake_case to match backend contract from spec.

export interface AuthStatus {
  require_login: boolean;
  has_users: boolean;
  authenticated: boolean;
  username?: string;
  display_name?: string;
  role?: "admin" | "user";
}

export interface User {
  id: string;
  username: string;
  display_name?: string;
  role: "admin" | "user";
  password: string; // hashed in real life
}

export interface Provider {
  id: string;
  name: string;
  display_name: string;
  description: string;
  auth_types: ("oauth" | "api_key" | "noauth" | "custom")[];
  capabilities: string[];
  icon_url?: string;
  connection_count: number;
  status: "active" | "inactive" | "error" | "needs_reauth";
}

export interface Connection {
  id: string;
  provider: string;
  name: string;
  auth_type: "oauth" | "api_key" | "noauth";
  is_active: boolean;
  models: string[];
  proxy_id?: string;
  priority: number;
  last_error?: string;
  needs_reauth: boolean;
  unavailable_until?: string;
  expires_at?: string;
}

export interface ApiKey {
  id: string;
  name: string;
  prefix: string;
  full_key?: string;
  scopes: string[];
  expires_at?: string;
  rpm_limit?: number;
  tpm_limit?: number;
  daily_spend_cap?: number;
  is_active: boolean;
  created_at: string;
}

export interface VirtualKey {
  id: string;
  name: string;
  prefix: string;
  budget_usd?: number;
  budget_used_usd: number;
  budget_period: "daily" | "weekly" | "monthly";
  rate_limit_rpm?: number;
  rate_limit_tpm?: number;
  team_id?: string;
  is_active: boolean;
}

export interface Team {
  id: string;
  name: string;
  budget_usd?: number;
  budget_used_usd: number;
  budget_period: "daily" | "weekly" | "monthly";
  rate_limit_rpm?: number;
}

export interface ComboStep {
  provider: string;
  model: string;
}

export interface Combo {
  id: string;
  name: string;
  strategy: "fallback" | "round_robin" | "least_used" | "auto" | "fastest" | "cheapest";
  steps: ComboStep[];
  sticky_limit?: number;
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
  target_model?: string;
  is_active: boolean;
  created_at: string;
}

export interface Model {
  id: string;
  provider: string;
  name: string;
  input_cost: number; // per million tokens
  output_cost: number;
  context_window: number;
  is_disabled: boolean;
  is_custom: boolean;
}

export interface Alias {
  id: string;
  alias: string;
  provider: string;
  model: string;
}

export interface PricingOverride {
  id: string;
  provider: string;
  model: string;
  input_cost: number;
  output_cost: number;
}

export interface UsageLog {
  id: string;
  timestamp: string;
  provider: string;
  model: string;
  api_key_id: string;
  api_key_name: string;
  status: "success" | "error";
  status_code: number;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  cost_usd: number;
  latency_ms: number;
  rtk_enabled: boolean;
  caveman_enabled: boolean;
  combo_name?: string;
  request?: object;
  response?: object;
}

export interface Quota {
  connection_id: string;
  provider: string;
  connection_name: string;
  account_label?: string; // email or human label, like 9router
  plan?: "free" | "pro" | "ultra" | "enterprise";
  used: number;
  limit: number; // 0 / null → unlimited
  unit?: string;
  reset_at: string;
  is_active: boolean;
  message?: string; // for providers without quota API
  error?: string;
}

export interface ChatMessage {
  role: "user" | "assistant" | "system";
  content: string;
  images?: string[];
}

export interface ChatSession {
  id: string;
  title: string;
  model: string;
  provider: string;
  messages: ChatMessage[];
  created_at: string;
  updated_at: string;
}

export interface ProxyPool {
  id: string;
  name: string;
  protocol: "http" | "https" | "socks5";
  host: string;
  port: number;
  username?: string;
  is_active: boolean;
  last_check_at?: string;
  last_check_status?: string;
}

export interface Tunnel {
  type: "cloudflare" | "tailscale";
  is_enabled: boolean;
  url?: string;
  status: "active" | "inactive" | "error" | "connecting";
}

export interface McpClient {
  ID: string;
  Name: string;
  Transport: "stdio" | "sse" | "streamable-http";
  Command?: string;
  Args?: string[];
  URL?: string;
  Env?: Record<string, string>;
  IsActive: boolean;
  HealthStatus?: string;
  ToolManifest?: { Tools: { Name: string; Description?: string; InputSchema?: unknown }[] };
  CreatedAt?: string;
  UpdatedAt?: string;
}

export interface McpInstance {
  ID: string;
  Name: string;
  Transport: "stdio" | "sse" | "streamable-http";
  Command?: string;
  Args?: string[];
  URL?: string;
  Env?: Record<string, string>;
  IsActive: boolean;
  HealthStatus?: string;
  ToolManifest?: { Tools: { Name: string; Description?: string; InputSchema?: unknown }[] };
  CreatedAt?: string;
  UpdatedAt?: string;
}

export interface McpAccount {
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
}

export interface McpTool {
  type: "function";
  function: {
    name: string;
    description?: string;
    parameters?: unknown;
  };
}

export interface McpToolGroup {
  id: number;
  name: string;
  tool_ids: string[];
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface MitmTool {
  id: string;
  name: string;
  enabled: boolean;
  dns_override: string;
  status: "active" | "inactive";
}

export interface Guardrails {
  guardrails_enabled: boolean;
  guardrails_blocklist: string[];
  pii_redaction_enabled: boolean;
  pii_redaction_types: string[];
}

export interface ModelLimit {
  id: number;
  model: string;
  max_tokens?: number;
  max_rpm?: number;
  allowed_key_ids: string[];
  created_at?: string;
}

export interface PromptTemplate {
  id: number;
  name: string;
  system_prompt: string;
  models: string[];
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface AlertChannel {
  id: number;
  name: string;
  channel_type: "webhook" | "discord" | "telegram" | "email";
  config: Record<string, any>;
  events: string[];
  is_active: boolean;
  created_at?: string;
}

export interface FeatureFlag {
  id: number;
  key: string;
  enabled: boolean;
  description?: string;
  created_at?: string;
}

export interface Settings {
  require_api_key: boolean;
  require_login: boolean;
  rtk_enabled: boolean;
  caveman_enabled: boolean;
  caveman_level: "lite" | "full" | "ultra";
  enable_request_logs: boolean;
  log_retention_days: number;
  cache_enabled: boolean;
  cache_ttl_seconds: number;
  proxy_url?: string;
  notify_webhook_url?: string;
  notify_on_reauth: boolean;
  allowed_sources: string[];
  tunnel_dashboard_access: boolean;
  theme?: "light" | "dark" | "system";
  language?: string;
  inject_errors?: boolean;
}

export interface AuditLog {
  id: string;
  timestamp: string;
  actor: string;
  action: string;
  target: string;
  details?: string;
}

export interface Skill {
  name: string;
  category: "Entry Skills" | "Endpoint Skills" | "Extension Skills";
  description: string;
  url: string;
}

export interface TrafficEvent {
  id: string; // client-generated for React keys
  timestamp: string;
  key_id: string;
  provider: string;
  model: string;
  status_class: string;
  status_code: number;
  latency_ms: number;
}

export interface ConsoleLogEntry {
  timestamp: string;
  level: string;
  message: string;
  attrs?: { key: string; value: string }[];
}
