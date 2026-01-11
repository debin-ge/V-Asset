"use client"

import * as React from "react"
import { historyApi, HistoryItem } from "@/lib/api/history"
import { downloadApi } from "@/lib/api/download"
import { formatDate, formatFileSize, formatDuration, getStatusText } from "@/lib/format"
import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Download, Trash2, Loader2, CheckCircle, XCircle, Clock } from "lucide-react"
import { toast } from "sonner"

export function History() {
    const [history, setHistory] = React.useState<HistoryItem[]>([])
    const [isLoading, setIsLoading] = React.useState(true)
    const [deletingId, setDeletingId] = React.useState<number | null>(null)
    const [downloadingId, setDownloadingId] = React.useState<number | null>(null)

    const loadHistory = React.useCallback(async () => {
        try {
            const data = await historyApi.getHistory()
            setHistory(data.items)
        } catch (error) {
            console.error("Failed to load history", error)
            toast.error("Failed to load history")
        } finally {
            setIsLoading(false)
        }
    }, [])

    React.useEffect(() => {
        loadHistory()
    }, [loadHistory])

    const handleDelete = async (historyId: number) => {
        setDeletingId(historyId)
        try {
            await historyApi.deleteHistory(historyId)
            setHistory(prev => prev.filter(h => h.history_id !== historyId))
            toast.success("Deleted successfully")
        } catch (error) {
            toast.error("Delete failed")
        } finally {
            setDeletingId(null)
        }
    }

    const handleDownload = async (historyId: number) => {
        setDownloadingId(historyId)
        try {
            await downloadApi.downloadFile(historyId)
            toast.success("Download started")
        } catch (error) {
            toast.error("Download failed")
        } finally {
            setDownloadingId(null)
        }
    }

    const getStatusIcon = (status: number) => {
        switch (status) {
            case 2: return <CheckCircle className="w-4 h-4 text-green-500" />
            case 3: return <XCircle className="w-4 h-4 text-red-500" />
            default: return <Clock className="w-4 h-4 text-yellow-500" />
        }
    }

    if (isLoading) {
        return (
            <div className="flex items-center justify-center py-12 text-gray-500">
                <Loader2 className="w-6 h-6 animate-spin mr-2" />
                Loading...
            </div>
        )
    }

    if (history.length === 0) {
        return (
            <div className="text-center py-12 space-y-4">
                <div className="text-4xl">📦</div>
                <h3 className="text-lg font-medium">No download history</h3>
                <p className="text-gray-500">Start your first download!</p>
            </div>
        )
    }

    return (
        <div className="space-y-4">
            {history.map((item) => (
                <Card key={item.history_id} className="overflow-hidden hover:shadow-md transition-shadow">
                    <CardContent className="p-0 flex flex-col sm:flex-row">
                        <div className="w-full sm:w-48 h-32 relative shrink-0 bg-gray-100">
                            {item.thumbnail ? (
                                <img
                                    src={item.thumbnail}
                                    alt={item.title}
                                    className="w-full h-full object-cover"
                                />
                            ) : (
                                <div className="w-full h-full flex items-center justify-center text-gray-400">
                                    No cover
                                </div>
                            )}
                            {item.duration > 0 && (
                                <div className="absolute bottom-2 right-2 bg-black/70 text-white text-xs px-2 py-1 rounded">
                                    {formatDuration(item.duration)}
                                </div>
                            )}
                        </div>
                        <div className="p-4 flex-1 flex flex-col justify-between">
                            <div>
                                <div className="flex items-center gap-2 mb-1 flex-wrap">
                                    <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-gray-100 text-gray-600">
                                        {item.platform}
                                    </span>
                                    <div className="flex items-center gap-1 text-xs">
                                        {getStatusIcon(item.status)}
                                        <span className="text-gray-500">{getStatusText(item.status)}</span>
                                    </div>
                                    <span className="text-xs text-gray-400">{formatDate(item.created_at)}</span>
                                </div>
                                <h4 className="font-medium line-clamp-1">{item.title}</h4>
                                {item.file_size > 0 && (
                                    <p className="text-xs text-gray-500 mt-1">
                                        {formatFileSize(item.file_size)} · {item.quality}
                                    </p>
                                )}
                            </div>
                            <div className="flex gap-2 mt-4 sm:mt-0 justify-end">
                                {item.status === 2 && (
                                    <Button
                                        size="sm"
                                        variant="outline"
                                        onClick={() => handleDownload(item.history_id)}
                                        disabled={downloadingId === item.history_id}
                                    >
                                        {downloadingId === item.history_id ? (
                                            <Loader2 className="w-4 h-4 animate-spin mr-2" />
                                        ) : (
                                            <Download className="w-4 h-4 mr-2" />
                                        )}
                                        Download
                                    </Button>
                                )}
                                <Button
                                    size="sm"
                                    variant="ghost"
                                    className="text-red-500 hover:text-red-600 hover:bg-red-50"
                                    onClick={() => handleDelete(item.history_id)}
                                    disabled={deletingId === item.history_id}
                                >
                                    {deletingId === item.history_id ? (
                                        <Loader2 className="w-4 h-4 animate-spin" />
                                    ) : (
                                        <Trash2 className="w-4 h-4" />
                                    )}
                                </Button>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            ))}
        </div>
    )
}
