import type { McpClient } from "./types";

// InstanceCreate is the body sent to POST /api/mcp/instances when installing a
// browsed MCP server (PAR-UI-054, §1.6). PascalCase keys mirror the mcp mock's
// instance shape (§1.2/§1.4); the mock fabricates ID/CreatedAt/IsActive itself.
export interface InstanceCreate {
  Name: string;
  Transport: string;
  Command?: string;
  Args?: string[];
  URL?: string;
}

// toInstancePayload maps a browsed McpClient to the install payload. Pure: it
// derives only the transport-relevant config (stdio -> Command/Args, sse/url ->
// URL) and never carries the source client's ID or health/active state. This is
// the authoritative install-contract proof (§1.6 point 4).
export function toInstancePayload(client: McpClient): InstanceCreate {
  const payload: InstanceCreate = {
    Name: client.Name,
    Transport: client.Transport,
  };
  if (client.Command !== undefined) payload.Command = client.Command;
  if (client.Args !== undefined) payload.Args = client.Args;
  if (client.URL !== undefined) payload.URL = client.URL;
  return payload;
}
