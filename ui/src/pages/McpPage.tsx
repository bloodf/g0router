import type { FormEvent, ReactNode } from "react";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ApiError,
  apiFetch,
  asyncError,
  getMcpServersPath,
  listMCPAccounts,
  listMCPClients,
  listMCPInstances,
  listMCPTools
} from "../api";
import type {
  AsyncState,
  MCPOAuthAccountResponse,
  MCPClientResponse,
  MCPInstanceResponse,
  MCPToolResponse
} from "../api";
import { EmptyState, ErrorState, LoadingState, Panel, StatusPill } from "../components/Primitives";

type MCPData = {
  accountsByInstance: Record<string, MCPOAuthAccountResponse[]>;
  clients: MCPClientResponse[];
  instances: MCPInstanceResponse[];
  tools: MCPToolResponse[];
};

type InstanceForm = {
  accountLabel: string;
  command: string;
  isActive: boolean;
  launchType: "command" | "npx" | "docker" | "http";
  name: string;
  serverKey: string;
  transport: "stdio" | "sse" | "streamable-http";
  url: string;
};

type OAuthForm = {
  authorizationURL: string;
  instanceID: string;
  redirectURI: string;
  resourceURI: string;
};

type OAuthStartResponse = {
  authorization_url: string;
  expires_at: string;
};

const emptyData: MCPData = {
  accountsByInstance: {},
  clients: [],
  instances: [],
  tools: []
};

const defaultInstanceForm: InstanceForm = {
  accountLabel: "",
  command: "",
  isActive: true,
  launchType: "http",
  name: "",
  serverKey: "",
  transport: "streamable-http",
  url: ""
};

export function McpPage() {
  const [state, setState] = useState<AsyncState<MCPData>>({ status: "loading" });
  const [instanceForm, setInstanceForm] = useState<InstanceForm>(defaultInstanceForm);
  const [oauthForm, setOAuthForm] = useState<OAuthForm>({
    authorizationURL: "",
    instanceID: "",
    redirectURI: defaultRedirectURI(),
    resourceURI: ""
  });
  const [createError, setCreateError] = useState("");
  const [oauthError, setOAuthError] = useState("");
  const [startedAuthURL, setStartedAuthURL] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [isStartingOAuth, setIsStartingOAuth] = useState(false);

  const loadMCPData = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const [clients, instances, tools] = await Promise.all([listMCPClients(), listMCPInstances(), listMCPTools()]);
      const accountEntries = await Promise.all(
        instances.map(async (instance) => [instance.ID, await listMCPAccounts(instance.ID)] as const)
      );
      const data = {
        accountsByInstance: Object.fromEntries(accountEntries),
        clients,
        instances,
        tools
      };
      setState(isMCPDataEmpty(data) ? { status: "empty", data } : { status: "success", data });
    } catch (error) {
      setState(asyncError<MCPData>(toApiError(error)));
    }
  }, []);

  useEffect(() => {
    void loadMCPData();
  }, [loadMCPData]);

  const data = state.status === "success" || state.status === "empty" ? state.data : emptyData;
  const totalAccounts = useMemo(
    () => Object.values(data.accountsByInstance).reduce((count, accounts) => count + accounts.length, 0),
    [data.accountsByInstance]
  );

  async function handleCreateInstance(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreateError("");
    setIsCreating(true);

    try {
      await apiFetch<MCPInstanceResponse>(getMcpServersPath(), {
        method: "POST",
        body: {
          account_label: blankToUndefined(instanceForm.accountLabel),
          command: blankToUndefined(instanceForm.command),
          is_active: instanceForm.isActive,
          launch_type: instanceForm.launchType,
          name: instanceForm.name.trim(),
          server_key: instanceForm.serverKey.trim(),
          transport: instanceForm.transport,
          url: blankToUndefined(instanceForm.url)
        }
      });
      setInstanceForm(defaultInstanceForm);
      await loadMCPData();
    } catch (error) {
      setCreateError(redactErrorMessage(toApiError(error).message));
    } finally {
      setIsCreating(false);
    }
  }

  async function handleStartOAuth(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setOAuthError("");
    setStartedAuthURL("");

    const instanceID = oauthForm.instanceID || data.instances[0]?.ID || "";
    if (instanceID === "") {
      setOAuthError("Select an MCP instance before starting OAuth.");
      return;
    }

    setIsStartingOAuth(true);
    try {
      const response = await apiFetch<OAuthStartResponse>(`${getMcpServersPath()}/${encodeURIComponent(instanceID)}/auth/start`, {
        method: "POST",
        body: {
          authorization_url: oauthForm.authorizationURL.trim(),
          redirect_uri: oauthForm.redirectURI.trim(),
          resource_uri: oauthForm.resourceURI.trim()
        }
      });
      setOAuthForm((current) => ({
        ...current,
        authorizationURL: "",
        instanceID,
        resourceURI: ""
      }));
      setStartedAuthURL(response.authorization_url);
    } catch (error) {
      setOAuthError(redactErrorMessage(toApiError(error).message));
    } finally {
      setIsStartingOAuth(false);
    }
  }

  return (
    <Panel title="MCP gateway" description="Configured MCP instances, accounts, health, and compact tool manifests.">
      <div className="space-y-5">
        {state.status === "loading" ? <LoadingState label="Loading MCP data" /> : null}
        {state.status === "auth-expired" ? (
          <ErrorState title="MCP session expired" message={redactErrorMessage(state.error.message)} onRetry={loadMCPData} />
        ) : null}
        {state.status === "error" ? (
          <ErrorState title="Could not load MCP gateway" message={redactErrorMessage(state.error.message)} onRetry={loadMCPData} />
        ) : null}
        {state.status === "success" || state.status === "empty" ? (
          <>
            <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
              <InstanceFormView
                error={createError}
                form={instanceForm}
                isSubmitting={isCreating}
                onChange={setInstanceForm}
                onSubmit={handleCreateInstance}
              />
              <OAuthFormView
                authURL={startedAuthURL}
                error={oauthError}
                form={oauthForm}
                instances={data.instances}
                isSubmitting={isStartingOAuth}
                onChange={setOAuthForm}
                onSubmit={handleStartOAuth}
              />
            </div>

            {state.status === "empty" ? (
              <EmptyState title="No MCP data" description="Create an instance or register a client to expose tools." />
            ) : (
              <MCPDashboard data={data} totalAccounts={totalAccounts} />
            )}
          </>
        ) : null}
      </div>
    </Panel>
  );
}

function InstanceFormView({
  error,
  form,
  isSubmitting,
  onChange,
  onSubmit
}: {
  error: string;
  form: InstanceForm;
  isSubmitting: boolean;
  onChange: (form: InstanceForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <form className="rounded-md border border-zinc-200 p-4" onSubmit={onSubmit}>
      <div className="mb-3 flex items-center justify-between gap-3">
        <h4 className="text-sm font-semibold text-zinc-700">Create instance</h4>
        <label className="inline-flex items-center gap-2 text-xs font-semibold text-zinc-600">
          <input
            checked={form.isActive}
            className="h-4 w-4 accent-zinc-950"
            type="checkbox"
            onChange={(event) => onChange({ ...form, isActive: event.target.checked })}
          />
          Active
        </label>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <TextField
          label="Instance name"
          required
          value={form.name}
          onChange={(name) => onChange({ ...form, name })}
        />
        <TextField
          label="Server key"
          required
          value={form.serverKey}
          onChange={(serverKey) => onChange({ ...form, serverKey })}
        />
        <SelectField
          label="Launch type"
          value={form.launchType}
          options={["http", "command", "npx", "docker"]}
          onChange={(launchType) =>
            onChange({
              ...form,
              launchType: launchType as InstanceForm["launchType"],
              transport: launchType === "http" ? "streamable-http" : "stdio"
            })
          }
        />
        <SelectField
          label="Transport"
          value={form.transport}
          options={form.launchType === "http" ? ["streamable-http", "sse"] : ["stdio"]}
          onChange={(transport) => onChange({ ...form, transport: transport as InstanceForm["transport"] })}
        />
        <TextField label="URL" value={form.url} onChange={(url) => onChange({ ...form, url })} />
        <TextField label="Command" value={form.command} onChange={(command) => onChange({ ...form, command })} />
        <TextField
          label="Account label"
          value={form.accountLabel}
          onChange={(accountLabel) => onChange({ ...form, accountLabel })}
        />
      </div>
      <div className="mt-4 flex flex-wrap items-center gap-3">
        <button
          className="rounded-md bg-zinc-950 px-3 py-2 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-zinc-400"
          disabled={isSubmitting}
          type="submit"
        >
          {isSubmitting ? "Creating" : "Create instance"}
        </button>
        {error ? <p className="text-sm font-medium text-rose-700">{error}</p> : null}
      </div>
    </form>
  );
}

function OAuthFormView({
  authURL,
  error,
  form,
  instances,
  isSubmitting,
  onChange,
  onSubmit
}: {
  authURL: string;
  error: string;
  form: OAuthForm;
  instances: MCPInstanceResponse[];
  isSubmitting: boolean;
  onChange: (form: OAuthForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <form className="rounded-md border border-zinc-200 p-4" onSubmit={onSubmit}>
      <h4 className="mb-3 text-sm font-semibold text-zinc-700">Start OAuth</h4>
      <div className="grid gap-3">
        <label className="grid gap-1 text-sm font-medium text-zinc-700">
          Instance
          <select
            className="rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
            value={form.instanceID}
            onChange={(event) => onChange({ ...form, instanceID: event.target.value })}
          >
            <option value="">First available instance</option>
            {instances.map((instance) => (
              <option key={instance.ID} value={instance.ID}>
                {instance.Name}
              </option>
            ))}
          </select>
        </label>
        <TextField
          label="Authorization URL"
          required
          value={form.authorizationURL}
          onChange={(authorizationURL) => onChange({ ...form, authorizationURL })}
        />
        <TextField
          label="Resource URI"
          required
          value={form.resourceURI}
          onChange={(resourceURI) => onChange({ ...form, resourceURI })}
        />
        <TextField
          label="Redirect URI"
          required
          value={form.redirectURI}
          onChange={(redirectURI) => onChange({ ...form, redirectURI })}
        />
      </div>
      <div className="mt-4 flex flex-wrap items-center gap-3">
        <button
          className="rounded-md bg-zinc-950 px-3 py-2 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-zinc-400"
          disabled={isSubmitting || instances.length === 0}
          type="submit"
        >
          {isSubmitting ? "Starting" : "Start OAuth"}
        </button>
        {authURL ? (
          <a className="text-sm font-semibold text-zinc-950 underline" href={authURL} rel="noreferrer" target="_blank">
            Open authorization URL
          </a>
        ) : null}
        {error ? <p className="text-sm font-medium text-rose-700">{error}</p> : null}
      </div>
    </form>
  );
}

function MCPDashboard({ data, totalAccounts }: { data: MCPData; totalAccounts: number }) {
  return (
    <>
      <div className="grid gap-3 sm:grid-cols-4">
        <SummaryItem label="Instances" value={data.instances.length} />
        <SummaryItem label="Clients" value={data.clients.length} />
        <SummaryItem label="Tools" value={data.tools.length} />
        <SummaryItem label="Accounts" value={totalAccounts} />
      </div>

      <div className="overflow-hidden rounded-md border border-zinc-200">
        <table aria-label="MCP instances" className="w-full min-w-[760px] text-left text-sm">
          <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
            <tr>
              <th className="px-4 py-3 font-semibold">Instance</th>
              <th className="px-4 py-3 font-semibold">Launch</th>
              <th className="px-4 py-3 font-semibold">Transport</th>
              <th className="px-4 py-3 font-semibold">Account</th>
              <th className="px-4 py-3 font-semibold">Tools</th>
              <th className="px-4 py-3 font-semibold">Health</th>
              <th className="px-4 py-3 font-semibold">Credentials</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200">
            {data.instances.map((instance) => (
              <tr key={instance.ID}>
                <td className="px-4 py-3">
                  <div className="font-semibold text-zinc-950">{instance.Name}</div>
                  <div className="mt-1 text-xs text-zinc-500">{instance.ServerKey}</div>
                </td>
                <td className="px-4 py-3 text-zinc-600">{instance.LaunchType}</td>
                <td className="px-4 py-3 text-zinc-600">{instance.Transport}</td>
                <td className="px-4 py-3 text-zinc-600">{textValue(instance.AccountLabel)}</td>
                <td className="px-4 py-3 text-zinc-600">{toolCount(instance.ID, instance.ToolManifest, data.tools)}</td>
                <td className="px-4 py-3">
                  <StatusPill tone={statusTone(instance.HealthStatus)}>{textValue(instance.HealthStatus, "unknown")}</StatusPill>
                </td>
                <td className="px-4 py-3">
                  <CredentialKeys env={instance.Env} headers={instance.Headers} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="grid gap-4 xl:grid-cols-2">
        <DataTable title="Clients">
          {data.clients.length === 0 ? (
            <p className="px-4 py-3 text-sm text-zinc-500">No legacy MCP clients.</p>
          ) : (
            <table className="w-full text-left text-sm">
              <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
                <tr>
                  <th className="px-4 py-3 font-semibold">Client</th>
                  <th className="px-4 py-3 font-semibold">Transport</th>
                  <th className="px-4 py-3 font-semibold">Tools</th>
                  <th className="px-4 py-3 font-semibold">Health</th>
                  <th className="px-4 py-3 font-semibold">Env</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-200">
                {data.clients.map((client) => (
                  <tr key={client.ID}>
                    <td className="px-4 py-3 font-semibold text-zinc-950">{client.Name}</td>
                    <td className="px-4 py-3 text-zinc-600">{client.Transport}</td>
                    <td className="px-4 py-3 text-zinc-600">{toolCount(client.ID, client.ToolManifest, data.tools)}</td>
                    <td className="px-4 py-3">
                      <StatusPill tone={statusTone(client.HealthStatus)}>{textValue(client.HealthStatus, "unknown")}</StatusPill>
                    </td>
                    <td className="px-4 py-3">
                      <CredentialKeys env={client.Env} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </DataTable>

        <DataTable title="Accounts">
          {totalAccounts === 0 ? (
            <p className="px-4 py-3 text-sm text-zinc-500">No OAuth accounts.</p>
          ) : (
            <table className="w-full text-left text-sm">
              <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
                <tr>
                  <th className="px-4 py-3 font-semibold">Instance</th>
                  <th className="px-4 py-3 font-semibold">Account</th>
                  <th className="px-4 py-3 font-semibold">Subject</th>
                  <th className="px-4 py-3 font-semibold">Resource</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-200">
                {data.instances.flatMap((instance) =>
                  (data.accountsByInstance[instance.ID] ?? []).map((account) => (
                    <tr key={account.id}>
                      <td className="px-4 py-3 font-semibold text-zinc-950">{instance.Name}</td>
                      <td className="px-4 py-3 text-zinc-600">{account.account_label}</td>
                      <td className="px-4 py-3 text-zinc-600">{account.email || account.subject || "metadata only"}</td>
                      <td className="px-4 py-3 text-zinc-600">{textValue(account.resource_uri)}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          )}
        </DataTable>
      </div>

      <DataTable title="Tools">
        {data.tools.length === 0 ? (
          <p className="px-4 py-3 text-sm text-zinc-500">No discovered tools.</p>
        ) : (
          <table className="w-full text-left text-sm">
            <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
              <tr>
                <th className="px-4 py-3 font-semibold">Tool</th>
                <th className="px-4 py-3 font-semibold">Description</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200">
              {data.tools.map((tool) => (
                <tr key={tool.function.name}>
                  <td className="px-4 py-3 font-mono text-xs text-zinc-700">{tool.function.name}</td>
                  <td className="px-4 py-3 text-zinc-600">{textValue(tool.function.description, "No description")}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </DataTable>
    </>
  );
}

function TextField({
  label,
  onChange,
  required = false,
  value
}: {
  label: string;
  onChange: (value: string) => void;
  required?: boolean;
  value: string;
}) {
  return (
    <label className="grid gap-1 text-sm font-medium text-zinc-700">
      {label}
      <input
        className="rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
        required={required}
        type="text"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  );
}

function SelectField({
  label,
  onChange,
  options,
  value
}: {
  label: string;
  onChange: (value: string) => void;
  options: string[];
  value: string;
}) {
  return (
    <label className="grid gap-1 text-sm font-medium text-zinc-700">
      {label}
      <select
        className="rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        {options.map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    </label>
  );
}

function DataTable({ children, title }: { children: ReactNode; title: string }) {
  return (
    <section className="overflow-hidden rounded-md border border-zinc-200">
      <h4 className="border-b border-zinc-200 bg-zinc-50 px-4 py-3 text-sm font-semibold text-zinc-700">{title}</h4>
      {children}
    </section>
  );
}

function SummaryItem({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border border-zinc-200 px-4 py-3">
      <p className="text-xs font-semibold uppercase text-zinc-500">{label}</p>
      <p className="mt-1 text-xl font-semibold text-zinc-950">{value}</p>
    </div>
  );
}

function CredentialKeys({ env, headers }: { env?: Record<string, string>; headers?: Record<string, string> }) {
  const entries = [
    ...credentialEntries("env", env),
    ...credentialEntries("header", headers)
  ];
  if (entries.length === 0) {
    return <span className="text-sm text-zinc-500">none</span>;
  }

  return (
    <div className="flex flex-wrap gap-1.5">
      {entries.map((entry) => (
        <span key={`${entry.scope}:${entry.key}`} className="rounded-md border border-zinc-200 px-2 py-1 text-xs text-zinc-600">
          <span className="font-semibold text-zinc-700">
            {entry.scope}:{entry.key}
          </span>{" "}
          <span>redacted</span>
        </span>
      ))}
    </div>
  );
}

function credentialEntries(scope: "env" | "header", values?: Record<string, string>) {
  return Object.keys(values ?? {})
    .sort()
    .map((key) => ({ key, scope }));
}

function statusTone(status: string | undefined): "neutral" | "good" | "warn" | "bad" {
  const normalized = (status ?? "").toLowerCase();
  if (normalized.includes("healthy") || normalized.includes("connected") || normalized.includes("ready")) {
    return "good";
  }
  if (normalized.includes("auth") || normalized.includes("start") || normalized.includes("unknown")) {
    return "warn";
  }
  if (normalized.includes("error") || normalized.includes("fail") || normalized.includes("offline")) {
    return "bad";
  }
  return "neutral";
}

function toolCount(ownerID: string, manifest: MCPClientResponse["ToolManifest"], tools: MCPToolResponse[]) {
  if (manifest?.tools) {
    return manifest.tools.length;
  }
  const compactCount = tools.filter((tool) => tool.function.name.startsWith(`${ownerID}__`)).length;
  return compactCount;
}

function isMCPDataEmpty(data: MCPData) {
  return data.clients.length === 0 && data.instances.length === 0 && data.tools.length === 0;
}

function toApiError(error: unknown) {
  if (error instanceof ApiError) {
    return error;
  }
  return new ApiError(0, error instanceof Error ? error.message : "unknown MCP error", undefined);
}

function blankToUndefined(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

function textValue(value: string | null | undefined, fallback = "-") {
  const text = value?.trim();
  return text && text !== "" ? text : fallback;
}

function redactErrorMessage(message: string) {
  return message
    .replace(/Bearer\s+[A-Za-z0-9._~+/=-]+/gi, "Bearer redacted")
    .replace(/((?:access|refresh)[_-]?token|api[_-]?key|authorization|password|secret)=([^&\s]+)/gi, "$1=redacted")
    .replace(/\b(sk-[A-Za-z0-9_-]+)/g, "redacted");
}

function defaultRedirectURI() {
  if (typeof window === "undefined") {
    return "/api/mcp/oauth/callback";
  }
  return `${window.location.origin}/api/mcp/oauth/callback`;
}
