import apiClient from '../api-client';
import { formatDuration } from '../format';

export interface VideoFormat {
    format_id: string;
    quality: string;
    extension: string;
    filesize: number;
    height: number;
    width?: number;
    fps: number;
    video_codec: string;
    audio_codec: string;
    vbr?: number;        // 视频码率 (kbps)
    abr?: number;        // 音频码率 (kbps)
    tbr?: number;        // 总码率 (kbps)
    asr?: number;        // 音频采样率 (Hz)
    format_note?: string; // 格式说明，如 "1080p", "medium"
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

        // 调试日志
        console.log('[DEBUG-parseUrl] Received from API:', {
            formatsCount: data.formats?.length || 0,
            videoFormats: data.formats?.filter((f: any) => f.video_codec && f.video_codec !== 'none').length || 0,
            audioFormats: data.formats?.filter((f: any) => f.audio_codec && f.audio_codec !== 'none' && (!f.video_codec || f.video_codec === 'none')).length || 0,
            maxHeight: Math.max(...(data.formats?.map((f: any) => f.height || 0) || [0])),
            first3Formats: data.formats?.slice(0, 3).map((f: any) => ({
                format_id: f.format_id,
                height: f.height,
                video_codec: f.video_codec,
                audio_codec: f.audio_codec,
                filesize: f.filesize
            }))
        });

        return {
            ...data,
            durationFormatted: formatDuration(data.duration),
            url, // 保存原始URL
        };
    },
};
