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
    phase?: string;            // downloading_video, downloading_audio, downloading, merging, processing
    phase_label?: string;      // 中文阶段标签
    downloaded_bytes: number;
    total_bytes: number;
    speed: string;
    eta: string;
    file_path?: string;
    error_message?: string;
}

type ProgressCallback = (progress: ProgressData) => void;

class ProgressWebSocket {
    private sockets: Map<string, WebSocket> = new Map();
    private reconnectTimers: Map<string, ReturnType<typeof setTimeout>> = new Map();
    private listeners: Map<string, ProgressCallback> = new Map();
    private reconnectAttempts: Map<string, number> = new Map();
    private connectingTasks: Set<string> = new Set();
    private readonly maxReconnectAttempts = 5;

    connect(taskId: string): void {
        const existing = this.sockets.get(taskId);
        if (this.connectingTasks.has(taskId) || existing?.readyState === WebSocket.OPEN) {
            return;
        }

        const token = tokenManager.getToken();
        if (!token) {
            console.warn('No token available for WebSocket connection');
            return;
        }

        this.connectingTasks.add(taskId);
        const wsUrl = `${WS_BASE_URL}/api/v1/ws/progress?task_id=${encodeURIComponent(taskId)}`;

        try {
            const ws = new WebSocket(wsUrl, ["bearer", token]);
            this.sockets.set(taskId, ws);

            ws.onopen = () => {
                console.log('WebSocket connected for task:', taskId);
                this.connectingTasks.delete(taskId);
                this.reconnectAttempts.set(taskId, 0);
            };

            ws.onmessage = (event) => {
                try {
                    const progress: ProgressData = JSON.parse(event.data);
                    const listener = this.listeners.get(taskId) ?? this.listeners.get(progress.task_id);
                    if (listener) {
                        listener(progress);
                    }
                } catch (e) {
                    console.error('Failed to parse WebSocket message:', e);
                }
            };

            ws.onclose = (event) => {
                console.log('WebSocket closed:', taskId, event.code, event.reason);
                this.connectingTasks.delete(taskId);
                this.sockets.delete(taskId);

                if (this.listeners.has(taskId)) {
                    const attempts = this.reconnectAttempts.get(taskId) ?? 0;
                    if (attempts < this.maxReconnectAttempts) {
                        const nextAttempt = attempts + 1;
                        this.reconnectAttempts.set(taskId, nextAttempt);
                        const delay = Math.min(1000 * Math.pow(2, nextAttempt), 30000);
                        console.log(`Reconnecting task ${taskId} in ${delay}ms (attempt ${nextAttempt})`);
                        const timer = setTimeout(() => this.connect(taskId), delay);
                        this.reconnectTimers.set(taskId, timer);
                    }
                }
            };

            ws.onerror = (error) => {
                console.error('WebSocket error for task:', taskId, error);
                this.connectingTasks.delete(taskId);
            };

        } catch (e) {
            console.error('Failed to create WebSocket:', e);
            this.connectingTasks.delete(taskId);
        }
    }

    // 订阅任务进度
    subscribe(taskId: string, callback: ProgressCallback): void {
        this.listeners.set(taskId, callback);
        console.log('Subscribing to progress updates for task:', taskId, 'via', WS_BASE_URL);

        this.connect(taskId);
    }

    // 取消订阅
    unsubscribe(taskId: string): void {
        this.listeners.delete(taskId);
        this.reconnectAttempts.delete(taskId);
        this.connectingTasks.delete(taskId);

        const timer = this.reconnectTimers.get(taskId);
        if (timer) {
            clearTimeout(timer);
            this.reconnectTimers.delete(taskId);
        }

        const socket = this.sockets.get(taskId);
        if (socket) {
            socket.close();
            this.sockets.delete(taskId);
        }
    }

    // 断开连接
    disconnect(): void {
        this.reconnectTimers.forEach((timer) => clearTimeout(timer));
        this.reconnectTimers.clear();
        this.sockets.forEach((socket) => socket.close());
        this.sockets.clear();
        this.listeners.clear();
        this.reconnectAttempts.clear();
        this.connectingTasks.clear();
    }

    // 检查连接状态
    isConnected(): boolean {
        for (const socket of this.sockets.values()) {
            if (socket.readyState === WebSocket.OPEN) {
                return true;
            }
        }
        return false;
    }
}

// 单例导出
export const wsClient = new ProgressWebSocket();
