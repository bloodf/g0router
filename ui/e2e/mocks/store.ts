// In-memory store for mock API layer. Each test gets a fresh instance.

import type {
  AuthStatus,
  User,
  Provider,
  Connection,
  ApiKey,
  VirtualKey,
  Team,
  Combo,
  RoutingRule,
  Model,
  Alias,
  PricingOverride,
  UsageLog,
  Quota,
  ChatSession,
  Tunnel,
  Settings,
  AuditLog,
  TrafficEvent,
  ConsoleLogEntry,
  Guardrails,
  ModelLimit,
  PromptTemplate,
  AlertChannel,
  FeatureFlag,
  McpClient,
  McpInstance,
  McpTool,
  McpToolGroup,
  ProxyPool,
  MitmTool,
  Skill,
} from "../../src/lib/types";
import {
  seedUsers,
  seedProviders,
  seedModels,
  seedTunnels,
  seedConnections,
  seedKeys,
  seedVirtualKeys,
  seedTeams,
  seedCombos,
  seedAliases,
  seedPricing,
  seedRoutingRules,
  seedUsageLogs,
  seedAuditLogs,
  seedQuota,
  seedChatSessions,
  seedGuardrails,
  seedPromptTemplates,
  seedAlertChannels,
  seedFeatureFlags,
  seedMcpClients,
  seedMcpInstances,
  seedMcpTools,
  seedMcpToolGroups,
  seedProxyPools,
  seedMitmStatus,
  seedSkills,
  seedConsoleLogs,
} from "./seed";

let idCounter = 0;
function nextId(): string {
  return `mock-${++idCounter}-${Date.now().toString(36)}`;
}

export class MockStore {
  auth: AuthStatus = {
    require_login: true,
    has_users: true,
    authenticated: false,
    username: "admin",
    display_name: "Administrator",
    role: "admin",
  };

  users: User[] = [];
  settings: Settings = {} as Settings;
  providers = new Map<string, Provider>();
  connections = new Map<string, Connection>();
  keys = new Map<string, ApiKey>();
  virtualKeys = new Map<string, VirtualKey>();
  models = new Map<string, Model>();
  combos = new Map<string, Combo>();
  aliases = new Map<string, Alias>();
  pricing = new Map<string, PricingOverride>();
  routingRules = new Map<string, RoutingRule>();
  teams = new Map<string, Team>();
  tunnels = new Map<string, Tunnel>();
  auditLogs: AuditLog[] = [];
  usageLogs: UsageLog[] = [];
  quotas: Quota[] = [];
  chatSessions: ChatSession[] = [];
  trafficEvents: TrafficEvent[] = [];
  consoleLogs: ConsoleLogEntry[] = [];

  disabledModels = new Set<string>();
  customModels: Model[] = [];
  modelLimits = new Map<string, ModelLimit>();
  guardrails: Guardrails = { guardrails_enabled: false, guardrails_blocklist: [], pii_redaction_enabled: false, pii_redaction_types: [] };
  promptTemplates = new Map<string, PromptTemplate>();
  alertChannels = new Map<string, AlertChannel>();
  featureFlags = new Map<string, FeatureFlag>();
  mcpClients = new Map<string, McpClient>();
  mcpInstances = new Map<string, McpInstance>();
  mcpTools: McpTool[] = [];
  mcpToolGroups = new Map<string, McpToolGroup>();
  proxyPools = new Map<string, ProxyPool>();
  mitmEnabled = false;
  mitmTools: MitmTool[] = [];
  skills: Skill[] = [];
  locale = { current: "en", available: ["en", "pt"] };

  nextId = nextId;

  reset() {
    this.auth = {
      require_login: true,
      has_users: true,
      authenticated: false,
      username: "admin",
      display_name: "Administrator",
      role: "admin",
    };
    this.users = [];
    this.settings = {} as Settings;
    this.providers.clear();
    this.connections.clear();
    this.keys.clear();
    this.virtualKeys.clear();
    this.models.clear();
    this.combos.clear();
    this.aliases.clear();
    this.pricing.clear();
    this.routingRules.clear();
    this.teams.clear();
    this.tunnels.clear();
    this.auditLogs = [];
    this.usageLogs = [];
    this.quotas = [];
    this.chatSessions = [];
    this.trafficEvents = [];
    this.consoleLogs = [];

    this.disabledModels.clear();
    this.customModels = [];
    this.modelLimits.clear();
    this.guardrails = { guardrails_enabled: false, guardrails_blocklist: [], pii_redaction_enabled: false, pii_redaction_types: [] };
    this.promptTemplates.clear();
    this.alertChannels.clear();
    this.featureFlags.clear();
    this.mcpClients.clear();
    this.mcpInstances.clear();
    this.mcpTools = [];
    this.mcpToolGroups.clear();
    this.proxyPools.clear();
    this.mitmEnabled = false;
    this.mitmTools = [];
    this.skills = [];
    this.locale = { current: "en", available: ["en", "pt"] };
  }

  seedAll() {
    this.reset();
    this.users = seedUsers();
    this.settings = {
      require_api_key: false,
      require_login: true,
      rtk_enabled: false,
      caveman_enabled: false,
      caveman_level: "lite",
      enable_request_logs: true,
      log_retention_days: 30,
      cache_enabled: false,
      cache_ttl_seconds: 3600,
      proxy_url: "",
      notify_webhook_url: "",
      notify_on_reauth: false,
      allowed_sources: ["local", "lan"],
      tunnel_dashboard_access: false,
      theme: "system",
      language: "en",
    };
    for (const p of seedProviders()) this.providers.set(p.id, p);
    for (const m of seedModels()) this.models.set(m.id, m);
    for (const t of seedTunnels()) this.tunnels.set(t.type, t);
    for (const c of seedConnections()) this.connections.set(c.id, c);
    for (const k of seedKeys()) this.keys.set(k.id, k);
    for (const vk of seedVirtualKeys()) this.virtualKeys.set(vk.id, vk);
    for (const t of seedTeams()) this.teams.set(t.id, t);
    for (const c of seedCombos()) this.combos.set(c.id, c);
    for (const a of seedAliases()) this.aliases.set(a.id, a);
    for (const p of seedPricing()) this.pricing.set(p.id, p);
    for (const r of seedRoutingRules()) this.routingRules.set(r.id, r);
    this.usageLogs = seedUsageLogs();
    this.auditLogs = seedAuditLogs();
    this.quotas = seedQuota();
    this.chatSessions = seedChatSessions();
    this.guardrails = seedGuardrails();
    for (const p of seedPromptTemplates()) this.promptTemplates.set(String(p.id), p);
    for (const a of seedAlertChannels()) this.alertChannels.set(String(a.id), a);
    for (const f of seedFeatureFlags()) this.featureFlags.set(String(f.id), f);
    for (const c of seedMcpClients()) this.mcpClients.set(c.ID, c);
    for (const i of seedMcpInstances()) this.mcpInstances.set(i.ID, i);
    this.mcpTools = seedMcpTools();
    for (const g of seedMcpToolGroups()) this.mcpToolGroups.set(String(g.id), g);
    for (const p of seedProxyPools()) this.proxyPools.set(p.id, p);
    const mitm = seedMitmStatus();
    this.mitmEnabled = mitm.enabled;
    this.mitmTools = mitm.tools;
    this.skills = seedSkills();
    this.consoleLogs = seedConsoleLogs();
  }
}
