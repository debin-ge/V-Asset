"use client"

import * as React from "react"
import { useState, useEffect } from "react"
import {
    cookieApi,
    CookieInfo,
    CookieListResponse,
    CookieStatus,
    CookieStatusLabel,
    PlatformOptions,
    CreateCookieRequest
} from "@/lib/api/cookie"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent } from "@/components/ui/card"
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog"
import { toast } from "sonner"
import {
    Plus,
    Trash2,
    RefreshCw,
    Cookie as CookieIcon,
    CheckCircle,
    XCircle,
    Loader2,
    Snowflake,
    Clock
} from "lucide-react"

export function CookieManagement() {
    const [cookies, setCookies] = useState<CookieInfo[]>([])
    const [loading, setLoading] = useState(true)
    const [total, setTotal] = useState(0)
    const [page, setPage] = useState(1)
    const [isDialogOpen, setIsDialogOpen] = useState(false)
    const [freezingId, setFreezingId] = useState<number | null>(null)
    const [filterPlatform, setFilterPlatform] = useState<string>("")

    // 表单状态
    const [formData, setFormData] = useState<CreateCookieRequest>({
        platform: "youtube",
        name: "",
        content: "",
        expire_at: "",
        freeze_seconds: 60,
    })
    const [submitting, setSubmitting] = useState(false)

    // 加载 Cookie 列表
    const loadCookies = async () => {
        setLoading(true)
        try {
            const params: { page: number; page_size: number; platform?: string } = {
                page,
                page_size: 20
            }
            if (filterPlatform) {
                params.platform = filterPlatform
            }
            const response: CookieListResponse = await cookieApi.list(params)
            setCookies(response.items || [])
            setTotal(response.total)
        } catch (error) {
            toast.error("加载 Cookie 列表失败")
            console.error(error)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        loadCookies()
    }, [page, filterPlatform])

    // 创建 Cookie
    const handleCreate = async () => {
        if (!formData.platform || !formData.name || !formData.content) {
            toast.error("平台、名称和内容不能为空")
            return
        }

        setSubmitting(true)
        try {
            // 格式化过期时间：从 YYYY-MM-DDTHH:MM 转换为 YYYY-MM-DD HH:MM:SS
            const requestData = {
                ...formData,
                expire_at: formData.expire_at
                    ? formData.expire_at.replace("T", " ") + ":00"
                    : "",
            }
            await cookieApi.create(requestData)
            toast.success("Cookie 添加成功")
            setIsDialogOpen(false)
            setFormData({
                platform: "youtube",
                name: "",
                content: "",
                expire_at: "",
                freeze_seconds: 60,
            })
            loadCookies()
        } catch (error) {
            toast.error("添加 Cookie 失败")
            console.error(error)
        } finally {
            setSubmitting(false)
        }
    }

    // 删除 Cookie
    const handleDelete = async (id: number) => {
        if (!confirm("确定要删除这个 Cookie 吗？")) return

        try {
            await cookieApi.delete(id)
            toast.success("Cookie 已删除")
            loadCookies()
        } catch (error) {
            toast.error("删除失败")
            console.error(error)
        }
    }

    // 冷冻 Cookie
    const handleFreeze = async (id: number) => {
        setFreezingId(id)
        try {
            const result = await cookieApi.freeze(id)
            toast.success(`Cookie 已冷冻至 ${result.frozen_until}`)
            loadCookies()
        } catch (error) {
            toast.error("冷冻失败")
            console.error(error)
        } finally {
            setFreezingId(null)
        }
    }

    // 获取状态样式
    const getStatusStyle = (status: number) => {
        switch (status) {
            case CookieStatus.ACTIVE:
                return "bg-green-100 text-green-700"
            case CookieStatus.EXPIRED:
                return "bg-red-100 text-red-700"
            case CookieStatus.FROZEN:
                return "bg-blue-100 text-blue-700"
            default:
                return "bg-gray-100 text-gray-700"
        }
    }

    // 获取平台标签
    const getPlatformLabel = (platform: string) => {
        const option = PlatformOptions.find(p => p.value === platform)
        return option?.label || platform
    }

    // 获取平台样式
    const getPlatformStyle = (platform: string) => {
        switch (platform) {
            case "youtube":
                return "bg-red-100 text-red-700"
            case "bilibili":
                return "bg-pink-100 text-pink-700"
            case "tiktok":
                return "bg-gray-900 text-white"
            case "twitter":
                return "bg-blue-100 text-blue-700"
            default:
                return "bg-gray-100 text-gray-700"
        }
    }

    return (
        <div className="space-y-6">
            {/* 工具栏 */}
            <div className="flex items-center justify-between flex-wrap gap-4">
                <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" onClick={loadCookies} disabled={loading}>
                        <RefreshCw className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`} />
                        刷新
                    </Button>
                    <select
                        className="px-3 py-1.5 border rounded-md text-sm"
                        value={filterPlatform}
                        onChange={(e) => setFilterPlatform(e.target.value)}
                    >
                        <option value="">全部平台</option>
                        {PlatformOptions.map(p => (
                            <option key={p.value} value={p.value}>{p.label}</option>
                        ))}
                    </select>
                    <span className="text-sm text-gray-500">共 {total} 个 Cookie</span>
                </div>

                <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
                    <DialogTrigger asChild>
                        <Button>
                            <Plus className="w-4 h-4 mr-2" />
                            添加 Cookie
                        </Button>
                    </DialogTrigger>
                    <DialogContent className="max-w-2xl">
                        <DialogHeader>
                            <DialogTitle>添加 Cookie</DialogTitle>
                            <DialogDescription>
                                添加一个新的平台 Cookie。内容应为 Netscape 格式。
                            </DialogDescription>
                        </DialogHeader>
                        <div className="grid gap-4 py-4">
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">平台 *</label>
                                    <select
                                        className="w-full px-3 py-2 border rounded-md"
                                        value={formData.platform}
                                        onChange={(e) => setFormData({ ...formData, platform: e.target.value })}
                                    >
                                        {PlatformOptions.map(p => (
                                            <option key={p.value} value={p.value}>{p.label}</option>
                                        ))}
                                    </select>
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">名称 *</label>
                                    <Input
                                        placeholder="如: 账号1"
                                        value={formData.name}
                                        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                    />
                                </div>
                            </div>
                            <div className="space-y-2">
                                <label className="text-sm font-medium">Cookie 内容 *</label>
                                <textarea
                                    className="w-full px-3 py-2 border rounded-md min-h-[120px] font-mono text-sm"
                                    placeholder="粘贴 Netscape 格式的 Cookie 内容..."
                                    value={formData.content}
                                    onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                                />
                            </div>
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">过期时间</label>
                                    <Input
                                        type="datetime-local"
                                        value={formData.expire_at ? formData.expire_at.replace(" ", "T").slice(0, 16) : ""}
                                        onChange={(e) => setFormData({ ...formData, expire_at: e.target.value })}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">使用后冷冻时间 (秒)</label>
                                    <Input
                                        type="number"
                                        placeholder="60"
                                        value={formData.freeze_seconds || ""}
                                        onChange={(e) => setFormData({ ...formData, freeze_seconds: parseInt(e.target.value) || 0 })}
                                    />
                                </div>
                            </div>
                        </div>
                        <DialogFooter>
                            <Button variant="outline" onClick={() => setIsDialogOpen(false)}>
                                取消
                            </Button>
                            <Button onClick={handleCreate} disabled={submitting}>
                                {submitting && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                                添加
                            </Button>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>
            </div>

            {/* Cookie 列表 */}
            {loading ? (
                <div className="flex items-center justify-center py-12">
                    <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
                </div>
            ) : cookies.length === 0 ? (
                <Card>
                    <CardContent className="flex flex-col items-center justify-center py-12">
                        <CookieIcon className="w-12 h-12 text-gray-300 mb-4" />
                        <p className="text-gray-500">暂无 Cookie</p>
                        <p className="text-sm text-gray-400">点击上方按钮添加 Cookie</p>
                    </CardContent>
                </Card>
            ) : (
                <div className="grid gap-4">
                    {cookies.map((cookie) => (
                        <Card key={cookie.id} className="hover:shadow-md transition-shadow">
                            <CardContent className="p-4">
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-4">
                                        <div className={`p-2 rounded-lg ${cookie.status === CookieStatus.ACTIVE ? "bg-green-100" : cookie.status === CookieStatus.FROZEN ? "bg-blue-100" : "bg-gray-100"}`}>
                                            <CookieIcon className={`w-5 h-5 ${cookie.status === CookieStatus.ACTIVE ? "text-green-600" : cookie.status === CookieStatus.FROZEN ? "text-blue-600" : "text-gray-400"}`} />
                                        </div>
                                        <div>
                                            <div className="font-medium flex items-center gap-2">
                                                {cookie.name}
                                                <span className={`px-2 py-0.5 text-xs rounded-full ${getPlatformStyle(cookie.platform)}`}>
                                                    {getPlatformLabel(cookie.platform)}
                                                </span>
                                                <span className={`px-2 py-0.5 text-xs rounded-full ${getStatusStyle(cookie.status)}`}>
                                                    {CookieStatusLabel[cookie.status]}
                                                </span>
                                            </div>
                                            <div className="text-sm text-gray-500 mt-1 flex items-center gap-4 flex-wrap">
                                                <span className="flex items-center gap-1">
                                                    <Clock className="w-3 h-3" />
                                                    使用 {cookie.use_count}
                                                </span>
                                                <span className="flex items-center gap-1">
                                                    <CheckCircle className="w-3 h-3 text-green-500" />
                                                    成功 {cookie.success_count}
                                                </span>
                                                <span className="flex items-center gap-1">
                                                    <XCircle className="w-3 h-3 text-red-500" />
                                                    失败 {cookie.fail_count}
                                                </span>
                                                {cookie.frozen_until && cookie.status === CookieStatus.FROZEN && (
                                                    <span className="flex items-center gap-1 text-blue-600">
                                                        <Snowflake className="w-3 h-3" />
                                                        冷冻至 {cookie.frozen_until}
                                                    </span>
                                                )}
                                                {cookie.expire_at && (
                                                    <span>过期: {cookie.expire_at}</span>
                                                )}
                                            </div>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => handleFreeze(cookie.id)}
                                            disabled={freezingId === cookie.id}
                                            title="手动冷冻"
                                        >
                                            {freezingId === cookie.id ? (
                                                <Loader2 className="w-4 h-4 animate-spin" />
                                            ) : (
                                                <Snowflake className="w-4 h-4" />
                                            )}
                                        </Button>
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => handleDelete(cookie.id)}
                                            className="text-red-500 hover:text-red-700 hover:bg-red-50"
                                        >
                                            <Trash2 className="w-4 h-4" />
                                        </Button>
                                    </div>
                                </div>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            )}
        </div>
    )
}
