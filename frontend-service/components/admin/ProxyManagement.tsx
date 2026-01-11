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

    // Form state
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

    // Load proxy list
    const loadProxies = async () => {
        setLoading(true)
        try {
            const response: ProxyListResponse = await proxyApi.list({ page, page_size: 20 })
            setProxies(response.items || [])
            setTotal(response.total)
        } catch (error) {
            toast.error("Failed to load proxy list")
            console.error(error)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        loadProxies()
    }, [page])

    // Create proxy
    const handleCreate = async () => {
        if (!formData.ip || !formData.port) {
            toast.error("IP and port cannot be empty")
            return
        }

        setSubmitting(true)
        try {
            const result = await proxyApi.create(formData)
            if (formData.check_health && !result.health_check_passed) {
                toast.warning(`Proxy added, but health check failed: ${result.health_check_error}`)
            } else {
                toast.success("Proxy added successfully")
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
            toast.error("Failed to add proxy")
            console.error(error)
        } finally {
            setSubmitting(false)
        }
    }

    // Delete proxy
    const handleDelete = async (id: number) => {
        if (!confirm("Are you sure you want to delete this proxy?")) return

        try {
            await proxyApi.delete(id)
            toast.success("Proxy deleted")
            loadProxies()
        } catch (error) {
            toast.error("Failed to delete")
            console.error(error)
        }
    }

    // Health check
    const handleHealthCheck = async (id: number) => {
        setCheckingId(id)
        try {
            const result = await proxyApi.checkHealth(id)
            if (result.healthy) {
                toast.success(`Check passed, latency: ${result.latency_ms}ms`)
            } else {
                toast.error(`Check failed: ${result.error}`)
            }
            loadProxies()
        } catch (error) {
            toast.error("Health check failed")
            console.error(error)
        } finally {
            setCheckingId(null)
        }
    }

    // Get status style
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
            {/* Toolbar */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" onClick={loadProxies} disabled={loading}>
                        <RefreshCw className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`} />
                        Refresh
                    </Button>
                    <span className="text-sm text-gray-500">Total: {total} proxies</span>
                </div>

                <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
                    <DialogTrigger asChild>
                        <Button>
                            <Plus className="w-4 h-4 mr-2" />
                            Add Proxy
                        </Button>
                    </DialogTrigger>
                    <DialogContent>
                        <DialogHeader>
                            <DialogTitle>Add Proxy</DialogTitle>
                            <DialogDescription>
                                Add a new proxy server. You can optionally perform a health check.
                            </DialogDescription>
                        </DialogHeader>
                        <div className="grid gap-4 py-4">
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">IP Address *</label>
                                    <Input
                                        placeholder="192.168.1.1"
                                        value={formData.ip}
                                        onChange={(e) => setFormData({ ...formData, ip: e.target.value })}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">Port *</label>
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
                                    <label className="text-sm font-medium">Username</label>
                                    <Input
                                        placeholder="Optional"
                                        value={formData.username}
                                        onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">Password</label>
                                    <Input
                                        type="password"
                                        placeholder="Optional"
                                        value={formData.password}
                                        onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                                    />
                                </div>
                            </div>
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">Protocol</label>
                                    <select
                                        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background"
                                        value={formData.protocol}
                                        onChange={(e) => setFormData({ ...formData, protocol: e.target.value as "http" | "https" | "socks5" })}
                                    >
                                        <option value="http">HTTP</option>
                                        <option value="https">HTTPS</option>
                                        <option value="socks5">SOCKS5</option>
                                    </select>
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">Region</label>
                                    <Input
                                        placeholder="e.g., US, EU"
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
                                    className="h-4 w-4 rounded border-gray-300"
                                />
                                <label htmlFor="check_health" className="text-sm">Perform health check on add</label>
                            </div>
                        </div>
                        <DialogFooter>
                            <Button onClick={handleCreate} disabled={submitting}>
                                Add
                            </Button>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>
            </div>

            {/* Proxy list */}
            {loading ? (
                <div className="flex items-center justify-center py-12">
                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-400"></div>
                </div>
            ) : proxies.length === 0 ? (
                <Card>
                    <CardContent className="flex flex-col items-center justify-center py-12">
                        <Server className="w-12 h-12 text-gray-300 mb-4" />
                        <p className="text-gray-500">No proxies yet</p>
                        <p className="text-sm text-gray-400">Click the button above to add a proxy</p>
                    </CardContent>
                </Card>
            ) : (
                <div className="grid gap-4">
                    {proxies.map((proxy) => (
                        <Card key={proxy.id} className="hover:shadow-md transition-shadow">
                            <CardContent className="p-6">
                                <div className="flex items-start justify-between">
                                    <div className="flex-1">
                                        <div className="flex items-center gap-3 mb-3">
                                            <div className="flex items-center gap-2">
                                                <Server className="w-5 h-5 text-gray-600" />
                                                <h3 className="font-semibold text-lg">
                                                    {proxy.protocol}://{proxy.ip}:{proxy.port}
                                                </h3>
                                            </div>
                                            <span className={`text-xs px-2 py-1 rounded-full font-medium ${getStatusStyle(proxy.status)}`}>
                                                {ProxyStatusLabel[proxy.status as keyof typeof ProxyStatusLabel]}
                                            </span>
                                        </div>
                                        <div className="grid grid-cols-2 gap-4 text-sm text-gray-600 mb-3">
                                            <div>
                                                <span className="text-gray-500">Region:</span> {proxy.region || "N/A"}
                                            </div>
                                            <div>
                                                <span className="text-gray-500">Username:</span> {proxy.username || "None"}
                                            </div>
                                            <div>
                                                <span className="text-gray-500">Success:</span> {proxy.success_count}
                                            </div>
                                            <div>
                                                <span className="text-gray-500">Failed:</span> {proxy.fail_count}
                                            </div>
                                            {proxy.last_check_at && (
                                                <div>
                                                    <span className="text-gray-500">Last check:</span> {proxy.last_check_at}
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => handleHealthCheck(proxy.id)}
                                            disabled={checkingId === proxy.id}
                                            title="Health check"
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
