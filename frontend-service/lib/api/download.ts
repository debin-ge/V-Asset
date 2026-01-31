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

export interface StreamDownloadParams {
    url: string;
    format_id?: string;
    name: string;
    ext: string;
    is_video: boolean;
}

export const downloadApi = {
    // 流式下载 - 直接触发浏览器下载
    streamDownload: (params: StreamDownloadParams): Promise<void> => {
        const baseUrl = process.env.NEXT_PUBLIC_API_BASE_URL || '';
        const token = localStorage.getItem('v-asset-token') || '';

        const queryParams = new URLSearchParams({
            url: params.url,
            name: params.name,
            ext: params.ext,
            is_video: params.is_video.toString(),
        });

        if (params.format_id) {
            queryParams.set('format_id', params.format_id);
        }

        // 创建带 token 的下载链接
        const downloadUrl = `${baseUrl}/api/v1/stream?${queryParams.toString()}`;

        // 使用 fetch 下载以携带 Authorization header
        return fetch(downloadUrl, {
            headers: {
                'Authorization': `Bearer ${token}`,
            },
        })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`Download failed: ${response.status}`);
                }

                // 获取文件名
                const contentDisposition = response.headers.get('content-disposition');
                let filename = params.name + '.' + params.ext;
                if (contentDisposition) {
                    const match = contentDisposition.match(/filename\*?=['"]?(?:UTF-8'')?([^'\";\n]+)/i);
                    if (match) filename = decodeURIComponent(match[1]);
                }

                return response.blob().then(blob => ({ blob, filename }));
            })
            .then(({ blob, filename }) => {
                // 创建下载链接
                const url = window.URL.createObjectURL(blob);
                const link = document.createElement('a');
                link.href = url;
                link.setAttribute('download', filename);
                document.body.appendChild(link);
                link.click();
                link.remove();
                window.URL.revokeObjectURL(url);
            })
            .catch(error => {
                console.error('Stream download error:', error);
                throw error;
            });
    },

    // 下载文件（blob流）- 旧接口，保留兼容
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

