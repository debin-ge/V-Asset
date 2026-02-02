"use client"

import * as React from "react"
import { toast } from "sonner"
import { parseApi, VideoInfo } from "@/lib/api/parse"
import { downloadApi, StreamDownloadParams, getProgress, ProgressResponse } from "@/lib/api/download"

export type DownloadStatus = "idle" | "parsing" | "parsed" | "downloading" | "completed" | "error"

export interface DownloadProgress {
    taskId: string;
    status: string;
    progress: number;
    speed: string;
    eta: number;
    filename: string;
    totalBytes: number;
    downloadedBytes: number;
}

export function useDownload() {
    const [url, setUrl] = React.useState("")
    const [status, setStatus] = React.useState<DownloadStatus>("idle")
    const [videoInfo, setVideoInfo] = React.useState<VideoInfo | null>(null)
    const [downloadProgress, setDownloadProgress] = React.useState<DownloadProgress | null>(null)
    const [downloadingTaskId, setDownloadingTaskId] = React.useState<string | null>(null)
    const pollingRef = React.useRef<NodeJS.Timeout | null>(null)

    // 清理轮询
    const stopPolling = React.useCallback(() => {
        if (pollingRef.current) {
            clearInterval(pollingRef.current)
            pollingRef.current = null
        }
    }, [])

    // 进度轮询
    React.useEffect(() => {
        if (!downloadingTaskId) {
            stopPolling()
            return
        }

        const pollProgress = async () => {
            try {
                const res = await getProgress(downloadingTaskId)
                setDownloadProgress({
                    taskId: res.task_id,
                    status: res.status,
                    progress: res.progress,
                    speed: res.speed,
                    eta: res.eta,
                    filename: res.filename,
                    totalBytes: res.total_bytes,
                    downloadedBytes: res.downloaded_bytes,
                })

                if (res.status === 'completed') {
                    stopPolling()
                    setStatus("completed")
                    setDownloadingTaskId(null)
                    toast.success("下载完成！")
                } else if (res.status === 'failed') {
                    stopPolling()
                    setStatus("error")
                    setDownloadingTaskId(null)
                    toast.error(res.error || "下载失败")
                }
            } catch (error) {
                console.error("Failed to get progress:", error)
                // 不停止轮询，继续尝试
            }
        }

        // 立即执行一次
        pollProgress()

        // 每秒轮询
        pollingRef.current = setInterval(pollProgress, 1000)

        return () => {
            stopPolling()
        }
    }, [downloadingTaskId, stopPolling])

    // 组件卸载时清理
    React.useEffect(() => {
        return () => {
            stopPolling()
        }
    }, [stopPolling])

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
            const message = error instanceof Error ? error.message : "解析失败，请检查链接"
            toast.error(message)
        }
    }

    // Start download - 直接流式下载
    const startDownload = async (type: 'video' | 'audio', formatId?: string) => {
        if (!videoInfo) return

        setStatus("downloading")
        setDownloadProgress(null)
        toast.info("正在开始下载...")

        try {
            const isVideo = type === 'video'
            const ext = isVideo ? 'mp4' : 'm4a'

            const params: StreamDownloadParams = {
                url: videoInfo.url,
                name: videoInfo.title || 'download',
                ext: ext,
                is_video: isVideo,
            }

            if (formatId) {
                params.format_id = formatId
            }

            // 直接触发浏览器下载
            await downloadApi.streamDownload(params)

            setStatus("completed")
            toast.success("下载完成！")
        } catch (error) {
            setStatus("error")
            const message = error instanceof Error ? error.message : "下载失败"
            toast.error(message)
        }
    }

    // 开始异步下载任务（带进度跟踪）
    const startAsyncDownload = async (type: 'video' | 'audio', formatId?: string, taskId?: string) => {
        if (!taskId) {
            // 如果没有 taskId，使用同步下载
            return startDownload(type, formatId)
        }

        setStatus("downloading")
        setDownloadProgress(null)
        setDownloadingTaskId(taskId)
        toast.info("正在开始下载...")
    }

    // Reset state
    const reset = () => {
        stopPolling()
        setUrl("")
        setStatus("idle")
        setVideoInfo(null)
        setDownloadProgress(null)
        setDownloadingTaskId(null)
    }

    return {
        url,
        setUrl,
        status,
        videoInfo,
        downloadProgress,
        handleParse,
        startDownload,
        startAsyncDownload,
        reset,
    }
}
