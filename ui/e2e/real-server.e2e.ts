import { expect, test } from "@playwright/test";
import { spawn, type ChildProcessWithoutNullStreams } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import net from "node:net";

const dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(dirname, "../..");

test.describe("dashboard real server smoke", () => {
  test.skip(({ isMobile }) => isMobile);

  test("uses embedded dashboard against a real authenticated g0router server", async ({ page }) => {
    const apiKeySecret = "real-server-e2e-secret";
    const dataDir = await mkdtemp(path.join(tmpdir(), "g0router-real-e2e-"));
    const port = await freePort();
    const env = {
      ...process.env,
      API_KEY_SECRET: apiKeySecret,
      DATA_DIR: dataDir,
      REQUIRE_API_KEY: "true"
    };

    let server: ChildProcessWithoutNullStreams | undefined;
    try {
      const rawKey = await createAPIKey(dataDir, apiKeySecret);
      server = spawn("go", ["run", "./cmd/g0router", "--data-dir", dataDir, "serve", "--port", String(port)], {
        cwd: repoRoot,
        env
      });
      await waitForHealth(port, server);

      await page.goto(`http://127.0.0.1:${port}/`);
      await expect(page.getByRole("heading", { name: "Gateway overview" })).toBeVisible();

      await page.getByLabel("Control-plane API key").fill(rawKey);
      await page.getByRole("button", { name: "Save key" }).click();

      await page.getByRole("button", { exact: true, name: "Settings" }).click();
      await expect(page.getByRole("heading", { exact: true, name: "Settings" })).toBeVisible();
      await expect(page.getByLabel("Proxy URL")).toHaveValue(/.*/);

      await page.getByRole("button", { name: "API Keys" }).click();
      await expect(page.getByRole("heading", { exact: true, name: "API Keys" })).toBeVisible();
      await page.getByLabel("Key name").fill("browser-real");
      await page.getByRole("button", { name: "Create key" }).click();
      await expect(page.getByText("New gateway key")).toBeVisible();
      await expect(page.locator("code").filter({ hasText: /^g0r_/ })).toBeVisible();
      await page.getByRole("button", { name: "Dismiss" }).click();
      await expect(page.getByRole("table", { name: "API keys" })).toContainText("browser-real");
    } finally {
      if (server) {
        server.kill("SIGTERM");
      }
      await rm(dataDir, { force: true, recursive: true });
    }
  });
});

async function createAPIKey(dataDir: string, apiKeySecret: string): Promise<string> {
  const { stdout } = await run("go", ["run", "./cmd/g0router", "--data-dir", dataDir, "keys", "add", "real-e2e"], {
    API_KEY_SECRET: apiKeySecret,
    DATA_DIR: dataDir
  });
  const parts = stdout.trim().split(/\s+/);
  const raw = parts[parts.length - 1];
  if (!raw?.startsWith("g0r_")) {
    throw new Error(`unexpected API key output: ${stdout}`);
  }
  return raw;
}

async function waitForHealth(port: number, server: ChildProcessWithoutNullStreams): Promise<void> {
  const deadline = Date.now() + 30000;
  let stderr = "";
  server.stderr.on("data", (chunk) => {
    stderr += chunk.toString();
  });
  while (Date.now() < deadline) {
    if (server.exitCode !== null) {
      throw new Error(`g0router serve exited early with ${server.exitCode}: ${stderr}`);
    }
    try {
      const response = await fetch(`http://127.0.0.1:${port}/healthz`);
      if (response.ok) {
        return;
      }
    } catch {
      // Retry until the Go server is listening.
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error(`timed out waiting for g0router healthz: ${stderr}`);
}

async function freePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      server.close(() => {
        if (address && typeof address === "object") {
          resolve(address.port);
          return;
        }
        reject(new Error("failed to allocate port"));
      });
    });
    server.on("error", reject);
  });
}

function run(command: string, args: string[], env: Record<string, string>): Promise<{ stdout: string; stderr: string }> {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      cwd: repoRoot,
      env: { ...process.env, ...env }
    });
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (chunk) => {
      stdout += chunk.toString();
    });
    child.stderr.on("data", (chunk) => {
      stderr += chunk.toString();
    });
    child.on("error", reject);
    child.on("close", (code) => {
      if (code === 0) {
        resolve({ stdout, stderr });
        return;
      }
      reject(new Error(`${command} ${args.join(" ")} exited ${code}: ${stderr}`));
    });
  });
}
