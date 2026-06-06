# Lovable UI Prompt: g0router Dashboard

## Project Overview

Build a **complete React dashboard UI** for `g0router` — a single-binary Go LLM gateway/proxy with 43+ providers, OAuth flows, MCP gateway, and real-time traffic monitoring. The UI is served as a static SPA embedded in the Go binary.

**Design Reference**: https://github.com/decolua/9router (Next.js dashboard style)
- Dark-first theme with light mode option
- Sidebar navigation with sections
- Card-based layout
- Clean, professional, data-dense

**Tech Stack** (use latest stable versions):
- React 19 + TypeScript
- Vite (bundler)
- Tailwind CSS v4 (or latest Lovable supports)
- shadcn/ui components
- Recharts for charts
- react-hook-form + zod for forms
- i18next + react-i18next (ship `en` + `pt-BR` complete; all other locales fall back to `en`)
- react-use-websocket
- date-fns for date formatting
- nuqs for query param state
- @tanstack/react-table for complex tables

---

## Theme System

### CSS Variables

```css
:root {
  /* Light mode */
  --background: 0 0% 100%;
  --foreground: 222.2 84% 4.9%;
  --card: 0 0% 100%;
  --card-foreground: 222.2 84% 4.9%;
  --popover: 0 0% 100%;
  --popover-foreground: 222.2 84% 4.9%;
  --primary: 222.2 47.4% 11.2%;
  --primary-foreground: 210 40% 98%;
  --secondary: 210 40% 96.1%;
  --secondary-foreground: 222.2 47.4% 11.2%;
  --muted: 210 40% 96.1%;
  --muted-foreground: 215.4 16.3% 46.9%;
  --accent: 210 40% 96.1%;
  --accent-foreground: 222.2 47.4% 11.2%;
  --destructive: 0 84.2% 60.2%;
  --destructive-foreground: 210 40% 98%;
  --border: 214.3 31.8% 91.4%;
  --input: 214.3 31.8% 91.4%;
  --ring: 222.2 84% 4.9%;
  --radius: 0.5rem;
  --success: 142 76% 36%;
  --warning: 38 92% 50%;
  --info: 217 91% 60%;
}

.dark {
  /* Dark mode (default) */
  --background: 222.2 84% 4.9%;
  --foreground: 210 40% 98%;
  --card: 222.2 84% 4.9%;
  --card-foreground: 210 40% 98%;
  --popover: 222.2 84% 4.9%;
  --popover-foreground: 210 40% 98%;
  --primary: 210 40% 98%;
  --primary-foreground: 222.2 47.4% 11.2%;
  --secondary: 217.2 32.6% 17.5%;
  --secondary-foreground: 210 40% 98%;
  --muted: 217.2 32.6% 17.5%;
  --muted-foreground: 215 20.2% 65.1%;
  --accent: 217.2 32.6% 17.5%;
  --accent-foreground: 210 40% 98%;
  --destructive: 0 62.8% 30.6%;
  --destructive-foreground: 210 40% 98%;
  --border: 217.2 32.6% 17.5%;
  --input: 217.2 32.6% 17.5%;
  --ring: 212.7 26.8% 83.9%;
  --success: 142 71% 45%;
  --warning: 38 92% 50%;
  --info: 217 91% 60%;
}
```

### Theme Toggle
- Header button cycles: Light → Dark → System → Light
- System mode respects `prefers-color-scheme`
- Persist preference to `localStorage` key `g0router-theme`

---

## Global Components

### Toast Notification System
- Global toast container (fixed bottom-right)
- 4 types with distinct colors:
  - **Success**: Green background, check icon, auto-dismiss 5s
  - **Error**: Red background, X icon, auto-dismiss 8s
  - **Warning**: Yellow background, alert icon, auto-dismiss 6s
  - **Info**: Blue background, info icon, auto-dismiss 5s
- Manual close button (X) on each toast
- Max 5 toasts visible, queue overflow
- Smooth slide-in/slide-out animation
- Used by: `toast.success("message")`, `toast.error("message")`, etc.

### Sidebar Navigation
- Fixed left sidebar, 240px width
- Collapsible on mobile (overlay drawer)
- Sections with headers:
  - **Gateway**: Dashboard, Providers, Connections, Models, Aliases, Combos
  - **Monitoring**: Usage, Quota, Logs, Traffic, Console
  - **MCP**: MCP Overview, Instances, Accounts, Tools, Tool Groups
  - **Network**: Tunnels, Proxy Pools, MITM
  - **Advanced**: Routing Rules, Virtual Keys, Teams, Guardrails, Model Limits, Prompt Repo, Alert Channels, Feature Flags
  - **System**: Settings, Audit Logs, Diagnostics, Skills
- Active item highlight with left border
- Icons from Lucide React
- Bottom: Theme toggle, Language switcher, User menu

### Header
- Fixed top bar, 64px height
- Left: Page title + breadcrumbs (Home > Providers > OpenAI)
- Right: Global search (Cmd+K), Language switcher, Theme toggle, Notifications bell, User avatar dropdown
- Mobile: Hamburger menu opens sidebar drawer

### Layout
- Two-column: Sidebar (fixed) + Main content (scrollable)
- Main content padding: 24px
- Max-width: none (full width tables/charts)
- Faint grid background pattern (subtle, dark mode only)

### Modals
- Centered overlay with backdrop blur
- Close on Escape key and backdrop click
- Consistent header, body, footer (actions) layout
- Size variants: sm (400px), md (500px), lg (700px), xl (900px)

### Confirm Dialog
- Reusable confirm modal with title, message, Cancel / Confirm actions
- Danger variant: red Confirm button

### Loading States
- Skeleton cards for dashboard metrics
- Skeleton rows for tables
- Spinner for buttons during async actions
- Full-page loader for route transitions

### Empty States
- Icon + title + description + optional action button
- Used for: no data, no connections, no sessions, etc.

### Error States
- Error boundary fallback with error message + retry button
- Inline error messages in forms
- API error toasts

---

## API Client Setup

```typescript
// Base API config
const API_BASE = '';  // Same origin, proxy to /api

// Auth: Session cookie is sent automatically
// No need for Authorization header on /api/* (session-based)
// For inference /v1/*, use API key in Authorization header

interface ApiResponse<T> {
  data: T;
  error?: string;
}

// Generic fetch wrapper with toast error handling
async function apiFetch<T>(path: string, options?: RequestInit): Promise<T>;
```

**Contract rules (IMPORTANT):**
- ALL `/api/*` JSON responses use the `{data, error}` envelope above.
- ALL JSON field names are `snake_case` — the Data Types section below is the
  exact backend contract. Do not invent PascalCase fields or normalizers.
- On `401` from any `/api/*` call: redirect to `/login` (except when
  `require_login` is false per `/api/auth/status`).
- `429` responses include `retry_after_seconds` in the error payload.
- Health endpoint is `GET /healthz` (NOT under `/api`).

---

## Pages

### 0. First-Run Setup Page (`/setup`)
- Shown when `GET /api/auth/status` returns `has_users: false`
- Full-screen centered card (same layout as Login)
- Fields: Username, Display Name (optional), Password, Confirm Password
- Submits `POST /api/auth/setup` — creates first admin user and logs in
- Redirect to `/dashboard` on success
- All routes redirect here while `has_users` is false and `require_login` is true

### 1. Login Page (`/login`)
- Full-screen centered card
- App logo + name
- Username input
- Password input with show/hide toggle
- "Sign In" button with loading state
- Rate-limit message: "Too many attempts. Try again in X minutes."
- No sidebar/header on this page
- Redirect to `/dashboard` on success
- Redirect to login if accessing `/` without session

### 2. Dashboard (`/dashboard`)
- **Metric Cards** (4 in a row):
  - Active Connections (number with green pulse if > 0)
  - Total Requests Today (number with trend arrow)
  - Total Tokens Today (number)
  - Total Cost Today ($ with 2 decimals)
- **Recent Traffic Events** (top half of page):
  - Live SSE stream from `/api/traffic/stream`
  - Table: Time | Key | Provider | Model | Status | Tokens | Cost
  - Auto-updating, latest first
- **Quick Actions** (cards):
  - Add Provider → links to Providers
  - Create API Key → links to API Keys
  - Test Chat → links to Chat Playground
  - View Logs → links to Logs
- **System Status**:
  - Cache enabled/disabled
  - RTK enabled/disabled
  - Caveman mode status
  - Recent health status of connections

### 3. Providers (`/providers`)
- **Grid view**: Cards for each provider
  - Provider icon (fallback to generic)
  - Provider name
  - Auth type badges: OAuth, API Key, Free, Custom
  - Connection count
  - Status: Active (green) / Needs Re-auth (yellow) / Error (red)
  - "Connect" or "Manage" button
- **Filter bar**: Search by name, filter by auth type
- **Batch Test**: "Test All Connections" button with progress
- Click card → navigate to Provider Detail

### 4. Provider Detail (`/providers/:id`)
- **Header**: Provider name + icon + back button
- **Breadcrumbs**: Home > Providers > OpenAI
- **Tabs**:
  - **Overview**: Description, auth types supported, capabilities list
  - **Connections**: Table of connections
    - Name | Auth Type | Status | Models | Proxy | Actions (test, edit, delete)
    - "Add Connection" button → modal with auth type selection
    - OAuth connections: "Connect" button initiates OAuth flow (opens `/api/oauth/:provider/start` in popup)
    - API Key connections: Form with key input + validation
    - Priority reorder: up/down arrows
  - **Models**: Table of models
    - Model ID | Name | Pricing | Status | Actions (test, disable, alias)
    - "Add Custom Model" button
    - "Suggested Models" section (fetched from provider)
    - Toggle to show/hide disabled models
  - **Aliases**: List of aliases for this provider
    - Alias Name → Model mapping
    - CRUD inline

### 5. Connections / Auth (`/connections`)
- Full CRUD table for all connections across all providers
- Filters: Provider, Auth Type, Status
- Columns: Provider | Name | Auth Type | Status | Expires | Last Error | Actions
- Bulk actions: Enable, Disable, Delete selected
- Test button per row

### 6. API Keys (`/keys`)
- **Endpoint Display**:
  - Local endpoint: `http://localhost:PORT/v1` with copy button
  - Tunnel endpoints (if active): Cloudflare URL, Tailscale URL
- **API Keys Table**:
  - Name | Prefix | Scopes | Expiry | RPM | TPM | Daily Cap | Status | Actions
  - Create button → modal with form
  - Delete button → confirm dialog
  - Toggle visibility (show/hide full key)
  - Copy button
- **Virtual Keys Section** (Bifrost-style):
  - Table: Name | Prefix | Budget | Used | Team | Status | Actions
  - Budget progress bar
  - Create/edit modal

### 7. Combos / Routing (`/combos`)
- **Combo List**: Cards or table
  - Name | Strategy | Steps | Status | Actions
  - Strategy badge: Fallback, Round-Robin, Least-Used, Auto, Fastest, Cheapest
- **Combo Detail / Edit**:
  - Name input
  - Strategy selector (dropdown)
  - Steps list: Provider → Model for each step
  - Drag-and-drop reorder (use @dnd-kit)
  - Add step button → model selector modal
  - Remove step button
  - Sticky limit input (if round-robin)
- **Routing Rules Table** (Bifrost):
  - Priority | Condition | Target | Actions
  - Condition examples: "model starts with gpt", "provider = openai"

### 8. Models (`/models`)
- Aggregated model catalog from all providers
- Search by model ID or name
- Filter by provider
- Columns: Model | Provider | Pricing (input/output) | Status | Actions
- Alias creation inline

### 9. Aliases (`/aliases`)
- Table: Alias Name | Provider | Model | Created | Actions
- Create button → modal with provider + model selectors
- Edit inline
- Delete with confirm

### 10. Pricing (`/pricing`)
- Table: Provider | Model | Input Cost | Output Cost | Override | Actions
- Override input fields (per-million tokens)
- Save button per row
- Import/Export buttons

### 11. Usage (`/usage`)
- **Period Selector**: Today / 24h / 7D / 30D / 60D buttons
- **Charts** (Recharts):
  - Requests over time (line chart)
  - Tokens over time (stacked area: input + output)
  - Cost over time (line chart)
- **Summary Cards**: Total Requests | Total Tokens | Total Cost | Avg Latency
- **Usage Table**: Paginated, filterable
  - Time | Provider | Model | Key | Tokens | Cost | Status | Actions (view details)
  - Expand row to see full request/response
- **Export**: CSV download button

### 12. Quota (`/quota`)
- **Auto-refresh**: Toggle with countdown (60s)
- **Filter bar**: Provider dropdown, Status (All/Active/Inactive)
- **Sort**: Expiring First, Remaining Low→High, Remaining High→Low
- **Cards Grid**: Per-connection quota cards
  - Provider name + connection name
  - Progress bar: Used / Limit
  - Reset time
  - Status badge
- **Bulk Actions**:
  - "Disable All Depleted" button
  - "Enable All Available" button
- **Pagination**: Page size selector (10/20/50/100)

### 13. Logs (`/logs`)
- **Request Logs Tab**:
  - Filter: Status class (2xx/4xx/5xx), Provider, Model, Date range, Search
  - Table: Time | Provider | Model | Key | Status | Tokens | Cost | RTK | Caveman | Actions
  - Expand for full request/response JSON
  - Copy JSON button
- **Console Logs Tab**:
  - Live SSE stream from `/api/console-logs/stream`
  - Color-coded levels: LOG (green), INFO (blue), WARN (yellow), ERROR (red), DEBUG (purple)
  - Auto-scroll toggle
  - "Clear Logs" button
  - Search/filter by level

### 14. Traffic (`/traffic`)
- **Live Topology Graph** (SVG-based):
  - Nodes: API Keys on left, Providers on right
  - Animated edges (pulses) showing active requests
  - Edge labels: request count in last 30s
  - Rolling 30s window
  - Pause/resume button
- **Recent Events Table** below graph

### 15. Chat Playground (`/chat`)
- **Session Sidebar** (left, collapsible):
  - "New Chat" button
  - Session list: Title | Model | Time
  - Delete session button (hover)
  - Search sessions
- **Chat Area** (center):
  - Model selector: Provider dropdown → Model dropdown (cascading)
  - Message list:
    - User messages: right-aligned, blue bubble
    - Assistant messages: left-aligned, gray bubble
    - Markdown rendering for assistant
    - Code blocks with syntax highlighting
  - Image attachments: thumbnail gallery above input
  - Input area: textarea with send button, image upload button
  - Stop button (visible while streaming)
  - Streaming indicator (pulsing dot)
- **Empty State**: "Select a model and start chatting"

### 16. MCP Overview (`/mcp`)
- **Stats Cards**: Active Clients | Registered Tools | Instances | Accounts
- **MCP Clients Table**: Name | Transport | Status | Tools Count | Actions
- **Recent MCP Events**: Table of tool executions

### 17. MCP Instances (`/mcp/instances`)
- Table: Name | Command | Type (stdio/SSE/HTTP) | Status | Health | Actions
- Create button → form with command/env vars
- Start/Stop/Restart buttons
- Health status indicator

### 18. MCP Accounts (`/mcp/accounts`)
- Table: Account | Provider | Status | Linked To | Actions
- OAuth connect buttons
- Disconnect button

### 19. MCP Tools (`/mcp/tools`)
- Table: Tool Name | Client | Description | Parameters | Actions
- Group by client (accordion)
- "Execute" button → opens execution modal with JSON params input
- Tool Groups section: assign tools to groups

### 20. MCP Tool Groups (`/mcp/tool-groups`)
- Table: Group Name | Tools Count | Actions
- Create/edit/delete
- Assign tools to group (multi-select)

### 21. Tunnels (`/tunnels`)
- **Cloudflare Card**:
  - Status: Inactive / Active (green)
  - URL (if active)
  - Start/Stop button
  - Health indicator
- **Tailscale Card**:
  - Same structure
- **Endpoint Display**: Local + tunnel URLs with copy buttons

### 22. Proxy Pools (`/proxy-pools`)
- Table: Name | Protocol | Host:Port | Status | Last Check | Actions
- Create button → form
- Test button per row
- Bulk import: textarea for proxy list (host:port or URL format)
- Bulk actions: Activate/Deactivate/Delete selected

### 23. MITM Proxy (`/mitm`)
- **Status Card**: Active/Inactive toggle
- **CA Certificate**: Download button + install instructions
- **Tool Cards** (grid):
  - Antigravity, GitHub Copilot, Cursor, Kiro
  - Each: Enable toggle, DNS override input, status
- **Instructions**: How to install CA cert per OS

### 24. Routing Rules (`/routing-rules`)
- Table: Priority | Name | Condition | Target Provider | Target Model | Actions
- Condition builder: Field (model/provider/header) | Operator (equals/contains/starts with) | Value
- Drag to reorder priority
- Enable/disable toggle per rule

### 25. Virtual Keys (`/virtual-keys`)
- Table: Name | Prefix | Budget | Used | Rate Limit | Team | Status | Actions
- Budget progress bar
- Create/edit modal: name, budget, period, RPM, TPM, team
- Filter by team

### 26. Teams (`/teams`)
- Table: Name | Budget | Keys Count | Members | Actions
- Create/edit modal
- View team members (future)

### 27. Guardrails (`/guardrails`)
- **Enable Toggle**
- **Blocklist**: Textarea with blocked terms (one per line)
- **PII Redaction Toggle**
- **PII Types**: Checkboxes for Email, Phone, SSN, Credit Card, IP Address
- **Test Area**: Input prompt → see filtered output

### 28. Model Limits (`/model-limits`)
- Table: Model | Max Tokens | Max Requests/Min | Allowed Keys | Time Window | Actions
- Create/edit modal

### 29. Prompt Repository (`/prompts`)
- Table: Name | Description | Variables | Actions
- Create/edit modal:
  - Name, Description
  - System prompt textarea
  - User prompt template textarea (with {{variable}} syntax)
  - Variables list (auto-detected from template)
- Preview panel: fill variables → see rendered prompt

### 30. Alert Channels (`/alerts`)
- Table: Name | Type | Target | Status | Actions
- Create modal:
  - Type: Webhook, Discord, Telegram, Email
  - Config per type (URL, token, etc.)
  - Event triggers: Quota Depleted, Connection Stale, Rate Limit Hit
- Test button: send test alert

### 31. Feature Flags (`/feature-flags`)
- Admin-only page
- Table: Key | Enabled | Description | Actions
- Toggle per flag
- Flags: adaptive_routing, semantic_cache, websocket_chat, guardrails, pii_redaction, etc.

### 32. Settings (`/settings`)
- **Tabs**:
  - **General**: Theme, Language, Log retention days, Cache TTL
  - **Security**: Require login toggle, Require API key toggle, Source IP policy (local/lan/tailscale/public), JWT secret
  - **Routing**: Global fallback strategy, Combo strategy, Sticky limits
  - **Proxy**: Outbound proxy URL, Test proxy button
  - **Notifications**: Webhook URL, Discord/Telegram tokens, Enable/disable alerts
  - **OIDC**: Issuer URL, Client ID, Client Secret, Scopes, Login label, Test connection
  - **Backup/Restore**: Export DB as JSON, Import from JSON file
  - **Advanced**: RTK toggle, Caveman mode (off/lite/full/ultra), Observability toggle, Tunnel dashboard access

### 33. Audit Logs (`/audit`)
- Table: Time | Actor | Action | Target | Details
- Filter: Action type, Actor, Date range
- Pagination

### 34. Diagnostics (`/diagnostics`)
- **Health Checks**:
  - Database connection
  - Inference engine
  - MCP manager
  - Cache status
- **Release Readiness**: Checklist of requirements
- **Version Info**: Build version, Go version, UI version
- **System Resources**: Memory usage, goroutine count

### 35. Skills (`/skills`)
- Grid of skill cards
- Categories: Entry Skills, Endpoint Skills, Extension Skills
- Each: Name, Description, Copy URL button (raw GitHub URL)
- Link to skills GitHub repo

### 36. Landing Page (`/landing`)
- **Hero Section**: Large headline, subheadline, CTA button ("Open Dashboard")
- **Features Grid**: 6 feature cards with icons
  - 43+ Providers, OAuth & API Keys, Real-time Monitoring, MCP Gateway, Routing Strategies, Token Compression
- **How It Works**: 3-step visual
- **Footer**: Links to docs, GitHub, version
- No sidebar/header — full-page standalone

---

## API Endpoints Reference

### Auth
- `GET /api/auth/status` → `{require_login, has_users, authenticated, username, display_name, role}` (public)
- `POST /api/auth/setup` — `{username, password, display_name?}` first-run only (409 after)
- `POST /api/auth/login` — `{username, password}` → set session cookie; 429 + `retry_after_seconds` when rate limited
- `POST /api/auth/logout` — clear session cookie
- `PUT /api/auth/password` — `{current_password, new_password}`
- `GET /api/auth/users` — list dashboard users (admin only)
- `POST /api/auth/users` — create user (admin only)
- `DELETE /api/auth/users/:id` — delete user (admin only)

### Providers
- `GET /api/providers` — list all providers
- `GET /api/providers/:id` — provider details
- `GET /api/providers/:id/models` — list models for provider
- `GET /api/providers/:id/connections` — list connections
- `POST /api/providers/:id/models/:model/test` — test model
- `POST /api/providers/test-batch` — batch test all
- `GET /api/providers/:id/suggested-models` — fetch suggested models

### Connections
- `GET /api/connections` — list all
- `POST /api/connections` — create
- `PUT /api/connections/:id` — update
- `DELETE /api/connections/:id` — delete
- `POST /api/connections/:id/test` — test connection
- `PUT /api/connections/:id/proxy` — assign proxy
- `POST /api/connections/bulk-disable` — disable depleted
- `POST /api/connections/bulk-enable` — enable available

### API Keys
- `GET /api/keys` — list
- `POST /api/keys` — create
- `DELETE /api/keys/:id` — delete

### Virtual Keys (Bifrost)
- `GET /api/virtual-keys` — list
- `POST /api/virtual-keys` — create
- `PUT /api/virtual-keys/:id` — update
- `DELETE /api/virtual-keys/:id` — delete

### Teams
- `GET /api/teams` — list
- `POST /api/teams` — create
- `PUT /api/teams/:id` — update
- `DELETE /api/teams/:id` — delete

### Combos / Routing
- `GET /api/combos` — list
- `POST /api/combos` — create
- `PUT /api/combos/:id` — update
- `DELETE /api/combos/:id` — delete
- `GET /api/routing-rules` — list rules
- `POST /api/routing-rules` — create rule
- `PUT /api/routing-rules/:id` — update
- `DELETE /api/routing-rules/:id` — delete

### Aliases
- `GET /api/aliases` — list
- `POST /api/aliases` — create
- `PUT /api/aliases/:id` — update
- `DELETE /api/aliases/:id` — delete

### Pricing
- `GET /api/pricing` — list
- `POST /api/pricing` — create override
- `PUT /api/pricing/:id` — update
- `DELETE /api/pricing/:id` — delete

### Usage
- `GET /api/usage/summary?period=today` — aggregated stats
- `GET /api/usage?limit=50&offset=0&period=7d` — paginated logs
- `GET /api/usage/chart?period=7d&granularity=day` — time-series buckets:
  `{buckets: [], requests: [], tokens_input: [], tokens_output: [], costs: []}`
  (periods: today|24h|7d|30d|60d; granularity hour|day)
- `GET /api/logs/:id` — single log detail

### Quota
- `GET /api/quota` — list all quotas

### Models
- `GET /api/models` — list all models
- `GET /api/models/disabled` — list disabled
- `POST /api/models/disabled` — `{provider, model}` disable
- `DELETE /api/models/disabled` — `{provider, model}` in body (composite key) — re-enable
- `POST /api/models/custom` — `{provider, model, display_name?}` add custom model
- `DELETE /api/models/custom/:id` — remove custom model

### Chat Sessions
- `GET /api/chat-sessions` — list (metadata only, no messages)
- `GET /api/chat-sessions/:id` — full session incl. messages
- `POST /api/chat-sessions` — create `{title?, model, provider}`
- `PUT /api/chat-sessions/:id` — update (title/messages)
- `DELETE /api/chat-sessions/:id` — delete
- Images: NO upload endpoint — client converts files to base64 data URLs and
  embeds them in message `images` arrays (max 4 per message, 5 MB each,
  png/jpeg/webp/gif only)

### Console Logs
- `GET /api/console-logs/stream` — SSE stream
- `DELETE /api/console-logs` — clear

### Traffic
- `GET /api/traffic/stream` — SSE stream

### MCP
- `GET /api/mcp/clients` — list clients
- `GET /api/mcp/instances` — list instances
- `POST /api/mcp/instances` — create
- `PUT /api/mcp/instances/:id` — update
- `DELETE /api/mcp/instances/:id` — delete
- `GET /api/mcp/accounts` — list accounts
- `GET /api/mcp/tools` — list tools
- `POST /api/mcp/tools/:name/execute` — execute tool
- `GET /api/mcp/tool-groups` — list groups
- `POST /api/mcp/tool-groups` — create
- `PUT /api/mcp/tool-groups/:id` — update
- `DELETE /api/mcp/tool-groups/:id` — delete

### Tunnels
- `GET /api/tunnels` — list status
- `POST /api/tunnels/cloudflare` — create/start
- `DELETE /api/tunnels/cloudflare` — stop
- `POST /api/tunnels/tailscale` — create/start
- `DELETE /api/tunnels/tailscale` — stop
- `GET /api/tunnels/health` — health check

### Proxy Pools
- `GET /api/proxy-pools` — list
- `POST /api/proxy-pools` — create
- `PUT /api/proxy-pools/:id` — update
- `DELETE /api/proxy-pools/:id` — delete
- `POST /api/proxy-pools/:id/test` — health check
- `POST /api/proxy-pools/batch` — batch import

### MITM
- `GET /api/mitm/status` — get status
- `POST /api/mitm/toggle` — enable/disable
- `GET /api/mitm/ca-cert` — download CA cert
- `PUT /api/mitm/tools/:tool` — update tool config

### Semantic Cache
- `GET /api/cache/semantic` — stats + entry list (key, model, hits, expires)
- `DELETE /api/cache/semantic` — clear

### Guardrails
- `GET /api/guardrails` — get config
- `PUT /api/guardrails` — update config
- `POST /api/guardrails/test` — test prompt filtering

### Model Limits
- `GET /api/model-limits` — list
- `POST /api/model-limits` — create
- `PUT /api/model-limits/:id` — update
- `DELETE /api/model-limits/:id` — delete

### Prompts
- `GET /api/prompt-templates` — list
- `POST /api/prompt-templates` — create
- `PUT /api/prompt-templates/:id` — update
- `DELETE /api/prompt-templates/:id` — delete

### Alerts
- `GET /api/alert-channels` — list
- `POST /api/alert-channels` — create
- `PUT /api/alert-channels/:id` — update
- `DELETE /api/alert-channels/:id` — delete
- `POST /api/alert-channels/:id/test` — test

### Feature Flags
- `GET /api/feature-flags` — list (flags are seeded by backend; no user creation)
- `PUT /api/feature-flags/:id` — toggle enabled

### Settings
- `GET /api/settings` — get all settings
- `PUT /api/settings` — update settings
- `POST /api/settings/backup` — export DB
- `POST /api/settings/restore` — import DB
- `POST /api/settings/proxy-test` — test proxy

### Audit
- `GET /api/audit` — list audit logs

### Diagnostics
- `GET /healthz` — health check (NOT under /api)
- `GET /api/diagnostics` — full diagnostics

### Skills
- `GET /api/skills` — `[{name, category, description, url}]` static catalog

### WebSocket
- `GET /api/ws` — WebSocket upgrade (chat streaming protocol; auth via session cookie before upgrade)
  - Client sends `{type:"chat", session_id, model, messages}`
  - Server streams `{type:"delta", content}` … `{type:"done", usage}` or `{type:"error", error}`

### Locale
- `GET /api/locale` — get locale
- `POST /api/locale` — set locale

### Update
- `GET /api/version` — `{version, go_version, build_date}`
- `POST /api/update/check` — `{current, latest, update_available, changelog_url}`
- `POST /api/update/apply` — download + stage update (admin only; applies on restart)

### Inference (OpenAI-compatible)
- `POST /v1/chat/completions` — chat with streaming support
- `GET /v1/models` — list models

---

## Data Types

```typescript
// Provider
interface Provider {
  id: string;
  name: string;
  display_name: string;
  auth_types: ('oauth' | 'api_key' | 'noauth' | 'custom')[];
  capabilities: string[];
  icon_url?: string;
  connection_count: number;
  status: 'active' | 'inactive' | 'error' | 'needs_reauth';
}

// Connection
interface Connection {
  id: string;
  provider: string;
  name: string;
  auth_type: 'oauth' | 'api_key' | 'noauth';
  is_active: boolean;
  models: string[];
  proxy_id?: string;
  priority: number;
  last_error?: string;
  needs_reauth: boolean;
  unavailable_until?: string;
}

// API Key
interface APIKey {
  id: string;
  name: string;
  prefix: string;
  scopes: string[];
  expires_at?: string;
  rpm_limit?: number;
  tpm_limit?: number;
  daily_spend_cap?: number;
  is_active: boolean;
}

// Virtual Key (Bifrost)
interface VirtualKey {
  id: string;
  name: string;
  prefix: string;
  budget_usd?: number;
  budget_used_usd: number;
  budget_period: 'daily' | 'weekly' | 'monthly';
  rate_limit_rpm?: number;
  rate_limit_tpm?: number;
  team_id?: string;
  is_active: boolean;
}

// Combo / Route
interface Combo {
  id: string;
  name: string;
  strategy: 'fallback' | 'round_robin' | 'least_used' | 'auto' | 'fastest' | 'cheapest';
  steps: ComboStep[];
  sticky_limit?: number;
  is_active: boolean;
}

interface ComboStep {
  provider: string;
  model: string;
}

// Routing Rule
interface RoutingRule {
  id: string;
  name: string;
  priority: number;
  condition: {
    field: 'model' | 'provider' | 'header';
    operator: 'equals' | 'contains' | 'starts_with';
    value: string;
  };
  target_provider: string;
  target_model?: string;
  is_active: boolean;
}

// Usage Log
interface UsageLog {
  id: string;
  timestamp: string;
  provider: string;
  model: string;
  api_key_id: string;
  status: 'success' | 'error';
  status_code: number;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  cost_usd: number;
  latency_ms: number;
  rtk_enabled: boolean;
  caveman_enabled: boolean;
  combo_name?: string;
}

// Quota
interface Quota {
  connection_id: string;
  provider: string;
  connection_name: string;
  used: number;
  limit: number;
  reset_at: string;
  is_active: boolean;
}

// Chat Session
interface ChatSession {
  id: string;
  title: string;
  model: string;
  provider: string;
  messages: ChatMessage[];
  created_at: string;
  updated_at: string;
}

interface ChatMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
  images?: string[]; // base64 data URLs
}

// Proxy Pool
interface ProxyPool {
  id: string;
  name: string;
  protocol: 'http' | 'https' | 'socks5';
  host: string;
  port: number;
  username?: string;
  is_active: boolean;
  last_check_at?: string;
  last_check_status?: string;
}

// Tunnel
interface Tunnel {
  type: 'cloudflare' | 'tailscale';
  is_enabled: boolean;
  url?: string;
  status: 'active' | 'inactive' | 'error';
}

// MCP Instance
interface MCPInstance {
  id: string;
  name: string;
  command: string;
  type: 'stdio' | 'sse' | 'http';
  status: 'running' | 'stopped' | 'error';
  health: 'healthy' | 'unhealthy';
}

// MCP Tool
interface MCPTool {
  name: string;
  client: string;
  description: string;
  parameters: object;
}

// Alert Channel
interface AlertChannel {
  id: string;
  name: string;
  channel_type: 'webhook' | 'discord' | 'telegram' | 'email';
  config: object;
  is_active: boolean;
}

// Prompt Template
interface PromptTemplate {
  id: string;
  name: string;
  description?: string;
  system_prompt?: string;
  user_prompt_template: string;
  variables: string[];
}

// Feature Flag
interface FeatureFlag {
  id: string;
  key: string;
  enabled: boolean;
  description?: string;
}

// Settings
interface Settings {
  require_api_key: boolean;
  require_login: boolean;
  rtk_enabled: boolean;
  caveman_enabled: boolean;
  caveman_level: 'lite' | 'full' | 'ultra';
  enable_request_logs: boolean;
  log_retention_days: number;
  cache_enabled: boolean;
  cache_ttl_seconds: number;
  proxy_url?: string;
  notify_webhook_url?: string;
  notify_on_reauth: boolean;
  allowed_sources: string[];
  tunnel_dashboard_access: boolean;
  theme?: 'light' | 'dark' | 'system';
  language?: string;
}

// Audit Log
interface AuditLog {
  id: string;
  timestamp: string;
  actor: string;
  action: string;
  target: string;
  details?: string;
}
```

---

## Responsive Breakpoints

- Mobile: < 768px — sidebar becomes drawer, tables stack, charts scroll
- Tablet: 768px - 1024px — sidebar collapses to icons-only
- Desktop: > 1024px — full sidebar, all features visible

---

## Accessibility

- All interactive elements keyboard accessible
- ARIA labels on icons and buttons
- Focus visible outlines
- Color contrast WCAG 2.1 AA compliant
- Screen reader friendly tables with proper headers

---

## Performance Targets

- First Contentful Paint < 1.5s
- Time to Interactive < 3s
- Bundle size < 500KB gzipped
- Lazy load heavy pages (charts, editor)
- Virtualize long lists (logs, sessions)
