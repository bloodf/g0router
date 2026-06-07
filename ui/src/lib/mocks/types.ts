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
  password: string; // hashed in real life; plaintext for mock
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
  keys_count: number;
  members: number;
}

export interface ComboStep {
  provider: string;
  model: string;
}

export interface Combo {
  id: string;
  name: string;
  strategy:
    | "fallback"
    | "round_robin"
    | "least_used"
    | "auto"
    | "fastest"
    | "cheapest";
  steps: ComboStep[];
  sticky_limit?: number;
  is_active: boolean;
}

export interface RoutingRule {
  id: string;
  name: string;
  priority: number;
  condition: {
    field: "model" | "provider" | "header";
    operator: "equals" | "contains" | "starts_with";
    value: string;
  };
  target_provider: string;
  target_model?: string;
  is_active: boolean;
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

export interface McpInstance {
  id: string;
  name: string;
  command: string;
  type: "stdio" | "sse" | "http";
  status: "running" | "stopped" | "error";
  health: "healthy" | "unhealthy";
  tools_count: number;
}

export interface McpAccount {
  id: string;
  account: string;
  provider: string;
  status: "linked" | "unlinked" | "error";
  linked_to?: string;
}

export interface McpTool {
  name: string;
  client: string;
  description: string;
  parameters: object;
}

export interface McpToolGroup {
  id: string;
  name: string;
  tools: string[];
}

export interface MitmTool {
  id: string;
  name: string;
  enabled: boolean;
  dns_override: string;
  status: "active" | "inactive";
}

export interface Guardrails {
  enabled: boolean;
  blocklist: string[];
  pii_redaction: boolean;
  pii_types: string[];
}

export interface ModelLimit {
  id: string;
  model: string;
  max_tokens?: number;
  max_requests_per_min?: number;
  allowed_keys: string[];
  time_window_seconds?: number;
}

export interface PromptTemplate {
  id: string;
  name: string;
  description?: string;
  system_prompt?: string;
  user_prompt_template: string;
  variables: string[];
}

export interface AlertChannel {
  id: string;
  name: string;
  channel_type: "webhook" | "discord" | "telegram" | "email";
  config: Record<string, string>;
  is_active: boolean;
}

export interface FeatureFlag {
  id: string;
  key: string;
  enabled: boolean;
  description?: string;
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
  // mock-only
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
  id: string;
  timestamp: string;
  api_key_id: string;
  api_key_name: string;
  provider: string;
  model: string;
  combo_id?: string;
  status: "success" | "error";
  tokens: number;
  latency_ms: number;
  cost_usd: number;
}

export interface ConsoleLogEntry {
  id: string;
  timestamp: string;
  level: "LOG" | "INFO" | "WARN" | "ERROR" | "DEBUG";
  message: string;
}
