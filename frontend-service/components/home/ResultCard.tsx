"use client"

import * as React from "react"
import { motion } from "framer-motion"
import { AlertTriangle, Download, Film, Headphones, Loader2, Wallet } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import type { VideoInfo, VideoFormat } from "@/lib/api/parse"
import { useAuth } from "@/hooks/use-auth"
import { Badge } from "@/components/ui/badge"
import { RemoteThumbnail } from "@/components/common/RemoteThumbnail"
import { billingApi, BillingEstimateResponse } from "@/lib/api/billing"
import { mapDownloadType, SelectedFormatPayload } from "@/lib/api/download"
import { formatCurrencyYuan, formatFileSize as formatSharedFileSize, parseCurrencyYuan } from "@/lib/format"
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog"

interface ResultCardProps {
    info: VideoInfo
    onDownload: (type: 'video' | 'audio', selectedFormat?: VideoFormat) => void
}

export function ResultCard({ info, onDownload }: ResultCardProps) {
    const { user, billingAccount, openAuthModal } = useAuth()
    const [pendingDownload, setPendingDownload] = React.useState<{ type: 'video' | 'audio'; format?: VideoFormat } | null>(null)
    const [estimate, setEstimate] = React.useState<BillingEstimateResponse | null>(null)
    const [isEstimateOpen, setIsEstimateOpen] = React.useState(false)
    const [isEstimating, setIsEstimating] = React.useState(false)

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

    const handleDownload = (type: 'video' | 'audio', selectedFormat?: VideoFormat) => {
        if (!user) {
            openAuthModal()
            return
        }

        setPendingDownload({ type, format: selectedFormat })
        setEstimate(null)
        setIsEstimateOpen(true)
        setIsEstimating(true)

        void billingApi.estimateDownload({
            url: info.url,
            platform: info.platform,
            mode: mapDownloadType(type),
            selected_format: toEstimateSelectedFormat(selectedFormat),
        }).then((response) => {
            setEstimate(response)
        }).catch((error) => {
            console.error("Failed to estimate billing", error)
            toast.error(error instanceof Error ? error.message : "Failed to estimate download cost")
            setIsEstimateOpen(false)
            setPendingDownload(null)
        }).finally(() => {
            setIsEstimating(false)
        })
    }

    const formatFileSize = (bytes?: number) => {
        if (!bytes || bytes === 0) return '-'
        return formatSharedFileSize(bytes)
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

    const confirmDownload = () => {
        if (!pendingDownload) {
            return
        }

        onDownload(pendingDownload.type, pendingDownload.format)
        setIsEstimateOpen(false)
        setPendingDownload(null)
        setEstimate(null)
    }

    const closeEstimateDialog = (open: boolean) => {
        setIsEstimateOpen(open)
        if (!open) {
            setPendingDownload(null)
            setEstimate(null)
            setIsEstimating(false)
        }
    }

    const availableBalanceFen = parseCurrencyYuan(billingAccount?.available_balance_fen)
    const estimatedCostFen = parseCurrencyYuan(estimate?.estimated_cost_fen)
    const insufficientBalance = !!estimate && availableBalanceFen < estimatedCostFen
    const balanceGapFen = estimate ? Math.max(estimatedCostFen - availableBalanceFen, 0) : 0
    const balanceAfterHoldFen = estimate ? Math.max(availableBalanceFen - estimatedCostFen, 0) : availableBalanceFen
    const selectedFormatLabel = describeSelectedFormat(pendingDownload?.type, pendingDownload?.format)

    return (
        <>
            <motion.div
                initial={{ y: 20, opacity: 0 }}
                animate={{ y: 0, opacity: 1 }}
                className="w-full max-w-4xl mx-auto mt-8 bg-white rounded-2xl shadow-xl overflow-hidden border border-gray-100"
            >
                {/* Video Info Header */}
                <div className="flex flex-col md:flex-row">
                    <div className="w-full md:w-48 h-32 md:h-auto relative">
                        <RemoteThumbnail
                            src={info.thumbnail}
                            alt={info.title}
                            className="w-full h-full"
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
                                                            onClick={() => handleDownload('video', format)}
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
                                            onClick={() => handleDownload('audio', format)}
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

            <Dialog open={isEstimateOpen} onOpenChange={closeEstimateDialog}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Confirm Download Billing</DialogTitle>
                        <DialogDescription>
                            Review the estimated traffic, cost, and balance impact before starting this download.
                        </DialogDescription>
                    </DialogHeader>

                    {isEstimating ? (
                        <div className="flex items-center justify-center py-10 text-sm text-gray-500">
                            <Loader2 className="mr-2 h-5 w-5 animate-spin" />
                            Calculating estimate...
                        </div>
                    ) : estimate ? (
                        <div className="space-y-4">
                            <div className="rounded-2xl bg-slate-50 p-4">
                                <div className="flex flex-wrap items-center gap-2">
                                    <Badge variant="secondary" className="text-xs">
                                        {pendingDownload?.type === "audio" ? "Audio" : "Video"}
                                    </Badge>
                                    <Badge variant="outline" className="text-xs">
                                        {selectedFormatLabel}
                                    </Badge>
                                    {estimate.is_estimated ? (
                                        <Badge className="bg-amber-500 text-white">Estimated</Badge>
                                    ) : (
                                        <Badge className="bg-emerald-500 text-white">Exact size</Badge>
                                    )}
                                </div>
                                <p className="mt-3 text-sm font-medium text-slate-900">{info.title}</p>
                                <p className="mt-1 text-sm text-slate-600">
                                    {describeSelectedFormatMeta(pendingDownload?.format)}
                                </p>
                            </div>

                            <div className="grid gap-3 md:grid-cols-2">
                                <div className="rounded-2xl border border-slate-100 p-4">
                                    <p className="text-xs uppercase tracking-wide text-slate-400">Estimated Traffic</p>
                                    <p className="mt-2 text-xl font-semibold text-slate-900">
                                        {formatSharedFileSize(estimate.estimated_traffic_bytes)}
                                    </p>
                                    <p className="mt-1 text-xs text-slate-500">
                                        Combined traffic for the complete billing flow
                                    </p>
                                </div>
                                <div className="rounded-2xl border border-slate-100 p-4">
                                    <p className="text-xs uppercase tracking-wide text-slate-400">Estimated Cost</p>
                                    <p className="mt-2 text-xl font-semibold text-slate-900">
                                        {formatCurrencyYuan(estimate.estimated_cost_fen)}
                                    </p>
                                    <p className="mt-1 text-xs text-slate-500">
                                        Pricing version #{estimate.pricing_version}
                                    </p>
                                </div>
                            </div>

                            <div className="grid gap-3 md:grid-cols-3">
                                <div className="rounded-2xl border border-slate-100 p-4">
                                    <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-slate-400">
                                        <Wallet className="h-3.5 w-3.5" />
                                        Available now
                                    </div>
                                    <p className="mt-2 text-lg font-semibold text-slate-900">
                                        {formatCurrencyYuan(availableBalanceFen)}
                                    </p>
                                </div>
                                <div className="rounded-2xl border border-slate-100 p-4">
                                    <p className="text-xs uppercase tracking-wide text-slate-400">Reserved after submit</p>
                                    <p className="mt-2 text-lg font-semibold text-slate-900">
                                        {formatCurrencyYuan(estimatedCostFen)}
                                    </p>
                                </div>
                                <div className="rounded-2xl border border-slate-100 p-4">
                                    <p className="text-xs uppercase tracking-wide text-slate-400">Remaining available</p>
                                    <p className="mt-2 text-lg font-semibold text-slate-900">
                                        {formatCurrencyYuan(balanceAfterHoldFen)}
                                    </p>
                                </div>
                            </div>

                            {estimate.is_estimated ? (
                                <div className="rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
                                    {describeEstimateReason(estimate.estimate_reason)}
                                </div>
                            ) : null}
                            {insufficientBalance ? (
                                <div className="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                                    <div className="flex items-start gap-2">
                                        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                                        <div>
                                            <p className="font-medium">Insufficient balance</p>
                                            <p className="mt-1">
                                                You need {formatCurrencyYuan(balanceGapFen)} more to cover this billing hold.
                                                Current available balance: {formatCurrencyYuan(availableBalanceFen)}.
                                            </p>
                                        </div>
                                    </div>
                                </div>
                            ) : null}
                        </div>
                    ) : null}

                    <DialogFooter>
                        <Button variant="outline" onClick={() => closeEstimateDialog(false)}>
                            Cancel
                        </Button>
                        <Button onClick={confirmDownload} disabled={isEstimating || !estimate || insufficientBalance}>
                            {insufficientBalance ? "Balance Too Low" : "Start Download"}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </>
    )
}

function toEstimateSelectedFormat(format?: VideoFormat): SelectedFormatPayload | undefined {
    if (!format) {
        return undefined
    }

    return {
        format_id: format.format_id,
        quality: format.quality,
        extension: format.extension,
        filesize: format.filesize,
        height: format.height,
        width: format.width,
        fps: format.fps,
        video_codec: format.video_codec,
        audio_codec: format.audio_codec,
        vbr: format.vbr,
        abr: format.abr,
        asr: format.asr,
    }
}

function describeSelectedFormat(type: "video" | "audio" | undefined, format?: VideoFormat) {
    if (!format) {
        return type === "audio" ? "Default audio" : "Default quality"
    }
    if (type === "audio") {
        if (format.abr) {
            return `${Math.round(format.abr)} kbps`
        }
        return format.extension?.toUpperCase() || "Audio"
    }
    if (format.height) {
        return `${format.height}p`
    }
    return format.quality || format.extension?.toUpperCase() || "Video"
}

function describeSelectedFormatMeta(format?: VideoFormat) {
    if (!format) {
        return "The platform will use its default delivery format."
    }

    const parts = [
        format.extension ? format.extension.toUpperCase() : "",
        format.width && format.height ? `${format.width}×${format.height}` : format.height ? `${format.height}p` : "",
        format.fps ? `${format.fps} FPS` : "",
        format.filesize ? formatSharedFileSize(format.filesize) : "",
    ].filter(Boolean)

    return parts.join(" • ") || "Selected format details are unavailable."
}

function describeEstimateReason(reason?: string) {
    if (reason === "unknown_filesize") {
        return "The exact file size is not available yet, so the pre-submit estimate stays at 0. Final billing will use the real size after the server download completes."
    }
    return "The platform used an estimated size because the exact file size was unavailable."
}
