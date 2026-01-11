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
    Clock
} from "lucide-react"

export function CookieManagement() {
    const [cookies, setCookies] = useState<CookieInfo[]>([])
    const [loading, setLoading] = useState(true)
    const [total, setTotal] = useState(0)
    const [page, setPage] = useState(1)
    const [isDialogOpen, setIsDialogOpen] = useState(false)

    const [filterPlatform, setFilterPlatform] = useState<string>("")

    // Form state
    const [formData, setFormData] = useState<CreateCookieRequest>({
        platform: "youtube",
        name: "",
        content: "",
        expire_at: "",
    })
    const [submitting, setSubmitting] = useState(false)

    // Load cookie list
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
            toast.error("Failed to load cookie list")
            console.error(error)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        loadCookies()
    }, [page, filterPlatform])

    // Create cookie
    const handleCreate = async () => {
        if (!formData.platform || !formData.name || !formData.content) {
            toast.error("Platform, name, and content cannot be empty")
            return
        }

        setSubmitting(true)
        try {
            // Format expire time: convert from YYYY-MM-DDTHH:MM to YYYY-MM-DD HH:MM:SS
            const requestData = {
                ...formData,
                expire_at: formData.expire_at
                    ? formData.expire_at.replace("T", " ") + ":00"
                    : "",
            }
            await cookieApi.create(requestData)
            toast.success("Cookie added successfully")
            setIsDialogOpen(false)
            setFormData({
                platform: "youtube",
                name: "",
                content: "",
                expire_at: "",
            })
            loadCookies()
        } catch (error) {
            toast.error("Failed to add cookie")
            console.error(error)
        } finally {
            setSubmitting(false)
        }
    }

    // Delete cookie
    const handleDelete = async (id: number) => {
        if (!confirm("Are you sure you want to delete this cookie?")) return

        try {
            await cookieApi.delete(id)
            toast.success("Cookie deleted")
            loadCookies()
        } catch (error) {
            toast.error("Failed to delete")
            console.error(error)
        }
    }

    // Calculate remaining time
    const getTimeRemaining = (expireAt: string | null) => {
        if (!expireAt) return "Never expires"

        const now = new Date()
        const expire = new Date(expireAt)
        const diff = expire.getTime() - now.getTime()

        if (diff < 0) return "Expired"

        const minutes = Math.floor(diff / 60000)
        const hours = Math.floor(minutes / 60)
        const days = Math.floor(hours / 24)

        if (days > 0) return `${days} days`
        if (hours > 0) return `${hours} hours`
        return `${minutes} minutes`
    }

    // Get status style
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

    return (
        <div className="space-y-6">
            {/* Toolbar */}
            <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center justify-between">
                <div className="flex items-center gap-2 flex-wrap">
                    <Button variant="outline" size="sm" onClick={loadCookies} disabled={loading}>
                        <RefreshCw className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`} />
                        Refresh
                    </Button>
                    <select
                        className="flex h-9 rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background"
                        value={filterPlatform}
                        onChange={(e) => {
                            setFilterPlatform(e.target.value)
                            setPage(1)
                        }}
                    >
                        <option value="">All platforms</option>
                        {PlatformOptions.map((platform) => (
                            <option key={platform.value} value={platform.value}>
                                {platform.label}
                            </option>
                        ))}
                    </select>
                    <span className="text-sm text-gray-500">Total: {total} cookies</span>
                </div>

                <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
                    <DialogTrigger asChild>
                        <Button>
                            <Plus className="w-4 h-4 mr-2" />
                            Add Cookie
                        </Button>
                    </DialogTrigger>
                    <DialogContent className="max-w-2xl">
                        <DialogHeader>
                            <DialogTitle>Add Cookie</DialogTitle>
                            <DialogDescription>
                                Add a new platform cookie. Expiration time defaults to 10 minutes from now, or you can customize it.
                            </DialogDescription>
                        </DialogHeader>
                        <div className="grid gap-4 py-4">
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">Platform *</label>
                                    <select
                                        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background"
                                        value={formData.platform}
                                        onChange={(e) => setFormData({ ...formData, platform: e.target.value })}
                                    >
                                        {PlatformOptions.map((platform) => (
                                            <option key={platform.value} value={platform.value}>
                                                {platform.label}
                                            </option>
                                        ))}
                                    </select>
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">Name *</label>
                                    <Input
                                        placeholder="e.g., session_cookie"
                                        value={formData.name}
                                        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                    />
                                </div>
                            </div>
                            <div className="space-y-2">
                                <label className="text-sm font-medium">Content *</label>
                                <textarea
                                    className="flex min-h-[100px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                                    placeholder="Paste cookie content here..."
                                    value={formData.content}
                                    onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                                />
                            </div>
                            <div className="space-y-2">
                                <label className="text-sm font-medium">Expiration Time</label>
                                <Input
                                    type="datetime-local"
                                    value={formData.expire_at}
                                    onChange={(e) => setFormData({ ...formData, expire_at: e.target.value })}
                                />
                                <p className="text-xs text-gray-500">
                                    Leave empty to default to 10 minutes from now
                                </p>
                            </div>
                        </div>
                        <DialogFooter>
                            <Button onClick={handleCreate} disabled={submitting}>
                                {submitting ? (
                                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                                ) : (
                                    "Add"
                                )}
                            </Button>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>
            </div>

            {/* Cookie list */}
            {loading ? (
                <div className="flex items-center justify-center py-12">
                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-400"></div>
                </div>
            ) : cookies.length === 0 ? (
                <Card>
                    <CardContent className="flex flex-col items-center justify-center py-12">
                        <CookieIcon className="w-12 h-12 text-gray-300 mb-4" />
                        <p className="text-gray-500">No cookies yet</p>
                        <p className="text-sm text-gray-400">Click the button above to add a cookie</p>
                    </CardContent>
                </Card>
            ) : (
                <div className="grid gap-4">
                    {cookies.map((cookie) => (
                        <Card key={cookie.id} className="hover:shadow-md transition-shadow">
                            <CardContent className="p-6">
                                <div className="flex items-start justify-between">
                                    <div className="flex-1">
                                        <div className="flex items-center gap-3 mb-3">
                                            <div className="flex items-center gap-2">
                                                <CookieIcon className="w-5 h-5 text-gray-600" />
                                                <h3 className="font-semibold text-lg">{cookie.name}</h3>
                                            </div>
                                            <span className="text-xs px-2 py-1 rounded bg-gray-100 text-gray-700">
                                                {cookie.platform}
                                            </span>
                                            <span className={`text-xs px-2 py-1 rounded-full font-medium ${getStatusStyle(cookie.status)}`}>
                                                {CookieStatusLabel[cookie.status as keyof typeof CookieStatusLabel]}
                                            </span>
                                        </div>
                                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm text-gray-600 mb-3">
                                            <div>
                                                <span className="text-gray-500">Use count:</span> {cookie.use_count}
                                            </div>
                                            <div>
                                                <span className="text-gray-500">Success:</span> {cookie.success_count}
                                            </div>
                                            <div>
                                                <span className="text-gray-500">Failed:</span> {cookie.fail_count}
                                            </div>
                                            <div className="flex items-center gap-1">
                                                <Clock className="w-3 h-3" />
                                                <span>{getTimeRemaining(cookie.expire_at)}</span>
                                            </div>
                                        </div>
                                        <div className="text-xs text-gray-500 space-y-1">
                                            {cookie.last_used_at && (
                                                <div>Last used: {cookie.last_used_at}</div>
                                            )}
                                            <div>Created: {cookie.created_at}</div>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-2">
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
