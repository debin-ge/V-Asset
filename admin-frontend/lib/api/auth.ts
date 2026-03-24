import apiClient from "@/lib/api-client";
import { buildAdminApiPath } from "@/lib/admin-api-path";
import type { AdminUser } from "@/types/auth";

export const authApi = {
  login: async (email: string, password: string): Promise<{ user: AdminUser }> => {
    const response = await apiClient.post(buildAdminApiPath("/api/v1/admin/auth/login"), { email, password });
    return response.data as { user: AdminUser };
  },
  logout: async (): Promise<void> => {
    await apiClient.post(buildAdminApiPath("/api/v1/admin/auth/logout"));
  },
  me: async (): Promise<{ user: AdminUser }> => {
    const response = await apiClient.get(buildAdminApiPath("/api/v1/admin/auth/me"));
    return response.data as { user: AdminUser };
  },
};
