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

    // 清理WebSocket订阅
    React.useEffect(() => {
        return () => {
            if (currentTaskId) {
                wsClient.unsubscribe(currentTaskId)
            }
        }
    }, [currentTaskId])

    // 处理进度更新
    const handleProgress = React.useCallback((data: ProgressData) => {
        setProgress(data.percent)
        setSpeed(data.speed || "0 MB/s")
        setTimeLeft(data.eta || "")

        if (data.status === 2) { // 完成
            setStatus("completed")
            toast.success("下载完成！")
            if (currentTaskId) {
                wsClient.unsubscribe(currentTaskId)
            }
        } else if (data.status === 3) { // 失败
            setStatus("error")
            toast.error(data.error_message || "下载失败")
            if (currentTaskId) {
                wsClient.unsubscribe(currentTaskId)
            }
        }
    }, [currentTaskId])

    // 解析URL
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

    // 开始下载
    const startDownload = async (type: 'video' | 'audio') => {
        if (!videoInfo) return

        setStatus("downloading")
        setProgress(0)

        try {
            const response = await downloadApi.submitDownload({
                url: videoInfo.url,
                mode: mapDownloadType(type),
                quality: "best",
            })

            setCurrentTaskId(response.task_id)
            setHistoryId(response.history_id)

            // 订阅进度
            wsClient.subscribe(response.task_id, handleProgress)

            toast.info(`下载任务已提交，预计耗时 ${response.estimated_time} 秒`)
        } catch (error) {
            setStatus("error")
            const message = error instanceof Error ? error.message : "提交下载任务失败"
            toast.error(message)
        }
    }

    // 下载文件到本地
    const downloadFile = async () => {
        if (!historyId) {
            toast.error("没有可下载的文件")
            return
        }

        try {
            await downloadApi.downloadFile(historyId)
            toast.success("文件下载已开始")
        } catch (error) {
            const message = error instanceof Error ? error.message : "文件下载失败"
            toast.error(message)
        }
    }

    // 重置状态
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
