import apiClient from '../api-client';
import { formatDuration } from '../format';

export interface VideoFormat {
    format_id: string;
    quality: string;
    extension: string;
    filesize: number;
    height: number;
    fps: number;
    video_codec: string;
    audio_codec: string;
}

export interface VideoInfo {
    video_id: string;
    platform: string;
    title: string;
    description: string;
    duration: number;        // 后端返回秒数
    durationFormatted: string; // 前端格式化后
    thumbnail: string;
    author: string;
    upload_date: string;
    view_count: number;
    formats: VideoFormat[];
    url: string;             // 前端自行保存
}

export const parseApi = {
    // 解析URL
    parseUrl: async (url: string, skipCache = false): Promise<VideoInfo> => {
        const response = await apiClient.post('/api/v1/parse', { url, skip_cache: skipCache });
        const data = response.data;
        return {
            ...data,
            durationFormatted: formatDuration(data.duration),
            url, // 保存原始URL
        };
    },
};
