import apiClient, { tokenManager } from '../api-client';

export interface User {
    user_id: string;
    email: string;
    nickname: string;
    avatar_url?: string;
    role?: number;
    created_at?: string;
}

export interface LoginResponse {
    token: string;
    refresh_token: string;
    expires_in: number;
    user: User;
}

export interface RegisterResponse {
    user_id: string;
    email: string;
    nickname: string;
}

export const authApi = {
    // 登录
    login: async (email: string, password: string): Promise<LoginResponse> => {
        const response = await apiClient.post('/api/v1/auth/login', { email, password });
        const data = response.data as LoginResponse;
        // 保存Token
        tokenManager.setTokens(data.token, data.refresh_token);
        return data;
    },

    // 注册
    register: async (email: string, password: string, nickname: string): Promise<RegisterResponse> => {
        const response = await apiClient.post('/api/v1/auth/register', { email, password, nickname });
        return response.data as RegisterResponse;
    },

    // 获取用户信息
    getProfile: async (): Promise<User> => {
        const response = await apiClient.get('/api/v1/auth/profile');
        return response.data as User;
    },

    // 更新用户信息
    updateProfile: async (nickname: string): Promise<User> => {
        const response = await apiClient.put('/api/v1/auth/profile', { nickname });
        return response.data as User;
    },

    // 修改密码
    changePassword: async (oldPassword: string, newPassword: string): Promise<void> => {
        await apiClient.put('/api/v1/auth/password', {
            old_password: oldPassword,
            new_password: newPassword
        });
    },

    // 登出
    logout: async (): Promise<void> => {
        try {
            await apiClient.post('/api/v1/auth/logout');
        } finally {
            tokenManager.clearTokens();
            localStorage.removeItem('v-asset-user');
        }
    },
};

