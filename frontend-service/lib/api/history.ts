import apiClient from '../api-client';

export interface HistoryItem {
    history_id: number;
    task_id: string;
    url: string;
    platform: string;
    title: string;
    mode: string;
    quality: string;
    file_size: number;
    status: number;
    file_name: string;
    created_at: string;
    completed_at: string;
    thumbnail: string;
    duration: number;
    author: string;
}

export interface HistoryResponse {
    total: number;
    page: number;
    page_size: number;
    items: HistoryItem[];
}

export interface QuotaResponse {
    daily_limit: number;
    daily_used: number;
    remaining: number;
    reset_at: string;
}

export interface StatsResponse {
    total_downloads: number;
    success_downloads: number;
    failed_downloads: number;
    total_size_bytes: number;
    top_platforms: { platform: string; count: number }[];
    recent_activity: { date: string; count: number }[];
}

export interface HistoryParams {
    status?: number;
    platform?: string;
    start_date?: string;
    end_date?: string;
    page?: number;
    page_size?: number;
    sort_by?: string;
    sort_order?: string;
}

export const historyApi = {
    // 获取下载历史
    getHistory: async (params?: HistoryParams): Promise<HistoryResponse> => {
        const response = await apiClient.get('/api/v1/user/history', { params });
        return response.data as HistoryResponse;
    },

    // 删除历史记录
    deleteHistory: async (historyId: number): Promise<void> => {
        await apiClient.delete(`/api/v1/user/history/${historyId}`);
    },

    // 获取配额
    getQuota: async (): Promise<QuotaResponse> => {
        const response = await apiClient.get('/api/v1/user/quota');
        return response.data as QuotaResponse;
    },

    // 获取用户统计
    getStats: async (): Promise<StatsResponse> => {
        const response = await apiClient.get('/api/v1/user/stats');
        return response.data as StatsResponse;
    },
};
