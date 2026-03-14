import apiClient from '../api-client';

export interface SelectedFormatPayload {
    format_id: string;
    quality?: string;
    extension?: string;
    filesize?: number;
    height?: number;
    width?: number;
    fps?: number;
    video_codec?: string;
    audio_codec?: string;
    vbr?: number;
    abr?: number;
    asr?: number;
}

export interface DownloadRequest {
    url: string;
    mode: string;       // 'quick_download' | 'archive'
    quality: string;    // '1080p' | '720p' | '160kbps'
    format?: string;
    format_id?: string;
    selected_format?: SelectedFormatPayload;
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

    // 下载文件（通过带鉴权头的请求避免把 bearer token 暴露到 URL）
    downloadFile: async (historyId: number): Promise<void> => {
        const response = await apiClient.get('/api/v1/download/file', {
            params: { history_id: historyId },
            responseType: 'blob',
        });

        const contentDisposition = response.headers['content-disposition'] as string | undefined;
        const filename = parseDownloadFilename(contentDisposition);
        const blobUrl = window.URL.createObjectURL(response.data as Blob);
        const link = document.createElement('a');
        link.href = blobUrl;
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        window.URL.revokeObjectURL(blobUrl);
    },
};

function parseDownloadFilename(contentDisposition?: string): string {
    if (!contentDisposition) {
        return 'download';
    }

    const encodedMatch = contentDisposition.match(/filename\*\s*=\s*([^;]+)/i);
    if (encodedMatch) {
        const rawValue = encodedMatch[1].trim().replace(/^"(.*)"$/, '$1');
        const encodedPart = rawValue.includes("''") ? rawValue.split("''").slice(1).join("''") : rawValue;

        try {
            return decodeURIComponent(encodedPart);
        } catch {
            return encodedPart;
        }
    }

    const quotedMatch = contentDisposition.match(/filename\s*=\s*"([^"]+)"/i);
    if (quotedMatch) {
        return quotedMatch[1];
    }

    const plainMatch = contentDisposition.match(/filename\s*=\s*([^;]+)/i);
    if (plainMatch) {
        return plainMatch[1].trim();
    }

    return 'download';
}

// 下载类型映射
export function mapDownloadType(type: 'video' | 'audio'): string {
    return type === 'video' ? 'quick_download' : 'quick_download';
}
