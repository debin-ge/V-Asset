import apiClient from '../api-client';

// 代理状态枚举
export const ProxyStatus = {
    ACTIVE: 0,
    INACTIVE: 1,
    CHECKING: 2,
} as const;

export const ProxyStatusLabel: Record<number, string> = {
    [ProxyStatus.ACTIVE]: '可用',
    [ProxyStatus.INACTIVE]: '不可用',
    [ProxyStatus.CHECKING]: '检查中',
};

// 代理信息接口
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
    // 创建代理
    create: async (data: CreateProxyRequest): Promise<CreateProxyResponse> => {
        const response = await apiClient.post('/api/v1/admin/proxies', data);
        return response.data as CreateProxyResponse;
    },

    // 更新代理
    update: async (data: UpdateProxyRequest): Promise<void> => {
        await apiClient.put(`/api/v1/admin/proxies/${data.id}`, data);
    },

    // 删除代理
    delete: async (id: number): Promise<void> => {
        await apiClient.delete(`/api/v1/admin/proxies/${id}`);
    },

    // 获取代理
    get: async (id: number): Promise<ProxyInfo> => {
        const response = await apiClient.get(`/api/v1/admin/proxies/${id}`);
        return response.data as ProxyInfo;
    },

    // 列表代理
    list: async (params?: ListProxiesParams): Promise<ProxyListResponse> => {
        const response = await apiClient.get('/api/v1/admin/proxies', { params });
        return response.data as ProxyListResponse;
    },

    // 健康检查
    checkHealth: async (id: number): Promise<HealthCheckResponse> => {
        const response = await apiClient.post(`/api/v1/admin/proxies/${id}/health-check`);
        return response.data as HealthCheckResponse;
    },
};
