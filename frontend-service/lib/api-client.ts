import axios, { AxiosInstance, AxiosError } from 'axios';

import { resolveApiBaseUrl } from './runtime-config';

// Token存储键名
const TOKEN_KEY = 'youdlp-token';
const REFRESH_TOKEN_KEY = 'youdlp-refresh-token';

// 创建axios实例
const apiClient: AxiosInstance = axios.create({
    baseURL: '',
    timeout: 30000,
    headers: {
        'Content-Type': 'application/json',
    },
});

// 请求拦截器 - 添加Token
apiClient.interceptors.request.use(
    (config) => {
        config.baseURL = resolveApiBaseUrl();

        const token = localStorage.getItem(TOKEN_KEY);
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => Promise.reject(error)
);

// 响应拦截器 - 统一错误处理
apiClient.interceptors.response.use(
    (response) => {
        // 跳过 blob 类型响应（如文件下载），直接返回原始响应
        if (response.config.responseType === 'blob') {
            return response;
        }

        // 后端响应格式: { code, message, data }
        const { code, message, data } = response.data;
        if (code !== 0) {
            return Promise.reject(new Error(message || '请求失败'));
        }
        return { ...response, data: data };
    },
    async (error: AxiosError) => {
        if (error.response?.status === 401) {
            // Token过期，尝试刷新或清除
            localStorage.removeItem(TOKEN_KEY);
            localStorage.removeItem(REFRESH_TOKEN_KEY);
            localStorage.removeItem('youdlp-user');
            // 触发事件通知组件
            window.dispatchEvent(new CustomEvent('auth:logout'));
        }

        const responseData = error.response?.data as { message?: string } | undefined;
        if (responseData?.message) {
            return Promise.reject(new Error(responseData.message));
        }

        return Promise.reject(error);
    }
);

// Token管理函数
export const tokenManager = {
    setTokens: (token: string, refreshToken: string) => {
        localStorage.setItem(TOKEN_KEY, token);
        localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
    },
    getToken: () => localStorage.getItem(TOKEN_KEY),
    getRefreshToken: () => localStorage.getItem(REFRESH_TOKEN_KEY),
    clearTokens: () => {
        localStorage.removeItem(TOKEN_KEY);
        localStorage.removeItem(REFRESH_TOKEN_KEY);
    },
    isAuthenticated: () => !!localStorage.getItem(TOKEN_KEY),
};

export default apiClient;
