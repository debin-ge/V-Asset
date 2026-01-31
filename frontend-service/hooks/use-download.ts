"use client"

import * as React from "react"
import { toast } from "sonner"
import { parseApi, VideoInfo } from "@/lib/api/parse"
import { downloadApi, StreamDownloadParams } from "@/lib/api/download"

export type DownloadStatus = "idle" | "parsing" | "parsed" | "downloading" | "completed" | "error"

export function useDownload() {
    const [url, setUrl] = React.useState("")
    const [status, setStatus] = React.useState<DownloadStatus>("idle")
    const [videoInfo, setVideoInfo] = React.useState<VideoInfo | null>(null)

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

    // Reset state
    const reset = () => {
        setUrl("")
        setStatus("idle")
        setVideoInfo(null)
    }

    return {
        url,
        setUrl,
        status,
        videoInfo,
        handleParse,
        startDownload,
        reset,
    }
}

