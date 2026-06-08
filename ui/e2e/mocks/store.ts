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
} from "../../src/lib/types";
import {
  seedUsers,
  seedProviders,
  seedModels,
  seedSettings,
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
  }

  seedAll() {
    this.reset();
    this.users = seedUsers();
    this.settings = seedSettings();
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
    this.consoleLogs = seedConsoleLogs();
  }
}
