import apiClient from "@/lib/api-client";
import type { CookieListResponse } from "@/types/cookie";

export const cookieApi = {
  list: async (params?: Record<string, string | number>) => {
    const response = await apiClient.get("/api/v1/admin/cookies", { params });
    return response.data as CookieListResponse;
  },
  create: async (data: Record<string, unknown>) => {
    const response = await apiClient.post("/api/v1/admin/cookies", data);
    return response.data as { id: number };
  },
  update: async (id: number, data: Record<string, unknown>) => {
    await apiClient.put(`/api/v1/admin/cookies/${id}`, data);
  },
  delete: async (id: number) => {
    await apiClient.delete(`/api/v1/admin/cookies/${id}`);
  },
  freeze: async (id: number, freezeSeconds: number) => {
    const response = await apiClient.post(`/api/v1/admin/cookies/${id}/freeze`, {
      freeze_seconds: freezeSeconds,
    });
    return response.data as { success: boolean; frozen_until: string };
  },
};

