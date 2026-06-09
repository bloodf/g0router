import type { Route } from "@playwright/test";

export function json(route: Route, data: unknown, status = 200) {
  return route.fulfill({
    status,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ data }),
  });
}

export function error(route: Route, message: string, status = 400) {
  return route.fulfill({
    status,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ error: message }),
  });
}

export function extractId(pathname: string, index: number) {
  return pathname.split("/")[index];
}
