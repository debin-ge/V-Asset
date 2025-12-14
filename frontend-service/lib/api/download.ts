import apiClient from '../api-client';

export interface DownloadRequest {
    url: string;
    mode: string;       // 'quick_download' | 'audio_only'
    quality: string;    // '1080p' | '720p' | 'best'
    format?: string;
}

export interface DownloadResponse {
    task_id: string;
    history_id: number;
    estimated_time: number;
}

export const downloadApi = {
    // 提交下载任务
    submitDownload: async (params: DownloadRequest): Promise<DownloadResponse> => {
        const response = await apiClient.post('/api/v1/download', params);
        return response.data as DownloadResponse;
    },

    // 下载文件（blob流）
    downloadFile: async (historyId: number): Promise<void> => {
        const response = await apiClient.get('/api/v1/download/file', {
            params: { history_id: historyId },
            responseType: 'blob',
        });

        // 从响应头获取文件名
        const contentDisposition = response.headers['content-disposition'];
        let filename = 'download';
        if (contentDisposition) {
            const match = contentDisposition.match(/filename="(.+)"/);
            if (match) filename = match[1];
        }

        // 创建下载链接
        const blob = new Blob([response.data]);
        const url = window.URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.setAttribute('download', filename);
        document.body.appendChild(link);
        link.click();
        link.remove();
        window.URL.revokeObjectURL(url);
    },
};

// 下载类型映射
export function mapDownloadType(type: 'video' | 'audio'): string {
    return type === 'video' ? 'quick_download' : 'audio_only';
}
