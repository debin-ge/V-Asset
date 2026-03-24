import apiClient from "@/lib/api-client";
import { buildAdminApiPath } from "@/lib/admin-api-path";
import type { CookieInfo, CookieListResponse } from "@/types/cookie";

export const cookieApi = {
  list: async (params?: Record<string, string | number>) => {
    const response = await apiClient.get(buildAdminApiPath("/api/v1/admin/cookies"), { params });
    return response.data as CookieListResponse;
  },
  get: async (id: number) => {
    const response = await apiClient.get(buildAdminApiPath(`/api/v1/admin/cookies/${id}`));
    return response.data as CookieInfo;
  },
  create: async (data: Record<string, unknown>) => {
    const response = await apiClient.post(buildAdminApiPath("/api/v1/admin/cookies"), data);
    return response.data as { id: number };
  },
  update: async (id: number, data: Record<string, unknown>) => {
    await apiClient.put(buildAdminApiPath(`/api/v1/admin/cookies/${id}`), data);
  },
  delete: async (id: number) => {
    await apiClient.delete(buildAdminApiPath(`/api/v1/admin/cookies/${id}`));
  },
  freeze: async (id: number, freezeSeconds: number) => {
    const response = await apiClient.post(buildAdminApiPath(`/api/v1/admin/cookies/${id}/freeze`), {
      freeze_seconds: freezeSeconds,
    });
    return response.data as { success: boolean; frozen_until: string };
  },
};
