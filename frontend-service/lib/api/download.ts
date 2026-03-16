import apiClient from '../api-client';
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
    mode: string;
    quality: string;
    format?: string;
    format_id?: string;
    selected_format?: SelectedFormatPayload;
}

export interface DownloadResponse {
    task_id: string;
    history_id: number;
    estimated_time: number;
}

interface FileDownloadTicketResponse {
    ticket: string;
    expires_in: number;
}

export const downloadApi = {
    submitDownload: async (params: DownloadRequest): Promise<DownloadResponse> => {
        const response = await apiClient.post('/api/v1/download', params);
        return response.data as DownloadResponse;
    },

    downloadFile: async (historyId: number): Promise<void> => {
        const response = await apiClient.post('/api/v1/download/file-ticket', {
            history_id: historyId,
        });
        const data = response.data as FileDownloadTicketResponse;
        startNativeDownload(data.ticket);
    },
};

function startNativeDownload(ticket: string) {
    const requestUrl = new URL('/api/v1/download/file/browser', resolveApiBaseUrl());
    requestUrl.searchParams.set('ticket', ticket);

    const link = document.createElement('a');
    link.href = requestUrl.toString();
    link.rel = 'noopener';
    link.style.display = 'none';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}

export function mapDownloadType(type: 'video' | 'audio'): string {
    return type === 'video' ? 'quick_download' : 'quick_download';
}
