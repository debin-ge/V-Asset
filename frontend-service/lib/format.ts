/**
 * 格式化时长（秒数 → MM:SS 或 HH:MM:SS）
 */
export function formatDuration(seconds: number): string {
    if (!seconds || seconds < 0) return '00:00';

    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = Math.floor(seconds % 60);

    if (h > 0) {
        return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
    }
    return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}

/**
 * 格式化日期时间
 */
export function formatDate(isoString: string): string {
    if (!isoString) return '';
    try {
        return new Date(isoString).toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
        });
    } catch {
        return isoString;
    }
}

/**
 * 格式化文件大小
 */
export function formatFileSize(bytes: number): string {
    if (!bytes || bytes === 0) return '0 B';

    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    const size = bytes / Math.pow(1024, i);

    return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

/**
 * 格式化观看次数
 */
export function formatViewCount(count: number): string {
    if (!count) return '0';

    if (count >= 100000000) {
        return `${(count / 100000000).toFixed(1)}亿`;
    }
    if (count >= 10000) {
        return `${(count / 10000).toFixed(1)}万`;
    }
    return count.toLocaleString();
}

/**
 * 下载类型映射（前端type → 后端mode）
 */
export function mapDownloadType(type: 'video' | 'audio'): string {
    return type === 'video' ? 'quick_download' : 'audio_only';
}

/**
 * 下载状态码映射
 */
export function getStatusText(status: number): string {
    const statusMap: Record<number, string> = {
        0: '等待中',
        1: '下载中',
        2: '已完成',
        3: '失败',
        4: '待清理',
        5: '已过期',
    };
    return statusMap[status] || '未知';
}

/**
 * 平台名称标准化
 */
export function normalizePlatform(platform: string): string {
    const platformMap: Record<string, string> = {
        'youtube': 'YouTube',
        'bilibili': 'Bilibili',
        'tiktok': 'TikTok',
    };
    return platformMap[platform.toLowerCase()] || platform;
}
