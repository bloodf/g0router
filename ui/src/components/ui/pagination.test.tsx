import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { Pagination, paginationRange } from "./pagination";

describe("paginationRange", () => {
  it("returns every page when the total is small (1, 3)", () => {
    expect(paginationRange(1, 3)).toEqual([1, 2, 3]);
  });

  it("windows the start with a trailing ellipsis (1, 10)", () => {
    expect(paginationRange(1, 10)).toEqual([1, 2, 3, 4, 5, "ellipsis", 10]);
  });

  it("windows the middle with both ellipses (5, 10)", () => {
    expect(paginationRange(5, 10)).toEqual([
      1,
      "ellipsis",
      4,
      5,
      6,
      "ellipsis",
      10,
    ]);
  });

  it("windows the end with a leading ellipsis (10, 10)", () => {
    expect(paginationRange(10, 10)).toEqual([1, "ellipsis", 6, 7, 8, 9, 10]);
  });
});

describe("Pagination", () => {
  it("renders a nav with page buttons", () => {
    const html = renderToString(
      <Pagination page={1} totalPages={5} onPageChange={() => {}} />
    );
    expect(html).toContain('aria-label="pagination"');
    expect(html).toContain("<nav");
  });

  it("marks the current page with aria-current", () => {
    const html = renderToString(
      <Pagination page={3} totalPages={5} onPageChange={() => {}} />
    );
    expect(html).toContain('aria-current="page"');
  });

  it("disables prev at the first page and next at the last page", () => {
    const first = renderToString(
      <Pagination page={1} totalPages={5} onPageChange={() => {}} />
    );
    const prevFirst = first.match(/data-testid="pagination-prev"[^>]*>/);
    expect(prevFirst![0]).toContain("disabled");

    const last = renderToString(
      <Pagination page={5} totalPages={5} onPageChange={() => {}} />
    );
    const nextLast = last.match(/data-testid="pagination-next"[^>]*>/);
    expect(nextLast![0]).toContain("disabled");
  });
});
