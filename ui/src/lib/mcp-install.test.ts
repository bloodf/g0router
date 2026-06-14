import { describe, it, expect } from "vitest";
import { toInstancePayload } from "./mcp-install";
import type { McpClient } from "./types";

const stdioClient: McpClient = {
  ID: "c1",
  Name: "Filesystem",
  Transport: "stdio",
  Command: "npx",
  Args: ["-y", "@modelcontextprotocol/server-filesystem"],
  Env: { ROOT: "/tmp" },
  IsActive: true,
  HealthStatus: "healthy",
  CreatedAt: "2024-01-01T00:00:00Z",
};

const sseClient: McpClient = {
  ID: "c2",
  Name: "GitHub",
  Transport: "sse",
  URL: "https://api.github.com/mcp",
  IsActive: true,
  HealthStatus: "healthy",
  CreatedAt: "2024-01-01T00:00:00Z",
};

describe("toInstancePayload", () => {
  it("maps a stdio client's name/transport/command/args", () => {
    const payload = toInstancePayload(stdioClient);
    expect(payload.Name).toBe("Filesystem");
    expect(payload.Transport).toBe("stdio");
    expect(payload.Command).toBe("npx");
    expect(payload.Args).toEqual([
      "-y",
      "@modelcontextprotocol/server-filesystem",
    ]);
  });

  it("maps an sse client's url and omits stdio-only fields", () => {
    const payload = toInstancePayload(sseClient);
    expect(payload.Name).toBe("GitHub");
    expect(payload.Transport).toBe("sse");
    expect(payload.URL).toBe("https://api.github.com/mcp");
    expect("Command" in payload).toBe(false);
    expect("Args" in payload).toBe(false);
  });

  it("omits absent optional fields (no URL for stdio)", () => {
    const payload = toInstancePayload(stdioClient);
    expect("URL" in payload).toBe(false);
  });

  it("does not mutate the source client", () => {
    const before = JSON.stringify(stdioClient);
    toInstancePayload(stdioClient);
    expect(JSON.stringify(stdioClient)).toBe(before);
  });

  it("does not carry the client ID into the payload", () => {
    const payload = toInstancePayload(stdioClient) as Record<string, unknown>;
    expect("ID" in payload).toBe(false);
  });
});
