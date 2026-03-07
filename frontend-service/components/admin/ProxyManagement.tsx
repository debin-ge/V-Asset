"use client"

import type { ReactNode } from "react"
import { useEffect, useState } from "react"
import { Activity, Clock3, RefreshCw, Route, ShieldCheck, ShieldX } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { proxyApi, type ProxySourceStatus } from "@/lib/api/proxy"

export function ProxyManagement() {
    const [status, setStatus] = useState<ProxySourceStatus | null>(null)
    const [loading, setLoading] = useState(true)
    const [probing, setProbing] = useState(false)

    const loadStatus = async () => {
        setLoading(true)
        try {
            const result = await proxyApi.getSourceStatus()
            setStatus(result)
        } catch (error) {
            toast.error("Failed to load dynamic proxy source status")
            console.error(error)
        } finally {
            setLoading(false)
        }
    }

    const handleProbe = async () => {
        setProbing(true)
        try {
            const result = await proxyApi.getSourceStatus()
            setStatus(result)
            if (result.healthy) {
                toast.success("Dynamic proxy lease acquired successfully")
            } else {
                toast.error(result.message || "Dynamic proxy source is unavailable")
            }
        } catch (error) {
            toast.error("Failed to probe dynamic proxy source")
            console.error(error)
        } finally {
            setProbing(false)
        }
    }

    useEffect(() => {
        loadStatus()
    }, [])

    return (
        <div className="space-y-6">
            <Card className="border-orange-200 bg-gradient-to-br from-orange-50 via-white to-amber-50">
                <CardContent className="p-6 space-y-4">
                    <div className="flex items-start justify-between gap-4">
                        <div className="space-y-2">
                            <div className="inline-flex items-center gap-2 rounded-full bg-orange-100 px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] text-orange-700">
                                <Route className="h-3.5 w-3.5" />
                                Dynamic Proxy Flow
                            </div>
                            <h2 className="text-xl font-semibold text-slate-900">Proxy leases are acquired per task now</h2>
                            <p className="max-w-3xl text-sm leading-6 text-slate-600">
                                The downloader no longer stores static proxy endpoints in the admin panel. Before every parse request,
                                the backend acquires a fresh proxy lease from the configured provider and reuses the same lease for the
                                matching yt-dlp parse and download task.
                            </p>
                        </div>
                        <Button variant="outline" onClick={handleProbe} disabled={probing}>
                            <RefreshCw className={`mr-2 h-4 w-4 ${probing ? "animate-spin" : ""}`} />
                            Probe Source
                        </Button>
                    </div>

                    <div className="grid gap-3 md:grid-cols-3">
                        <div className="rounded-2xl border border-orange-100 bg-white/80 p-4">
                            <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Step 1</p>
                            <p className="mt-2 text-sm font-medium text-slate-900">Parse requests a fresh lease</p>
                            <p className="mt-1 text-sm text-slate-600">Every parse call hits the proxy provider before yt-dlp starts.</p>
                        </div>
                        <div className="rounded-2xl border border-orange-100 bg-white/80 p-4">
                            <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Step 2</p>
                            <p className="mt-2 text-sm font-medium text-slate-900">Lease metadata flows through MQ</p>
                            <p className="mt-1 text-sm text-slate-600">`proxy_url` and `proxy_lease_id` are attached to the download task.</p>
                        </div>
                        <div className="rounded-2xl border border-orange-100 bg-white/80 p-4">
                            <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Step 3</p>
                            <p className="mt-2 text-sm font-medium text-slate-900">Download must reuse the same lease</p>
                            <p className="mt-1 text-sm text-slate-600">Workers now fail fast if a task arrives without a proxy lease.</p>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {loading ? (
                <Card>
                    <CardContent className="flex items-center justify-center py-12">
                        <RefreshCw className="h-6 w-6 animate-spin text-slate-400" />
                    </CardContent>
                </Card>
            ) : (
                <div className="grid gap-4 lg:grid-cols-[1.2fr,0.8fr]">
                    <Card>
                        <CardContent className="p-6 space-y-5">
                            <div className="flex items-center justify-between">
                                <div>
                                    <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Current Status</p>
                                    <h3 className="mt-1 text-lg font-semibold text-slate-900">Dynamic proxy source probe</h3>
                                </div>
                                <div className={`inline-flex items-center gap-2 rounded-full px-3 py-1 text-sm font-medium ${status?.healthy ? "bg-emerald-100 text-emerald-700" : "bg-rose-100 text-rose-700"}`}>
                                    {status?.healthy ? <ShieldCheck className="h-4 w-4" /> : <ShieldX className="h-4 w-4" />}
                                    {status?.healthy ? "Healthy" : "Unavailable"}
                                </div>
                            </div>

                            <div className="grid gap-4 sm:grid-cols-2">
                                <InfoItem icon={<Activity className="h-4 w-4" />} label="Mode" value={status?.mode || "dynamic-lease"} />
                                <InfoItem icon={<Clock3 className="h-4 w-4" />} label="Last Probe" value={status?.checked_at || "N/A"} />
                                <InfoItem icon={<Route className="h-4 w-4" />} label="Lease ID" value={status?.proxy_lease_id || "N/A"} />
                                <InfoItem icon={<Clock3 className="h-4 w-4" />} label="Lease Expiry" value={status?.proxy_expire_at || "N/A"} />
                            </div>

                            <div className="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                                <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Resolved Proxy URL</p>
                                <p className="mt-2 break-all font-mono text-sm text-slate-700">
                                    {status?.proxy_url || "No lease returned yet"}
                                </p>
                            </div>

                            <div className="rounded-2xl border border-slate-200 bg-white p-4">
                                <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Backend Message</p>
                                <p className="mt-2 text-sm leading-6 text-slate-600">
                                    {status?.message || "No status available"}
                                </p>
                            </div>
                        </CardContent>
                    </Card>

                    <Card>
                        <CardContent className="p-6 space-y-4">
                            <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Operational Notes</p>
                            <ul className="space-y-3 text-sm leading-6 text-slate-600">
                                <li>The previous add, delete and static health-check workflow has been retired for downloads.</li>
                                <li>If probing fails, verify `PROXY_API_ENDPOINT` and `PROXY_API_KEY` on the asset service.</li>
                                <li>Proxy credentials are masked here on purpose. Actual credentials only stay inside backend services.</li>
                                <li>Cached parse metadata no longer reuses old leases. A fresh lease is attached every time a task starts.</li>
                            </ul>
                        </CardContent>
                    </Card>
                </div>
            )}
        </div>
    )
}

function InfoItem({ icon, label, value }: { icon: ReactNode; label: string; value: string }) {
    return (
        <div className="rounded-2xl border border-slate-200 bg-white p-4">
            <div className="flex items-center gap-2 text-slate-500">
                {icon}
                <p className="text-xs uppercase tracking-[0.2em]">{label}</p>
            </div>
            <p className="mt-3 break-all text-sm font-medium text-slate-900">{value}</p>
        </div>
    )
}
