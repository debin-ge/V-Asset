"use client"

import * as React from "react"
import { mockApi, VideoInfo } from "@/lib/mock-api"
import { toast } from "sonner"

export type DownloadStatus = "idle" | "parsing" | "parsed" | "downloading" | "completed" | "error"

export function useDownload() {
    const [url, setUrl] = React.useState("")
    const [status, setStatus] = React.useState<DownloadStatus>("idle")
    const [videoInfo, setVideoInfo] = React.useState<VideoInfo | null>(null)
    const [progress, setProgress] = React.useState(0)
    const [speed, setSpeed] = React.useState("0 MB/s")
    const [timeLeft, setTimeLeft] = React.useState("")

    const handleParse = async (inputUrl: string) => {
        if (!inputUrl) return
        setStatus("parsing")
        try {
            const info = await mockApi.parseUrl(inputUrl)
            setVideoInfo(info)
            setStatus("parsed")
        } catch (error) {
            setStatus("error")
            toast.error("Failed to parse URL. Please check the link and try again.")
        }
    }

    const startDownload = (type: 'video' | 'audio') => {
        setStatus("downloading")
        setProgress(0)

        // Simulate download progress
        let currentProgress = 0
        const interval = setInterval(() => {
            currentProgress += Math.random() * 5
            if (currentProgress >= 100) {
                currentProgress = 100
                clearInterval(interval)
                setStatus("completed")
                toast.success("Download completed!")
            }

            setProgress(Math.min(currentProgress, 100))
            setSpeed(`${(Math.random() * 5 + 1).toFixed(1)} MB/s`)
            setTimeLeft(`About ${Math.max(0, Math.floor((100 - currentProgress) / 2))}s`)
        }, 200)
    }

    const reset = () => {
        setUrl("")
        setStatus("idle")
        setVideoInfo(null)
        setProgress(0)
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
        reset,
    }
}
