import apiClient from "@/lib/api-client";
import { buildAdminApiPath } from "@/lib/admin-api-path";
import type {
  ProxyCreatePayload,
  ProxyListParams,
  ProxyListResponse,
  ProxySourcePolicy,
  ProxySourceStatus,
  ProxyUpdatePayload,
  UpdateProxySourcePolicyPayload,
} from "@/types/proxy";

export const proxyApi = {
  getSourceStatus: async (): Promise<ProxySourceStatus> => {
    const response = await apiClient.get(buildAdminApiPath("/api/v1/admin/proxies/source/status"));
    return response.data as ProxySourceStatus;
  },
  getCurrentPolicy: async (): Promise<ProxySourcePolicy> => {
    const response = await apiClient.get(buildAdminApiPath("/api/v1/admin/proxy-policies/current"));
    return response.data as ProxySourcePolicy;
  },
  updatePolicy: async (id: number, payload: UpdateProxySourcePolicyPayload): Promise<void> => {
    await apiClient.put(buildAdminApiPath(`/api/v1/admin/proxy-policies/${id}`), payload);
  },
  list: async (params?: ProxyListParams): Promise<ProxyListResponse> => {
    const response = await apiClient.get(buildAdminApiPath("/api/v1/admin/proxies"), { params });
    return response.data as ProxyListResponse;
  },
  create: async (payload: ProxyCreatePayload): Promise<void> => {
    await apiClient.post(buildAdminApiPath("/api/v1/admin/proxies"), payload);
  },
  update: async (id: number, payload: ProxyUpdatePayload): Promise<void> => {
    await apiClient.put(buildAdminApiPath(`/api/v1/admin/proxies/${id}`), payload);
  },
  updateStatus: async (id: number, status: number): Promise<void> => {
    await apiClient.patch(buildAdminApiPath(`/api/v1/admin/proxies/${id}/status`), { status });
  },
  delete: async (id: number): Promise<void> => {
    await apiClient.delete(buildAdminApiPath(`/api/v1/admin/proxies/${id}`));
  },
};
