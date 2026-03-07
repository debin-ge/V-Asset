import apiClient from '../api-client';

export interface ProxySourceStatus {
    healthy: boolean;
    mode: string;
    message: string;
    proxy_url?: string;
    proxy_lease_id?: string;
    proxy_expire_at?: string;
    checked_at: string;
}

export const proxyApi = {
    getSourceStatus: async (): Promise<ProxySourceStatus> => {
        const response = await apiClient.get('/api/v1/admin/proxies/source/status');
        return response.data as ProxySourceStatus;
    },
};
