
export interface VideoInfo {
    title: string;
    thumbnail: string;
    duration: string;
    platform: 'YouTube' | 'Bilibili' | 'TikTok' | 'Other';
    url: string;
}

export interface User {
    id: string;
    email: string;
    nickname: string;
    avatar?: string;
    quota: number;
}

export interface DownloadHistoryItem {
    id: string;
    title: string;
    thumbnail: string;
    platform: string;
    duration: string;
    downloadedAt: string;
    type: 'video' | 'audio';
}

export const mockApi = {
    parseUrl: async (url: string): Promise<VideoInfo> => {
        return new Promise((resolve, reject) => {
            setTimeout(() => {
                if (!url.startsWith('http')) {
                    reject(new Error('Invalid URL'));
                    return;
                }
                resolve({
                    title: 'Rick Astley - Never Gonna Give You Up (Official Music Video)',
                    thumbnail: 'https://i.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg',
                    duration: '03:32',
                    platform: 'YouTube',
                    url,
                });
            }, 1500);
        });
    },

    login: async (email: string): Promise<User> => {
        return new Promise((resolve) => {
            setTimeout(() => {
                resolve({
                    id: 'user_123',
                    email,
                    nickname: email.split('@')[0],
                    quota: 20,
                    avatar: 'https://github.com/shadcn.png',
                });
            }, 1000);
        });
    },

    getHistory: async (): Promise<DownloadHistoryItem[]> => {
        return new Promise((resolve) => {
            setTimeout(() => {
                resolve([
                    {
                        id: '1',
                        title: 'Rick Astley - Never Gonna Give You Up',
                        thumbnail: 'https://i.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg',
                        platform: 'YouTube',
                        duration: '03:32',
                        downloadedAt: '2025-12-01 15:30',
                        type: 'video',
                    },
                    {
                        id: '2',
                        title: 'Awesome Bilibili Video',
                        thumbnail: 'https://i0.hdslb.com/bfs/archive/example.jpg',
                        platform: 'Bilibili',
                        duration: '10:45',
                        downloadedAt: '2025-11-30 20:15',
                        type: 'video',
                    },
                ]);
            }, 800);
        });
    },
};
