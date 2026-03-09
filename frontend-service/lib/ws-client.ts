import { tokenManager } from './api-client';

function resolveWsBaseUrl(): string {
    const explicitWsUrl = process.env.NEXT_PUBLIC_WS_URL;
    if (explicitWsUrl) {
        return explicitWsUrl;
    }

    const apiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL;
    if (apiBaseUrl) {
        try {
            const apiUrl = new URL(apiBaseUrl);
            apiUrl.protocol = apiUrl.protocol === 'https:' ? 'wss:' : 'ws:';
            return apiUrl.origin;
        } catch (error) {
            console.warn('Invalid NEXT_PUBLIC_API_BASE_URL for WebSocket resolution:', error);
        }
    }

    if (typeof window !== 'undefined') {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        return `${protocol}//${window.location.host}`;
    }

    return 'ws://localhost:8080';
}

const WS_BASE_URL = resolveWsBaseUrl();

export interface ProgressData {
    task_id: string;
    status: number | string;   // 0=待处理, 1=下载中, 2=完成, 3=失败 或 "pending"/"downloading"/"completed"/"failed"
    status_text: string;
    percent: number;
    downloaded_bytes: number;
    total_bytes: number;
    speed: string;
    eta: string;
    file_path?: string;
    error_message?: string;
}

type ProgressCallback = (progress: ProgressData) => void;

class ProgressWebSocket {
    private ws: WebSocket | null = null;
    private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    private listeners: Map<string, ProgressCallback> = new Map();
    private reconnectAttempts = 0;
    private maxReconnectAttempts = 5;
    private isConnecting = false;

    connect(): void {
        if (this.isConnecting || this.ws?.readyState === WebSocket.OPEN) {
            return;
        }

        const token = tokenManager.getToken();
        if (!token) {
            console.warn('No token available for WebSocket connection');
            return;
        }

        this.isConnecting = true;
        const wsUrl = `${WS_BASE_URL}/api/v1/ws/progress?token=${token}`;

        try {
            this.ws = new WebSocket(wsUrl);

            this.ws.onopen = () => {
                console.log('WebSocket connected');
                this.isConnecting = false;
                this.reconnectAttempts = 0;
            };

            this.ws.onmessage = (event) => {
                try {
                    const progress: ProgressData = JSON.parse(event.data);
                    const listener = this.listeners.get(progress.task_id);
                    if (listener) {
                        listener(progress);
                    }
                } catch (e) {
                    console.error('Failed to parse WebSocket message:', e);
                }
            };

            this.ws.onclose = (event) => {
                console.log('WebSocket closed:', event.code, event.reason);
                this.isConnecting = false;
                this.ws = null;

                // 自动重连（如果有活跃的监听器）
                if (this.listeners.size > 0 && this.reconnectAttempts < this.maxReconnectAttempts) {
                    this.reconnectAttempts++;
                    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
                    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
                    this.reconnectTimer = setTimeout(() => this.connect(), delay);
                }
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.isConnecting = false;
            };

        } catch (e) {
            console.error('Failed to create WebSocket:', e);
            this.isConnecting = false;
        }
    }

    // 订阅任务进度
    subscribe(taskId: string, callback: ProgressCallback): void {
        this.listeners.set(taskId, callback);
        console.log('Subscribing to progress updates for task:', taskId, 'via', WS_BASE_URL);

        // 确保WebSocket已连接
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            this.connect();
        }
    }

    // 取消订阅
    unsubscribe(taskId: string): void {
        this.listeners.delete(taskId);

        // 如果没有监听器了，可以选择断开连接
        if (this.listeners.size === 0) {
            this.disconnect();
        }
    }

    // 断开连接
    disconnect(): void {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }

        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }

        this.listeners.clear();
        this.reconnectAttempts = 0;
    }

    // 检查连接状态
    isConnected(): boolean {
        return this.ws?.readyState === WebSocket.OPEN;
    }
}

// 单例导出
export const wsClient = new ProgressWebSocket();
