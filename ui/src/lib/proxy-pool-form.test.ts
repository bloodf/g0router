import { describe, it, expect } from "vitest";
import { toProxyPoolPayload, type ProxyPoolForm } from "./proxy-pool-form";

const baseForm: ProxyPoolForm = {
  name: "EU West",
  protocol: "https",
  host: "eu-west.proxy.example.com",
  port: "3128",
  username: "user2",
  is_active: true,
};

describe("toProxyPoolPayload", () => {
  it("maps name/protocol/host/username/is_active from the form", () => {
    const payload = toProxyPoolPayload(baseForm);
    expect(payload.name).toBe("EU West");
    expect(payload.protocol).toBe("https");
    expect(payload.host).toBe("eu-west.proxy.example.com");
    expect(payload.username).toBe("user2");
    expect(payload.is_active).toBe(true);
  });

  it("coerces the port string to a number", () => {
    const payload = toProxyPoolPayload(baseForm);
    expect(payload.port).toBe(3128);
    expect(typeof payload.port).toBe("number");
  });

  it("defaults an empty/invalid port to 0", () => {
    expect(toProxyPoolPayload({ ...baseForm, port: "" }).port).toBe(0);
    expect(toProxyPoolPayload({ ...baseForm, port: "abc" }).port).toBe(0);
  });

  it("preserves an empty username (optional auth)", () => {
    const payload = toProxyPoolPayload({ ...baseForm, username: "" });
    expect(payload.username).toBe("");
  });

  it("does not mutate the source form", () => {
    const before = JSON.stringify(baseForm);
    toProxyPoolPayload(baseForm);
    expect(JSON.stringify(baseForm)).toBe(before);
  });
});
