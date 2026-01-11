/**
 * Format duration (seconds → MM:SS or HH:MM:SS)
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
 * Format date time
 */
export function formatDate(isoString: string): string {
    if (!isoString) return '';
    try {
        return new Date(isoString).toLocaleString('en-US', {
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
 * Format file size
 */
export function formatFileSize(bytes: number): string {
    if (!bytes || bytes === 0) return '0 B';

    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    const size = bytes / Math.pow(1024, i);

    return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

/**
 * Format view count
 */
export function formatViewCount(count: number): string {
    if (!count) return '0';

    if (count >= 100000000) {
        return `${(count / 100000000).toFixed(1)}B`;
    }
    if (count >= 1000000) {
        return `${(count / 1000000).toFixed(1)}M`;
    }
    if (count >= 1000) {
        return `${(count / 1000).toFixed(1)}K`;
    }
    return count.toLocaleString();
}

/**
 * Download type mapping (frontend type → backend mode)
 */
export function mapDownloadType(type: 'video' | 'audio'): string {
    return type === 'video' ? 'quick_download' : 'audio_only';
}

/**
 * Download status code mapping
 */
export function getStatusText(status: number): string {
    const statusMap: Record<number, string> = {
        0: 'Pending',
        1: 'Downloading',
        2: 'Completed',
        3: 'Failed',
        4: 'Cleanup',
        5: 'Expired',
    };
    return statusMap[status] || 'Unknown';
}

/**
 * Normalize platform name
 */
export function normalizePlatform(platform: string): string {
    const platformMap: Record<string, string> = {
        'youtube': 'YouTube',
        'bilibili': 'Bilibili',
        'tiktok': 'TikTok',
    };
    return platformMap[platform.toLowerCase()] || platform;
}
