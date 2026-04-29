import apiClient from "@/lib/api-client";
import { buildAdminApiPath } from "@/lib/admin-api-path";
import type { DashboardHealthResponse, Overview, RequestTrend, UserStats } from "@/types/stats";

export const statsApi = {
  getOverview: async (): Promise<Overview> => {
    const response = await apiClient.get(buildAdminApiPath("/api/v1/admin/stats/overview"));
    return response.data as Overview;
  },
  getRequestTrend: async (granularity: "day" | "hour", limit: number): Promise<RequestTrend> => {
    const response = await apiClient.get(buildAdminApiPath("/api/v1/admin/stats/requests"), {
      params: { granularity, limit },
    });
    return response.data as RequestTrend;
  },
  getDashboardHealth: async (): Promise<DashboardHealthResponse> => {
    const response = await apiClient.get(buildAdminApiPath("/api/v1/admin/stats/dashboard-health"));
    return response.data as DashboardHealthResponse;
  },
  getUsers: async (): Promise<UserStats> => {
    const response = await apiClient.get(buildAdminApiPath("/api/v1/admin/stats/users"));
    return response.data as UserStats;
  },
};
