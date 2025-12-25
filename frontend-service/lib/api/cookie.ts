import apiClient from '../api-client';

// Cookie 状态枚举
export const CookieStatus = {
    ACTIVE: 0,
    EXPIRED: 1,
    FROZEN: 2,
} as const;

export const CookieStatusLabel: Record<number, string> = {
    [CookieStatus.ACTIVE]: '可用',
    [CookieStatus.EXPIRED]: '已过期',
    [CookieStatus.FROZEN]: '冷冻中',
};

// 平台选项
export const PlatformOptions = [
    { value: 'youtube', label: 'YouTube' },
    { value: 'bilibili', label: 'Bilibili' },
    { value: 'tiktok', label: 'TikTok' },
    { value: 'twitter', label: 'Twitter' },
    { value: 'instagram', label: 'Instagram' },
];

// Cookie 信息接口
export interface CookieInfo {
    id: number;
    platform: string;
    name: string;
    content: string;
    status: number;
    expire_at: string;
    frozen_until: string;
    freeze_seconds: number;
    last_used_at: string;
    use_count: number;
    success_count: number;
    fail_count: number;
    created_at: string;
    updated_at: string;
}

export interface CookieListResponse {
    total: number;
    page: number;
    page_size: number;
    items: CookieInfo[];
}

export interface CreateCookieRequest {
    platform: string;
    name: string;
    content: string;
    expire_at?: string;
    freeze_seconds?: number;
}

export interface UpdateCookieRequest {
    id: number;
    name?: string;
    content?: string;
    expire_at?: string;
    freeze_seconds?: number;
}

export interface ListCookiesParams {
    platform?: string;
    status?: number;
    page?: number;
    page_size?: number;
}

export interface FreezeCookieResponse {
    success: boolean;
    frozen_until: string;
}

export const cookieApi = {
    // 创建 Cookie
    create: async (data: CreateCookieRequest): Promise<{ id: number }> => {
        const response = await apiClient.post('/api/v1/admin/cookies', data);
        return response.data as { id: number };
    },

    // 更新 Cookie
    update: async (data: UpdateCookieRequest): Promise<void> => {
        await apiClient.put(`/api/v1/admin/cookies/${data.id}`, data);
    },

    // 删除 Cookie
    delete: async (id: number): Promise<void> => {
        await apiClient.delete(`/api/v1/admin/cookies/${id}`);
    },

    // 获取 Cookie
    get: async (id: number): Promise<CookieInfo> => {
        const response = await apiClient.get(`/api/v1/admin/cookies/${id}`);
        return response.data as CookieInfo;
    },

    // 列表 Cookie
    list: async (params?: ListCookiesParams): Promise<CookieListResponse> => {
        const response = await apiClient.get('/api/v1/admin/cookies', { params });
        return response.data as CookieListResponse;
    },

    // 冷冻 Cookie
    freeze: async (id: number, freezeSeconds?: number): Promise<FreezeCookieResponse> => {
        const response = await apiClient.post(`/api/v1/admin/cookies/${id}/freeze`, {
            freeze_seconds: freezeSeconds || 0,
        });
        return response.data as FreezeCookieResponse;
    },
};
