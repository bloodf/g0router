import { describe, it, expect } from "vitest";
import { moveStep } from "./combo-order";

describe("moveStep", () => {
  it("moves an element down", () => {
    expect(moveStep(["a", "b", "c"], 0, 2)).toEqual(["b", "c", "a"]);
  });

  it("moves an element up", () => {
    expect(moveStep(["a", "b", "c"], 2, 0)).toEqual(["c", "a", "b"]);
  });

  it("moves an adjacent element by one position", () => {
    expect(moveStep(["a", "b", "c"], 0, 1)).toEqual(["b", "a", "c"]);
  });

  it("is a no-op when from === to", () => {
    expect(moveStep(["a", "b", "c"], 1, 1)).toEqual(["a", "b", "c"]);
  });

  it("leaves order intact for an out-of-range from index", () => {
    expect(moveStep(["a", "b", "c"], 5, 0)).toEqual(["a", "b", "c"]);
    expect(moveStep(["a", "b", "c"], -1, 0)).toEqual(["a", "b", "c"]);
  });

  it("leaves order intact for an out-of-range to index", () => {
    expect(moveStep(["a", "b", "c"], 0, 9)).toEqual(["a", "b", "c"]);
    expect(moveStep(["a", "b", "c"], 0, -1)).toEqual(["a", "b", "c"]);
  });

  it("does not mutate the input array", () => {
    const input = ["a", "b", "c"];
    const result = moveStep(input, 0, 2);
    expect(input).toEqual(["a", "b", "c"]);
    expect(result).not.toBe(input);
  });

  it("preserves relative order of untouched elements", () => {
    expect(moveStep(["a", "b", "c", "d", "e"], 3, 1)).toEqual([
      "a",
      "d",
      "b",
      "c",
      "e",
    ]);
  });

  it("works with object members keyed by reference", () => {
    const a = { provider: "groq", model: "llama-3-70b" };
    const b = { provider: "openai", model: "gpt-4o-mini" };
    expect(moveStep([a, b], 0, 1)).toEqual([b, a]);
  });
});
