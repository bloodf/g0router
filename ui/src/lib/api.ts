import { useUserStore } from "@/stores/user";

export class ApiError extends Error {
  constructor(
    message: string,
    public status?: number,
    public code?: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

interface ApiEnvelope<T> {
  data?: T;
  error?: { message: string; code?: string } | null;
}

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const token = useUserStore.getState().token;
  const headers = new Headers(init.headers);
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  if (
    init.body &&
    typeof init.body === "string" &&
    !headers.has("Content-Type")
  ) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${window.location.origin}${path}`, {
    ...init,
    headers,
  });

  const text = await response.text();
  let envelope: ApiEnvelope<T> | undefined;
  if (text) {
    try {
      envelope = JSON.parse(text) as ApiEnvelope<T>;
    } catch {
      throw new ApiError(`non-JSON response: ${text}`, response.status);
    }
  }

  if (!response.ok) {
    const message = envelope?.error?.message ?? `HTTP ${response.status}`;
    throw new ApiError(message, response.status, envelope?.error?.code);
  }

  if (envelope?.error) {
    throw new ApiError(envelope.error.message, response.status, envelope.error.code);
  }

  return envelope?.data as T;
}
