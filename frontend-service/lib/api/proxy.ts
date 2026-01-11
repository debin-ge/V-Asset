import apiClient from '../api-client';

// Proxy status enum
export const ProxyStatus = {
    ACTIVE: 0,
    INACTIVE: 1,
    CHECKING: 2,
} as const;

export const ProxyStatusLabel: Record<number, string> = {
    [ProxyStatus.ACTIVE]: 'Active',
    [ProxyStatus.INACTIVE]: 'Inactive',
    [ProxyStatus.CHECKING]: 'Checking',
};

// Proxy info interface
export interface ProxyInfo {
    id: number;
    ip: string;
    port: number;
    username: string;
    password: string;
    protocol: string;
    region: string;
    status: number;
    last_check_at: string;
    last_check_result: string;
    success_count: number;
    fail_count: number;
    last_used_at: string;
    created_at: string;
    updated_at: string;
}

export interface ProxyListResponse {
    total: number;
    page: number;
    page_size: number;
    items: ProxyInfo[];
}

export interface CreateProxyRequest {
    ip: string;
    port: number;
    username?: string;
    password?: string;
    protocol?: string;
    region?: string;
    check_health?: boolean;
}

export interface CreateProxyResponse {
    id: number;
    health_check_passed: boolean;
    health_check_error: string;
}

export interface UpdateProxyRequest {
    id: number;
    username?: string;
    password?: string;
    protocol?: string;
    region?: string;
}

export interface ListProxiesParams {
    status?: number;
    protocol?: string;
    region?: string;
    page?: number;
    page_size?: number;
}

export interface HealthCheckResponse {
    healthy: boolean;
    error: string;
    latency_ms: number;
}

export const proxyApi = {
    // Create proxy
    create: async (data: CreateProxyRequest): Promise<CreateProxyResponse> => {
        const response = await apiClient.post('/api/v1/admin/proxies', data);
        return response.data as CreateProxyResponse;
    },

    // Update proxy
    update: async (data: UpdateProxyRequest): Promise<void> => {
        await apiClient.put(`/api/v1/admin/proxies/${data.id}`, data);
    },

    // Delete proxy
    delete: async (id: number): Promise<void> => {
        await apiClient.delete(`/api/v1/admin/proxies/${id}`);
    },

    // Get proxy
    get: async (id: number): Promise<ProxyInfo> => {
        const response = await apiClient.get(`/api/v1/admin/proxies/${id}`);
        return response.data as ProxyInfo;
    },

    // List proxies
    list: async (params?: ListProxiesParams): Promise<ProxyListResponse> => {
        const response = await apiClient.get('/api/v1/admin/proxies', { params });
        return response.data as ProxyListResponse;
    },

    // Health check
    checkHealth: async (id: number): Promise<HealthCheckResponse> => {
        const response = await apiClient.post(`/api/v1/admin/proxies/${id}/health-check`);
        return response.data as HealthCheckResponse;
    },
};
