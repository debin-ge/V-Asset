"use client"

import * as React from "react"
import { toast } from "sonner"
import { parseApi, VideoInfo } from "@/lib/api/parse"
import { downloadApi, mapDownloadType } from "@/lib/api/download"
import { wsClient, ProgressData } from "@/lib/ws-client"
import { formatFileSize } from "@/lib/format"

export type DownloadStatus = "idle" | "parsing" | "parsed" | "downloading" | "completed" | "error"

export function useDownload() {
    const [url, setUrl] = React.useState("")
    const [status, setStatus] = React.useState<DownloadStatus>("idle")
    const [videoInfo, setVideoInfo] = React.useState<VideoInfo | null>(null)
    const [progress, setProgress] = React.useState(0)
    const [speed, setSpeed] = React.useState("0 MB/s")
    const [timeLeft, setTimeLeft] = React.useState("")
    const [currentTaskId, setCurrentTaskId] = React.useState<string | null>(null)
    const [historyId, setHistoryId] = React.useState<number | null>(null)

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
            await downloadApi.downloadFile(hId)
            toast.success("文件下载已开始")
        } catch (error) {
            const message = error instanceof Error ? error.message : "文件下载失败"
            toast.error(message)
        }
    }, [])

    // Handle progress update
    const handleProgress = React.useCallback((data: ProgressData) => {
        setProgress(data.percent)
        setSpeed(data.speed || "0 MB/s")
        setTimeLeft(data.eta || "")

        // 支持字符串和数字两种 status 格式
        const isCompleted = data.status === 2 || data.status === "completed" || data.status_text === "completed"
        const isFailed = data.status === 3 || data.status === "failed" || data.status_text === "failed"

        if (isCompleted) {
            setStatus("completed")
            toast.success("服务端下载完成！正在传输到本地...")
            if (currentTaskId) {
                wsClient.unsubscribe(currentTaskId)
            }
            // Auto trigger file download to browser
            const hId = historyIdRef.current
            console.log('[Download] Triggering file download, historyId:', hId)
            if (hId) {
                triggerFileDownload(hId)
            } else {
                console.error('[Download] historyId is null, cannot trigger download')
                toast.error("无法获取下载文件信息，请在历史记录中手动下载")
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
            const message = error instanceof Error ? error.message : "Parse failed, please check the link"
            toast.error(message)
        }
    }

    // Start download
    const startDownload = async (type: 'video' | 'audio', formatId?: string) => {
        if (!videoInfo) return

        setStatus("downloading")
        setProgress(0)

        try {
            const downloadParams: any = {
                url: videoInfo.url,
                mode: mapDownloadType(type),
                quality: "best",
                format: "mp4", // 默认使用 mp4 格式
            }

            // Add format_id to request params if specified
            if (formatId) {
                downloadParams.format_id = formatId
            }

            const response = await downloadApi.submitDownload(downloadParams)

            setCurrentTaskId(response.task_id)
            setHistoryId(response.history_id)

            // Subscribe to progress
            wsClient.subscribe(response.task_id, handleProgress)

            toast.info(`Download task submitted, estimated time: ${response.estimated_time}s`)
        } catch (error) {
            setStatus("error")
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
            toast.success("File download started")
        } catch (error) {
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
    }
}
