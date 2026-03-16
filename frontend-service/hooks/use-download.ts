"use client"

import * as React from "react"
import { AxiosError } from "axios"
import { toast } from "sonner"
import { parseApi } from "@/lib/api/parse"
import type { VideoFormat, VideoInfo } from "@/lib/api/parse"
import { downloadApi, mapDownloadType } from "@/lib/api/download"
import type { DownloadRequest } from "@/lib/api/download"
import { wsClient, ProgressData } from "@/lib/ws-client"
import { useAuth } from "@/hooks/use-auth"

export type DownloadStatus = "idle" | "parsing" | "parsed" | "downloading" | "completed" | "error"

export function useDownload() {
    const { refreshBillingAccount } = useAuth()
    const [url, setUrl] = React.useState("")
    const [status, setStatus] = React.useState<DownloadStatus>("idle")
    const [videoInfo, setVideoInfo] = React.useState<VideoInfo | null>(null)
    const [progress, setProgress] = React.useState(0)
    const [speed, setSpeed] = React.useState("0 MB/s")
    const [timeLeft, setTimeLeft] = React.useState("")
    const [currentTaskId, setCurrentTaskId] = React.useState<string | null>(null)
    const [historyId, setHistoryId] = React.useState<number | null>(null)
    const [autoDownloadAttempted, setAutoDownloadAttempted] = React.useState(false)
    const [phase, setPhase] = React.useState<string>("")
    const [phaseLabel, setPhaseLabel] = React.useState<string>("")

    // Use ref to store historyId for access in callbacks
    const historyIdRef = React.useRef<number | null>(null)
    React.useEffect(() => {
        historyIdRef.current = historyId
    }, [historyId])

    // Cleanup WebSocket subscription
    React.useEffect(() => {
        return () => {
            if (currentTaskId) {
                wsClient.unsubscribe(currentTaskId)
            }
        }
    }, [currentTaskId])

    // Auto download file to browser
    const triggerFileDownload = React.useCallback(async (hId: number) => {
        try {
            setAutoDownloadAttempted(true)
            await downloadApi.downloadFile(hId)
            scheduleBillingRefresh(refreshBillingAccount)
            toast.success("Browser download started")
        } catch (error) {
            await refreshBillingAccount()
            console.error("[Download] Automatic download did not start", error)
            toast.info("Automatic download failed, please click the button below to download manually")
        }
    }, [refreshBillingAccount])

    // Handle progress update
    const handleProgress = React.useCallback((data: ProgressData) => {
        setProgress(data.percent)
        setSpeed(data.speed || "0 MB/s")
        setTimeLeft(data.eta || "")
        if (data.phase) setPhase(data.phase)
        if (data.phase_label) setPhaseLabel(data.phase_label)

        // 支持字符串和数字两种 status 格式
        const isCompleted = data.status === 2 || data.status === "completed" || data.status_text === "completed"
        const isFailed = data.status === 3 || data.status === "failed" || data.status_text === "failed"

        if (isCompleted) {
            setPhase("transferring")
            setPhaseLabel("Starting browser download...")
            setProgress(100)
            if (currentTaskId) {
                wsClient.unsubscribe(currentTaskId)
            }
            // Auto trigger file download to browser
            const hId = historyIdRef.current
            if (hId) {
                setTimeout(() => {
                    triggerFileDownload(hId).finally(() => {
                        setStatus("completed")
                    })
                }, 1000)
            } else {
                toast.error("Unable to get download file information, please re-submit the download task")
                setStatus("completed")
            }
        } else if (isFailed) {
            setStatus("error")
            toast.error(data.error_message || "Download failed")
            if (currentTaskId) {
                wsClient.unsubscribe(currentTaskId)
            }
        }
    }, [currentTaskId, triggerFileDownload])

    // Parse URL
    const handleParse = async (inputUrl: string) => {
        if (!inputUrl) return
        setStatus("parsing")
        setUrl(inputUrl)
        try {
            const info = await parseApi.parseUrl(inputUrl)
            setVideoInfo(info)
            setStatus("parsed")
        } catch (error) {
            setStatus("error")
            let message = error instanceof Error ? error.message : "Parse failed, please check the link"
            if (error instanceof AxiosError && error.code === "ECONNABORTED") {
                message = "Parsing took too long, please try again later or contact the administrator to increase the parsing timeout"
            }
            toast.error(message)
        }
    }

    // Start download
    const startDownload = async (type: 'video' | 'audio', selectedFormat?: VideoFormat) => {
        if (!videoInfo) return

        setStatus("downloading")
        setProgress(0)

        try {
            const quality = getSelectedQuality(type, selectedFormat)
            const outputFormat = selectedFormat?.extension || (type === "audio" ? "m4a" : "mp4")
            const downloadParams: DownloadRequest = {
                url: videoInfo.url,
                mode: mapDownloadType(type),
                quality,
                format: outputFormat,
            }

            if (selectedFormat) {
                downloadParams.format_id = selectedFormat.format_id
                downloadParams.selected_format = {
                    format_id: selectedFormat.format_id,
                    quality: quality,
                    extension: selectedFormat.extension,
                    filesize: selectedFormat.filesize,
                    height: selectedFormat.height,
                    width: selectedFormat.width,
                    fps: selectedFormat.fps,
                    video_codec: selectedFormat.video_codec,
                    audio_codec: selectedFormat.audio_codec,
                    vbr: selectedFormat.vbr,
                    abr: selectedFormat.abr,
                    asr: selectedFormat.asr,
                }
            }

            const response = await downloadApi.submitDownload(downloadParams)

            setCurrentTaskId(response.task_id)
            setHistoryId(response.history_id)
            await refreshBillingAccount()

            // Subscribe to progress
            wsClient.subscribe(response.task_id, handleProgress)

            toast.info(`Download task submitted`)
        } catch (error) {
            setStatus("error")
            await refreshBillingAccount()
            const message = error instanceof Error ? error.message : "Failed to submit download task"
            toast.error(message)
        }
    }

    // Download file locally
    const downloadFile = async () => {
        if (!historyId) {
            toast.error("No file available for download")
            return
        }

        try {
            await downloadApi.downloadFile(historyId)
            scheduleBillingRefresh(refreshBillingAccount)
            toast.success("Browser download started")
        } catch (error) {
            await refreshBillingAccount()
            const message = error instanceof Error ? error.message : "File download failed"
            toast.error(message)
        }
    }

    // Reset state
    const reset = () => {
        if (currentTaskId) {
            wsClient.unsubscribe(currentTaskId)
        }
        setUrl("")
        setStatus("idle")
        setVideoInfo(null)
        setProgress(0)
        setSpeed("0 MB/s")
        setTimeLeft("")
        setCurrentTaskId(null)
        setHistoryId(null)
        setAutoDownloadAttempted(false)
        setPhase("")
        setPhaseLabel("")
    }

    return {
        url,
        setUrl,
        status,
        videoInfo,
        progress,
        speed,
        timeLeft,
        handleParse,
        startDownload,
        downloadFile,
        reset,
        historyId,
        autoDownloadAttempted,
        phase,
        phaseLabel,
    }
}

function scheduleBillingRefresh(refreshBillingAccount: () => Promise<void>) {
    void refreshBillingAccount()
    window.setTimeout(() => {
        void refreshBillingAccount()
    }, 2000)
    window.setTimeout(() => {
        void refreshBillingAccount()
    }, 10000)
}

function getSelectedQuality(type: 'video' | 'audio', selectedFormat?: VideoFormat): string {
    if (!selectedFormat) {
        return type === "audio" ? "audio" : "best"
    }

    if (type === "audio") {
        if (selectedFormat.abr && selectedFormat.abr > 0) {
            return `${Math.round(selectedFormat.abr)}kbps`
        }
        if (selectedFormat.asr && selectedFormat.asr > 0) {
            return `${(selectedFormat.asr / 1000).toFixed(1)}kHz`
        }
    }

    if (selectedFormat.quality && selectedFormat.quality !== "audio") {
        return selectedFormat.quality
    }

    if (selectedFormat.height && selectedFormat.height > 0) {
        return `${selectedFormat.height}p`
    }

    return selectedFormat.extension || (type === "audio" ? "audio" : "best")
}
