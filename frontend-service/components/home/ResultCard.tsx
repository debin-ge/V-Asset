"use client"

import * as React from "react"
import { motion } from "framer-motion"
import { Download, Film, Headphones } from "lucide-react"
import { Button } from "@/components/ui/button"
import { VideoInfo, VideoFormat } from "@/lib/api/parse"
import { useAuth } from "@/hooks/use-auth"
import { Badge } from "@/components/ui/badge"

interface ResultCardProps {
    info: VideoInfo
    onDownload: (type: 'video' | 'audio', formatId?: string) => void
}

export function ResultCard({ info, onDownload }: ResultCardProps) {
    const { user, openAuthModal } = useAuth()

    // Filter video formats (has video_codec, regardless of audio_codec)
    const videoFormats = info.formats?.filter(f =>
        f.video_codec &&
        f.video_codec !== 'none'
    ) || []

    // Filter pure audio formats (only audio_codec, no video_codec)
    const audioFormats = info.formats?.filter(f =>
        f.audio_codec &&
        f.audio_codec !== 'none' &&
        (!f.video_codec || f.video_codec === 'none')
    ) || []

    // 调试日志
    console.log('[DEBUG-ResultCard] Formats:', {
        totalFormats: info.formats?.length || 0,
        videoFormats: videoFormats.length,
        audioFormats: audioFormats.length,
        maxVideoHeight: Math.max(...videoFormats.map(f => f.height || 0), 0),
        videoDetails: videoFormats.slice(0, 3).map(f => ({
            format_id: f.format_id,
            height: f.height,
            video_codec: f.video_codec,
            filesize: f.filesize
        })),
        audioDetails: audioFormats.slice(0, 3).map(f => ({
            format_id: f.format_id,
            audio_codec: f.audio_codec,
            abr: f.abr,
            filesize: f.filesize
        }))
    });

    // Determine default tab based on available formats
    const getDefaultTab = (): 'video' | 'audio' => {
        if (videoFormats.length > 0) return 'video'
        if (audioFormats.length > 0) return 'audio'
        return 'video'
    }

    const [selectedTab, setSelectedTab] = React.useState<'video' | 'audio'>(getDefaultTab())

    // Sort video formats by resolution descending, then by extension
    const sortedVideoFormats = [...videoFormats].sort((a, b) => {
        const heightDiff = (b.height || 0) - (a.height || 0)
        if (heightDiff !== 0) return heightDiff
        // Same resolution, sort by extension (mp4 first)
        if (a.extension === 'mp4' && b.extension !== 'mp4') return -1
        if (b.extension === 'mp4' && a.extension !== 'mp4') return 1
        return 0
    })

    // Sort audio formats by bitrate descending
    const sortedAudioFormats = [...audioFormats].sort((a, b) => (b.abr || 0) - (a.abr || 0))

    // Group video formats by resolution
    const groupedVideoFormats = sortedVideoFormats.reduce((acc, format) => {
        const resolution = format.height ? `${format.height}p` : 'Unknown'
        if (!acc[resolution]) {
            acc[resolution] = []
        }
        acc[resolution].push(format)
        return acc
    }, {} as Record<string, VideoFormat[]>)

    const handleDownload = (type: 'video' | 'audio', formatId?: string) => {
        if (!user) {
            openAuthModal()
            return
        }
        onDownload(type, formatId)
    }

    const formatFileSize = (bytes?: number) => {
        if (!bytes || bytes === 0) return '-'
        const mb = bytes / (1024 * 1024)
        if (mb < 1) {
            return `${(bytes / 1024).toFixed(1)}KB`
        }
        if (mb < 1024) {
            return `${mb.toFixed(1)}MB`
        }
        return `${(mb / 1024).toFixed(2)}GB`
    }

    const formatBitrate = (kbps?: number) => {
        if (!kbps || kbps === 0) return ''
        if (kbps < 1000) {
            return `${Math.round(kbps)}kbps`
        }
        return `${(kbps / 1000).toFixed(1)}Mbps`
    }

    const getCodecDisplayName = (codec?: string) => {
        if (!codec || codec === 'none') return ''
        // Simplify codec name display
        if (codec.startsWith('avc1')) return 'H.264'
        if (codec.startsWith('av01')) return 'AV1'
        if (codec === 'vp9' || codec.startsWith('vp09')) return 'VP9'
        if (codec === 'opus') return 'Opus'
        if (codec.startsWith('mp4a')) return 'AAC'
        if (codec.startsWith('vp8')) return 'VP8'
        return codec.toUpperCase()
    }

    const getExtensionDisplayName = (ext?: string) => {
        if (!ext) return '-'
        return ext.toUpperCase()
    }

    const getQualityBadgeColor = (height?: number) => {
        if (!height) return 'bg-gray-500 text-white'
        if (height >= 2160) return 'bg-purple-500 text-white'
        if (height >= 1440) return 'bg-indigo-500 text-white'
        if (height >= 1080) return 'bg-blue-500 text-white'
        if (height >= 720) return 'bg-green-500 text-white'
        if (height >= 480) return 'bg-yellow-500 text-white'
        return 'bg-gray-500 text-white'
    }

    const getAudioQualityColor = (abr?: number) => {
        if (!abr) return 'bg-gray-500 text-white'
        if (abr >= 128) return 'bg-green-500 text-white'
        if (abr >= 64) return 'bg-yellow-500 text-white'
        return 'bg-gray-500 text-white'
    }

    const formatResolution = (width?: number, height?: number) => {
        if (width && height) return `${width}×${height}`
        if (height) return `${height}p`
        return '-'
    }

    return (
        <motion.div
            initial={{ y: 20, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            className="w-full max-w-4xl mx-auto mt-8 bg-white rounded-2xl shadow-xl overflow-hidden border border-gray-100"
        >
            {/* Video Info Header */}
            <div className="flex flex-col md:flex-row">
                <div className="w-full md:w-48 h-32 md:h-auto relative">
                    <img
                        src={info.thumbnail}
                        alt={info.title}
                        className="w-full h-full object-cover"
                    />
                    <div className="absolute bottom-2 right-2 bg-black/70 text-white text-xs px-2 py-1 rounded">
                        {info.durationFormatted}
                    </div>
                </div>
                <div className="p-4 flex-1">
                    <div className="flex items-center gap-2 mb-2 flex-wrap">
                        <Badge variant="secondary" className="text-xs">
                            {info.platform}
                        </Badge>
                        <span className="text-xs text-gray-500">
                            {videoFormats.length} video • {audioFormats.length} audio
                        </span>
                    </div>
                    <h3 className="font-semibold text-lg leading-tight line-clamp-2">
                        {info.title}
                    </h3>
                </div>
            </div>

            {/* Tab Switcher - Always visible */}
            <div className="border-t border-gray-100 bg-gray-50">
                <div className="flex border-b border-gray-200">
                    <button
                        onClick={() => setSelectedTab('video')}
                        className={`flex-1 px-4 py-3 font-medium transition-colors flex items-center justify-center gap-2 ${selectedTab === 'video'
                            ? 'bg-white text-blue-600 border-b-2 border-blue-600'
                            : 'text-gray-600 hover:bg-gray-100'
                            }`}
                    >
                        <Film className="w-4 h-4" />
                        Video ({videoFormats.length})
                    </button>
                    <button
                        onClick={() => setSelectedTab('audio')}
                        className={`flex-1 px-4 py-3 font-medium transition-colors flex items-center justify-center gap-2 ${selectedTab === 'audio'
                            ? 'bg-white text-green-600 border-b-2 border-green-600'
                            : 'text-gray-600 hover:bg-gray-100'
                            }`}
                    >
                        <Headphones className="w-4 h-4" />
                        Audio ({audioFormats.length})
                    </button>
                </div>

                {/* Format List - Always visible */}
                <div className="p-4 max-h-[500px] overflow-y-auto">
                    {/* Video formats - grouped by resolution */}
                    {selectedTab === 'video' && (
                        <div className="space-y-4">
                            {Object.entries(groupedVideoFormats)
                                .sort(([a], [b]) => parseInt(b) - parseInt(a))
                                .map(([resolution, formats]) => (
                                    <div key={resolution} className="space-y-2">
                                        <div className="flex items-center gap-2">
                                            <Badge className={getQualityBadgeColor(formats[0].height)}>
                                                {resolution}
                                            </Badge>
                                            <span className="text-xs text-gray-500">
                                                {formatResolution(formats[0].width, formats[0].height)} • {formats[0].fps || 30} FPS
                                            </span>
                                        </div>
                                        <div className="grid gap-2">
                                            {formats.map((format) => (
                                                <div
                                                    key={format.format_id}
                                                    className="flex items-center justify-between p-3 bg-white rounded-lg border border-gray-200 hover:border-blue-300 hover:shadow-sm transition-all"
                                                >
                                                    <div className="flex items-center gap-3 flex-1 flex-wrap">
                                                        <div className="flex items-center gap-2">
                                                            {getCodecDisplayName(format.video_codec) && (
                                                                <Badge variant="outline" className="text-xs font-mono">
                                                                    {getCodecDisplayName(format.video_codec)}
                                                                </Badge>
                                                            )}
                                                            {getCodecDisplayName(format.audio_codec) && (
                                                                <Badge variant="outline" className="text-xs font-mono">
                                                                    {getCodecDisplayName(format.audio_codec)}
                                                                </Badge>
                                                            )}
                                                            <Badge variant="outline" className="text-xs">
                                                                {getExtensionDisplayName(format.extension)}
                                                            </Badge>
                                                        </div>
                                                        <span className="text-sm text-gray-600">
                                                            {formatFileSize(format.filesize)}
                                                        </span>
                                                        {format.vbr ? (
                                                            <span className="text-xs text-gray-400">
                                                                {formatBitrate(format.vbr)}
                                                            </span>
                                                        ) : null}
                                                    </div>
                                                    <Button
                                                        size="sm"
                                                        onClick={() => handleDownload('video', format.format_id)}
                                                        className="bg-blue-600 hover:bg-blue-700 text-white"
                                                    >
                                                        <Download className="w-3 h-3 mr-1" />
                                                        Download
                                                    </Button>
                                                </div>
                                            ))}
                                        </div>
                                    </div>
                                ))}
                            {videoFormats.length === 0 && (
                                <div className="text-center text-gray-500 py-8">
                                    No video formats available
                                </div>
                            )}
                        </div>
                    )}

                    {/* Audio formats */}
                    {selectedTab === 'audio' && (
                        <div className="space-y-2">
                            {sortedAudioFormats.map((format) => (
                                <div
                                    key={format.format_id}
                                    className="flex items-center justify-between p-3 bg-white rounded-lg border border-gray-200 hover:border-green-300 hover:shadow-sm transition-all"
                                >
                                    <div className="flex items-center gap-3 flex-1 flex-wrap">
                                        <Badge className={getAudioQualityColor(format.abr)}>
                                            {format.abr ? formatBitrate(format.abr) : '-'}
                                        </Badge>
                                        <div className="flex items-center gap-2">
                                            {getCodecDisplayName(format.audio_codec) && (
                                                <Badge variant="outline" className="text-xs font-mono">
                                                    {getCodecDisplayName(format.audio_codec)}
                                                </Badge>
                                            )}
                                            <Badge variant="outline" className="text-xs">
                                                {getExtensionDisplayName(format.extension)}
                                            </Badge>
                                        </div>
                                        <span className="text-sm text-gray-600">
                                            {formatFileSize(format.filesize)}
                                        </span>
                                        {format.asr ? (
                                            <span className="text-xs text-gray-400">
                                                {(format.asr / 1000).toFixed(1)}kHz
                                            </span>
                                        ) : null}
                                    </div>
                                    <Button
                                        size="sm"
                                        onClick={() => handleDownload('audio', format.format_id)}
                                        className="bg-green-600 hover:bg-green-700 text-white"
                                    >
                                        <Download className="w-3 h-3 mr-1" />
                                        Download
                                    </Button>
                                </div>
                            ))}
                            {audioFormats.length === 0 && (
                                <div className="text-center text-gray-500 py-8">
                                    No audio formats available
                                </div>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </motion.div>
    )
}
