"use client"

import * as React from "react"
import { historyApi, StatsResponse, QuotaResponse } from "@/lib/api/history"
import { formatFileSize } from "@/lib/format"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts"
import { Loader2, Download, Database, Zap } from "lucide-react"

export function Stats() {
    const [stats, setStats] = React.useState<StatsResponse | null>(null)
    const [quota, setQuota] = React.useState<QuotaResponse | null>(null)
    const [isLoading, setIsLoading] = React.useState(true)
    const [error, setError] = React.useState<string | null>(null)

    React.useEffect(() => {
        const loadData = async () => {
            try {
                const [statsData, quotaData] = await Promise.all([
                    historyApi.getStats(),
                    historyApi.getQuota()
                ])
                setStats(statsData)
                setQuota(quotaData)
            } catch (err) {
                console.error("Failed to load stats", err)
                setError("Failed to load statistics")
            } finally {
                setIsLoading(false)
            }
        }
        loadData()
    }, [])

    if (isLoading) {
        return (
            <div className="flex items-center justify-center py-12 text-gray-500">
                <Loader2 className="w-6 h-6 animate-spin mr-2" />
                Loading...
            </div>
        )
    }

    if (error) {
        return (
            <div className="text-center py-12 text-red-500">
                {error}
            </div>
        )
    }

    // Transform chart data
    const chartData = stats?.recent_activity?.map(a => ({
        name: a.date.slice(5), // MM-DD
        downloads: a.count
    })) ?? []

    // Calculate quota reset time
    const getResetTimeText = () => {
        if (!quota?.reset_at) return "Resets tomorrow"
        try {
            const resetDate = new Date(quota.reset_at)
            const now = new Date()
            const diffHours = Math.max(0, Math.floor((resetDate.getTime() - now.getTime()) / (1000 * 60 * 60)))
            return `Resets in ${diffHours}h`
        } catch {
            return "Resets tomorrow"
        }
    }

    return (
        <div className="space-y-6">
            <div className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Total Downloads</CardTitle>
                        <Download className="w-4 h-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">{stats?.total_downloads ?? 0}</div>
                        <p className="text-xs text-muted-foreground">
                            Success {stats?.success_downloads ?? 0} · Failed {stats?.failed_downloads ?? 0}
                        </p>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Today's Quota</CardTitle>
                        <Zap className="w-4 h-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">
                            {quota?.remaining ?? 0} / {quota?.daily_limit ?? 0}
                        </div>
                        <p className="text-xs text-muted-foreground">{getResetTimeText()}</p>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Total Download Size</CardTitle>
                        <Database className="w-4 h-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">
                            {formatFileSize(stats?.total_size_bytes ?? 0)}
                        </div>
                        <p className="text-xs text-muted-foreground">Cumulative file size</p>
                    </CardContent>
                </Card>
            </div>

            {/* Platform distribution */}
            {stats?.top_platforms && stats.top_platforms.length > 0 && (
                <Card>
                    <CardHeader>
                        <CardTitle>Platform Distribution</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="flex flex-wrap gap-3">
                            {stats.top_platforms.map(p => (
                                <div key={p.platform} className="flex items-center gap-2 bg-gray-50 px-3 py-2 rounded-lg">
                                    <span className="font-medium">{p.platform}</span>
                                    <span className="text-sm text-gray-500">{p.count} times</span>
                                </div>
                            ))}
                        </div>
                    </CardContent>
                </Card>
            )}

            {/* Download activity chart */}
            {chartData.length > 0 && (
                <Card className="col-span-4">
                    <CardHeader>
                        <CardTitle>Recent Download Activity</CardTitle>
                    </CardHeader>
                    <CardContent className="pl-2">
                        <div className="h-[300px]">
                            <ResponsiveContainer width="100%" height="100%">
                                <BarChart data={chartData}>
                                    <XAxis dataKey="name" stroke="#888888" fontSize={12} tickLine={false} axisLine={false} />
                                    <YAxis stroke="#888888" fontSize={12} tickLine={false} axisLine={false} tickFormatter={(value) => `${value}`} />
                                    <Tooltip
                                        cursor={{ fill: 'transparent' }}
                                        contentStyle={{ borderRadius: '8px', border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }}
                                    />
                                    <Bar dataKey="downloads" fill="#0070F3" radius={[4, 4, 0, 0]} />
                                </BarChart>
                            </ResponsiveContainer>
                        </div>
                    </CardContent>
                </Card>
            )}
        </div>
    )
}
