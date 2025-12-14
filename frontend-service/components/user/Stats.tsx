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
                setError("加载统计数据失败")
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
                加载中...
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

    // 转换图表数据
    const chartData = stats?.recent_activity?.map(a => ({
        name: a.date.slice(5), // MM-DD
        downloads: a.count
    })) ?? []

    // 计算配额重置时间
    const getResetTimeText = () => {
        if (!quota?.reset_at) return "明日重置"
        try {
            const resetDate = new Date(quota.reset_at)
            const now = new Date()
            const diffHours = Math.max(0, Math.floor((resetDate.getTime() - now.getTime()) / (1000 * 60 * 60)))
            return `${diffHours}小时后重置`
        } catch {
            return "明日重置"
        }
    }

    return (
        <div className="space-y-6">
            <div className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">总下载次数</CardTitle>
                        <Download className="w-4 h-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">{stats?.total_downloads ?? 0}</div>
                        <p className="text-xs text-muted-foreground">
                            成功 {stats?.success_downloads ?? 0} · 失败 {stats?.failed_downloads ?? 0}
                        </p>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">今日配额</CardTitle>
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
                        <CardTitle className="text-sm font-medium">总下载大小</CardTitle>
                        <Database className="w-4 h-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">
                            {formatFileSize(stats?.total_size_bytes ?? 0)}
                        </div>
                        <p className="text-xs text-muted-foreground">累计下载文件大小</p>
                    </CardContent>
                </Card>
            </div>

            {/* 平台分布 */}
            {stats?.top_platforms && stats.top_platforms.length > 0 && (
                <Card>
                    <CardHeader>
                        <CardTitle>平台分布</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="flex flex-wrap gap-3">
                            {stats.top_platforms.map(p => (
                                <div key={p.platform} className="flex items-center gap-2 bg-gray-50 px-3 py-2 rounded-lg">
                                    <span className="font-medium">{p.platform}</span>
                                    <span className="text-sm text-gray-500">{p.count} 次</span>
                                </div>
                            ))}
                        </div>
                    </CardContent>
                </Card>
            )}

            {/* 下载活动图表 */}
            {chartData.length > 0 && (
                <Card className="col-span-4">
                    <CardHeader>
                        <CardTitle>近期下载活动</CardTitle>
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
