import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "./api/client";
import type { AuthStatus } from "./types";

export function useAuthStatus() {
  return useQuery({
    queryKey: ["auth", "status"],
    queryFn: () => apiFetch<AuthStatus>("/api/auth/status", { silent: true }),
    staleTime: 30000,
  });
}

export function useLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: { username: string; password: string }) =>
      apiFetch("/api/auth/login", { method: "POST", body: b }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["auth"] }),
  });
}

export function useSetup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: { username: string; password: string; display_name?: string }) =>
      apiFetch("/api/auth/setup", { method: "POST", body: b }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["auth"] }),
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiFetch("/api/auth/logout", { method: "POST" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["auth"] }),
  });
}
