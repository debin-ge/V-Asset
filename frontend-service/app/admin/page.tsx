"use client"

import * as React from "react"
import { Suspense } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import { useAuth } from "@/hooks/use-auth"
import { ProxyManagement } from "@/components/admin/ProxyManagement"
import { CookieManagement } from "@/components/admin/CookieManagement"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Server, Cookie, Shield } from "lucide-react"

function AdminPageContent() {
    const { user, isLoading } = useAuth()
    const searchParams = useSearchParams()
    const router = useRouter()
    const tab = searchParams.get("tab") || "proxies"

    // 检查登录和权限
    React.useEffect(() => {
        if (!isLoading && !user) {
            router.push("/")
        }
    }, [user, isLoading, router])

    if (isLoading || !user) {
        return null
    }

    const handleTabChange = (value: string) => {
        router.push(`/admin?tab=${value}`)
    }

    return (
        <div className="container max-w-6xl mx-auto py-10 px-4">
            <div className="flex flex-col md:flex-row gap-8">
                <aside className="w-full md:w-64 shrink-0">
                    <div className="sticky top-24 space-y-4">
                        <div className="flex items-center gap-3 px-2 mb-6">
                            <div className="h-12 w-12 rounded-full bg-gradient-to-br from-orange-500 to-red-600 flex items-center justify-center text-white text-xl shadow-lg">
                                <Shield className="w-6 h-6" />
                            </div>
                            <div className="overflow-hidden">
                                <h2 className="font-bold">管理后台</h2>
                                <p className="text-xs text-gray-500">资源管理</p>
                            </div>
                        </div>

                        <Tabs value={tab} onValueChange={handleTabChange} orientation="vertical" className="w-full">
                            <TabsList className="flex flex-col h-auto bg-transparent space-y-1 p-0">
                                <TabsTrigger
                                    value="proxies"
                                    className="w-full justify-start px-4 py-3 data-[state=active]:bg-orange-50 data-[state=active]:text-orange-600 hover:bg-gray-50 transition-colors"
                                >
                                    <Server className="w-4 h-4 mr-3" />
                                    代理管理
                                </TabsTrigger>
                                <TabsTrigger
                                    value="cookies"
                                    className="w-full justify-start px-4 py-3 data-[state=active]:bg-orange-50 data-[state=active]:text-orange-600 hover:bg-gray-50 transition-colors"
                                >
                                    <Cookie className="w-4 h-4 mr-3" />
                                    Cookie 管理
                                </TabsTrigger>
                            </TabsList>
                        </Tabs>
                    </div>
                </aside>

                <main className="flex-1 min-w-0">
                    <div className="mb-6">
                        <h1 className="text-2xl font-bold">
                            {tab === "proxies" ? "代理管理" : "Cookie 管理"}
                        </h1>
                        <p className="text-gray-500">
                            {tab === "proxies"
                                ? "管理代理服务器，支持添加、删除和健康检查。"
                                : "管理平台 Cookie，支持添加、删除和冷冻操作。"}
                        </p>
                    </div>

                    <Tabs value={tab} className="w-full">
                        <TabsContent value="proxies" className="mt-0 focus-visible:outline-none animate-in fade-in slide-in-from-bottom-4 duration-500">
                            <ProxyManagement />
                        </TabsContent>
                        <TabsContent value="cookies" className="mt-0 focus-visible:outline-none animate-in fade-in slide-in-from-bottom-4 duration-500">
                            <CookieManagement />
                        </TabsContent>
                    </Tabs>
                </main>
            </div>
        </div>
    )
}

export default function AdminPage() {
    return (
        <Suspense fallback={<div className="flex items-center justify-center min-h-screen">Loading...</div>}>
            <AdminPageContent />
        </Suspense>
    )
}
