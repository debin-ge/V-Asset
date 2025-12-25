"use client"

import * as React from "react"
import { useState, useEffect } from "react"
import {
    proxyApi,
    ProxyInfo,
    ProxyListResponse,
    ProxyStatus,
    ProxyStatusLabel,
    CreateProxyRequest
} from "@/lib/api/proxy"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
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
    Server,
    CheckCircle,
    XCircle,
    Loader2,
    Activity
} from "lucide-react"

export function ProxyManagement() {
    const [proxies, setProxies] = useState<ProxyInfo[]>([])
    const [loading, setLoading] = useState(true)
    const [total, setTotal] = useState(0)
    const [page, setPage] = useState(1)
    const [isDialogOpen, setIsDialogOpen] = useState(false)
    const [checkingId, setCheckingId] = useState<number | null>(null)

    // 表单状态
    const [formData, setFormData] = useState<CreateProxyRequest>({
        ip: "",
        port: 0,
        username: "",
        password: "",
        protocol: "http",
        region: "",
        check_health: true,
    })
    const [submitting, setSubmitting] = useState(false)

    // 加载代理列表
    const loadProxies = async () => {
        setLoading(true)
        try {
            const response: ProxyListResponse = await proxyApi.list({ page, page_size: 20 })
            setProxies(response.items || [])
            setTotal(response.total)
        } catch (error) {
            toast.error("加载代理列表失败")
            console.error(error)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        loadProxies()
    }, [page])

    // 创建代理
    const handleCreate = async () => {
        if (!formData.ip || !formData.port) {
            toast.error("IP 和端口不能为空")
            return
        }

        setSubmitting(true)
        try {
            const result = await proxyApi.create(formData)
            if (formData.check_health && !result.health_check_passed) {
                toast.warning(`代理已添加，但健康检查失败：${result.health_check_error}`)
            } else {
                toast.success("代理添加成功")
            }
            setIsDialogOpen(false)
            setFormData({
                ip: "",
                port: 0,
                username: "",
                password: "",
                protocol: "http",
                region: "",
                check_health: true,
            })
            loadProxies()
        } catch (error) {
            toast.error("添加代理失败")
            console.error(error)
        } finally {
            setSubmitting(false)
        }
    }

    // 删除代理
    const handleDelete = async (id: number) => {
        if (!confirm("确定要删除这个代理吗？")) return

        try {
            await proxyApi.delete(id)
            toast.success("代理已删除")
            loadProxies()
        } catch (error) {
            toast.error("删除失败")
            console.error(error)
        }
    }

    // 健康检查
    const handleHealthCheck = async (id: number) => {
        setCheckingId(id)
        try {
            const result = await proxyApi.checkHealth(id)
            if (result.healthy) {
                toast.success(`检查通过，延迟: ${result.latency_ms}ms`)
            } else {
                toast.error(`检查失败: ${result.error}`)
            }
            loadProxies()
        } catch (error) {
            toast.error("健康检查失败")
            console.error(error)
        } finally {
            setCheckingId(null)
        }
    }

    // 获取状态样式
    const getStatusStyle = (status: number) => {
        switch (status) {
            case ProxyStatus.ACTIVE:
                return "bg-green-100 text-green-700"
            case ProxyStatus.INACTIVE:
                return "bg-red-100 text-red-700"
            case ProxyStatus.CHECKING:
                return "bg-yellow-100 text-yellow-700"
            default:
                return "bg-gray-100 text-gray-700"
        }
    }

    return (
        <div className="space-y-6">
            {/* 工具栏 */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" onClick={loadProxies} disabled={loading}>
                        <RefreshCw className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`} />
                        刷新
                    </Button>
                    <span className="text-sm text-gray-500">共 {total} 个代理</span>
                </div>

                <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
                    <DialogTrigger asChild>
                        <Button>
                            <Plus className="w-4 h-4 mr-2" />
                            添加代理
                        </Button>
                    </DialogTrigger>
                    <DialogContent>
                        <DialogHeader>
                            <DialogTitle>添加代理</DialogTitle>
                            <DialogDescription>
                                添加一个新的代理服务器。可选择是否进行健康检查。
                            </DialogDescription>
                        </DialogHeader>
                        <div className="grid gap-4 py-4">
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">IP 地址 *</label>
                                    <Input
                                        placeholder="192.168.1.1"
                                        value={formData.ip}
                                        onChange={(e) => setFormData({ ...formData, ip: e.target.value })}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">端口 *</label>
                                    <Input
                                        type="number"
                                        placeholder="8080"
                                        value={formData.port || ""}
                                        onChange={(e) => setFormData({ ...formData, port: parseInt(e.target.value) || 0 })}
                                    />
                                </div>
                            </div>
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">用户名</label>
                                    <Input
                                        placeholder="可选"
                                        value={formData.username}
                                        onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">密码</label>
                                    <Input
                                        type="password"
                                        placeholder="可选"
                                        value={formData.password}
                                        onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                                    />
                                </div>
                            </div>
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">协议</label>
                                    <select
                                        className="w-full px-3 py-2 border rounded-md"
                                        value={formData.protocol}
                                        onChange={(e) => setFormData({ ...formData, protocol: e.target.value })}
                                    >
                                        <option value="http">HTTP</option>
                                        <option value="https">HTTPS</option>
                                        <option value="socks5">SOCKS5</option>
                                    </select>
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">地区</label>
                                    <Input
                                        placeholder="如: US, CN"
                                        value={formData.region}
                                        onChange={(e) => setFormData({ ...formData, region: e.target.value })}
                                    />
                                </div>
                            </div>
                            <div className="flex items-center gap-2">
                                <input
                                    type="checkbox"
                                    id="check_health"
                                    checked={formData.check_health}
                                    onChange={(e) => setFormData({ ...formData, check_health: e.target.checked })}
                                />
                                <label htmlFor="check_health" className="text-sm">添加时进行健康检查</label>
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

            {/* 代理列表 */}
            {loading ? (
                <div className="flex items-center justify-center py-12">
                    <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
                </div>
            ) : proxies.length === 0 ? (
                <Card>
                    <CardContent className="flex flex-col items-center justify-center py-12">
                        <Server className="w-12 h-12 text-gray-300 mb-4" />
                        <p className="text-gray-500">暂无代理</p>
                        <p className="text-sm text-gray-400">点击上方按钮添加代理</p>
                    </CardContent>
                </Card>
            ) : (
                <div className="grid gap-4">
                    {proxies.map((proxy) => (
                        <Card key={proxy.id} className="hover:shadow-md transition-shadow">
                            <CardContent className="p-4">
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-4">
                                        <div className={`p-2 rounded-lg ${proxy.status === ProxyStatus.ACTIVE ? "bg-green-100" : "bg-gray-100"}`}>
                                            <Server className={`w-5 h-5 ${proxy.status === ProxyStatus.ACTIVE ? "text-green-600" : "text-gray-400"}`} />
                                        </div>
                                        <div>
                                            <div className="font-medium">
                                                {proxy.ip}:{proxy.port}
                                                <span className={`ml-2 px-2 py-0.5 text-xs rounded-full ${getStatusStyle(proxy.status)}`}>
                                                    {ProxyStatusLabel[proxy.status]}
                                                </span>
                                                <span className="ml-2 px-2 py-0.5 text-xs rounded-full bg-blue-100 text-blue-700">
                                                    {proxy.protocol.toUpperCase()}
                                                </span>
                                                {proxy.region && (
                                                    <span className="ml-2 px-2 py-0.5 text-xs rounded-full bg-gray-100 text-gray-600">
                                                        {proxy.region}
                                                    </span>
                                                )}
                                            </div>
                                            <div className="text-sm text-gray-500 mt-1 flex items-center gap-4">
                                                <span className="flex items-center gap-1">
                                                    <CheckCircle className="w-3 h-3 text-green-500" />
                                                    成功 {proxy.success_count}
                                                </span>
                                                <span className="flex items-center gap-1">
                                                    <XCircle className="w-3 h-3 text-red-500" />
                                                    失败 {proxy.fail_count}
                                                </span>
                                                {proxy.last_check_at && (
                                                    <span>上次检查: {proxy.last_check_at}</span>
                                                )}
                                            </div>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => handleHealthCheck(proxy.id)}
                                            disabled={checkingId === proxy.id}
                                        >
                                            {checkingId === proxy.id ? (
                                                <Loader2 className="w-4 h-4 animate-spin" />
                                            ) : (
                                                <Activity className="w-4 h-4" />
                                            )}
                                        </Button>
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => handleDelete(proxy.id)}
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
