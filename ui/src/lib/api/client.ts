import { toast } from "sonner";

export class ApiError extends Error {
  status: number;
  payload: unknown;

  constructor(status: number, payload: unknown) {
    super(typeof payload === "string" ? payload : JSON.stringify(payload));
    this.status = status;
    this.payload = payload;
    this.name = "ApiError";
  }
}

export interface ApiOptions {
  method?: "GET" | "POST" | "PUT" | "DELETE" | "PATCH";
  body?: any;
  silent?: boolean;
}

export async function apiFetch<T = any>(
  path: string,
  opts: ApiOptions = {},
): Promise<T> {
  const method = opts.method || "GET";
  try {
    const headers: Record<string, string> = {};
    const fetchOpts: RequestInit = {
      method,
      credentials: "same-origin",
      headers,
    };

    if (method !== "GET" && opts.body !== undefined) {
      headers["Content-Type"] = "application/json";
      fetchOpts.body = JSON.stringify(opts.body);
    }

    const response = await fetch(path, fetchOpts);

    // 204 No Content — nothing to parse
    if (response.status === 204) {
      return undefined as T;
    }

    const contentType = response.headers.get("content-type");
    if (!contentType || !contentType.includes("application/json")) {
      const text = await response.text();
      throw new ApiError(response.status, text || response.statusText || "Request failed");
    }

    const json = await response.json();

    // Non-2xx status: backend returned an error
    if (!response.ok) {
      const msg = typeof json.error === "string" ? json.error : JSON.stringify(json);
      throw new ApiError(response.status, msg);
    }

    // 2xx status — unwrap {data} envelope if present
    if ("data" in json) {
      return json.data as T;
    }
    return json as T;
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.status === 401) {
        if (typeof window !== "undefined" && !window.location.pathname.startsWith("/login")) {
          window.location.href = "/login";
        }
      } else if (e.status === 429) {
        const payload = e.payload as any;
        if (!opts.silent) toast.warning(`Rate limited. Retry in ${payload.retry_after_seconds ?? 15}s`);
      } else if (!opts.silent) {
        const msg = typeof e.payload === "string" ? e.payload : (e.payload as any)?.error || "Request failed";
        toast.error(msg);
      }
    } else if (!opts.silent) {
      toast.error(String((e as Error).message || e));
    }
    throw e;
  }
}

export function normalizeListResponse<T>(response: any): { items: T[]; total: number } {
  if (response && Array.isArray(response.items)) {
    return response;
  }
  if (response && Array.isArray(response.data)) {
    return { items: response.data, total: response.total ?? response.data.length };
  }
  if (Array.isArray(response)) {
    return { items: response, total: response.length };
  }
  return { items: [], total: 0 };
}
