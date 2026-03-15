import apiClient, { tokenManager } from '../api-client';
import { resolveApiBaseUrl } from '../runtime-config';

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

    // 下载文件（使用 fetch 避免 axios 超时中断大文件下载）
    downloadFile: async (historyId: number): Promise<void> => {
        const token = tokenManager.getToken();
        if (!token) {
            throw new Error('User not authenticated');
        }

        const requestUrl = new URL('/api/v1/download/file', resolveApiBaseUrl());
        requestUrl.searchParams.set('history_id', String(historyId));

        const response = await fetch(requestUrl.toString(), {
            method: 'GET',
            headers: {
                Authorization: `Bearer ${token}`,
            },
        });
        if (!response.ok) {
            throw await toDownloadError(response);
        }

        const blob = await response.blob();
        if (blob.size === 0) {
            throw new Error('Downloaded file is empty');
        }

        const contentDisposition = response.headers.get('content-disposition') ?? undefined;
        const filename = parseDownloadFilename(contentDisposition);
        triggerBrowserDownload(blob, filename);
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

async function toDownloadError(response: Response): Promise<Error> {
    const contentType = response.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
        try {
            const payload = await response.json() as { message?: string };
            if (payload?.message) {
                return new Error(payload.message);
            }
        } catch {
            // fall through to generic error handling
        }
    }

    try {
        const text = await response.text();
        if (text) {
            return new Error(text);
        }
    } catch {
        // ignore read failure
    }

    return new Error(`Download failed (${response.status})`);
}

function triggerBrowserDownload(blob: Blob, filename: string) {
    const blobUrl = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = blobUrl;
    link.download = filename;
    link.rel = 'noopener';
    link.style.display = 'none';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);

    // 某些浏览器在 click 后立即 revoke 会导致文件没有真正开始落盘。
    window.setTimeout(() => {
        window.URL.revokeObjectURL(blobUrl);
    }, 60_000);
}

// 下载类型映射
export function mapDownloadType(type: 'video' | 'audio'): string {
    return type === 'video' ? 'quick_download' : 'quick_download';
}
