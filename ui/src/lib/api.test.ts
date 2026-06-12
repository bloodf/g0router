import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { apiFetch, ApiError } from "./api";

describe("apiFetch", () => {
  beforeEach(() => {
    vi.stubGlobal("window", { location: { origin: "http://localhost:20129" } });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("unwraps {data, error} envelope", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        text: async () => JSON.stringify({ data: { id: 1 }, error: null }),
      })
    );

    const result = await apiFetch<{ id: number }>("/api/test");
    expect(result).toEqual({ id: 1 });
  });

  it("throws on error envelope", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        text: async () =>
          JSON.stringify({ data: null, error: { message: "not found" } }),
      })
    );

    await expect(apiFetch("/api/test")).rejects.toThrow(ApiError);
    await expect(apiFetch("/api/test")).rejects.toThrow("not found");
  });

  it("throws on HTTP error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        text: async () => "",
      })
    );

    await expect(apiFetch("/api/test")).rejects.toThrow(ApiError);
    await expect(apiFetch("/api/test")).rejects.toThrow("HTTP 500");
  });
});
