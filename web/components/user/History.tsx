"use client"

import * as React from "react"
import { mockApi, DownloadHistoryItem } from "@/lib/mock-api"
import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Download, Trash2, ExternalLink } from "lucide-react"

export function History() {
    const [history, setHistory] = React.useState<DownloadHistoryItem[]>([])
    const [isLoading, setIsLoading] = React.useState(true)

    React.useEffect(() => {
        const loadHistory = async () => {
            try {
                const data = await mockApi.getHistory()
                setHistory(data)
            } catch (error) {
                console.error("Failed to load history", error)
            } finally {
                setIsLoading(false)
            }
        }
        loadHistory()
    }, [])

    if (isLoading) {
        return <div className="text-center py-12 text-gray-500">Loading history...</div>
    }

    if (history.length === 0) {
        return (
            <div className="text-center py-12 space-y-4">
                <div className="text-4xl">📦</div>
                <h3 className="text-lg font-medium">No download history</h3>
                <p className="text-gray-500">Start your first download to see it here.</p>
            </div>
        )
    }

    return (
        <div className="space-y-4">
            {history.map((item) => (
                <Card key={item.id} className="overflow-hidden hover:shadow-md transition-shadow">
                    <CardContent className="p-0 flex flex-col sm:flex-row">
                        <div className="w-full sm:w-48 h-32 relative shrink-0">
                            <img
                                src={item.thumbnail}
                                alt={item.title}
                                className="w-full h-full object-cover"
                            />
                            <div className="absolute bottom-2 right-2 bg-black/70 text-white text-xs px-2 py-1 rounded">
                                {item.duration}
                            </div>
                        </div>
                        <div className="p-4 flex-1 flex flex-col justify-between">
                            <div>
                                <div className="flex items-center gap-2 mb-1">
                                    <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-gray-100 text-gray-600">
                                        {item.platform}
                                    </span>
                                    <span className="text-xs text-gray-400">{item.downloadedAt}</span>
                                </div>
                                <h4 className="font-medium line-clamp-1">{item.title}</h4>
                            </div>
                            <div className="flex gap-2 mt-4 sm:mt-0 justify-end">
                                <Button size="sm" variant="outline">
                                    <Download className="w-4 h-4 mr-2" />
                                    Redownload
                                </Button>
                                <Button size="sm" variant="ghost" className="text-red-500 hover:text-red-600 hover:bg-red-50">
                                    <Trash2 className="w-4 h-4" />
                                </Button>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            ))}
        </div>
    )
}
