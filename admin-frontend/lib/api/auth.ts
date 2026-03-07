import apiClient from "@/lib/api-client";
import type { AdminUser } from "@/types/auth";

export const authApi = {
  login: async (email: string, password: string): Promise<{ user: AdminUser }> => {
    const response = await apiClient.post("/api/v1/admin/auth/login", { email, password });
    return response.data as { user: AdminUser };
  },
  logout: async (): Promise<void> => {
    await apiClient.post("/api/v1/admin/auth/logout");
  },
  me: async (): Promise<{ user: AdminUser }> => {
    const response = await apiClient.get("/api/v1/admin/auth/me");
    return response.data as { user: AdminUser };
  },
};

