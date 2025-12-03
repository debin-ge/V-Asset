"use client"

import * as React from "react"
import { useSearchParams, useRouter } from "next/navigation"
import { useAuth } from "@/hooks/use-auth"
import { Profile } from "@/components/user/Profile"
import { History } from "@/components/user/History"
import { Stats } from "@/components/user/Stats"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { User, History as HistoryIcon, BarChart } from "lucide-react"

export default function UserPage() {
    const { user, isLoading } = useAuth()
    const searchParams = useSearchParams()
    const router = useRouter()
    const tab = searchParams.get("tab") || "profile"

    // Redirect if not logged in
    React.useEffect(() => {
        if (!isLoading && !user) {
            router.push("/")
        }
    }, [user, isLoading, router])

    if (isLoading || !user) {
        return null
    }

    const handleTabChange = (value: string) => {
        router.push(`/user?tab=${value}`)
    }

    return (
        <div className="container max-w-4xl mx-auto py-10 px-4">
            <div className="flex flex-col md:flex-row gap-8">
                <aside className="w-full md:w-64 shrink-0">
                    <div className="sticky top-24 space-y-4">
                        <div className="flex items-center gap-3 px-2 mb-6">
                            <div className="h-12 w-12 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-white text-xl font-bold shadow-lg">
                                {user.nickname[0].toUpperCase()}
                            </div>
                            <div className="overflow-hidden">
                                <h2 className="font-bold truncate">{user.nickname}</h2>
                                <p className="text-xs text-gray-500 truncate">{user.email}</p>
                            </div>
                        </div>

                        <Tabs value={tab} onValueChange={handleTabChange} orientation="vertical" className="w-full">
                            <TabsList className="flex flex-col h-auto bg-transparent space-y-1 p-0">
                                <TabsTrigger
                                    value="profile"
                                    className="w-full justify-start px-4 py-3 data-[state=active]:bg-blue-50 data-[state=active]:text-blue-600 hover:bg-gray-50 transition-colors"
                                >
                                    <User className="w-4 h-4 mr-3" />
                                    Profile
                                </TabsTrigger>
                                <TabsTrigger
                                    value="history"
                                    className="w-full justify-start px-4 py-3 data-[state=active]:bg-blue-50 data-[state=active]:text-blue-600 hover:bg-gray-50 transition-colors"
                                >
                                    <HistoryIcon className="w-4 h-4 mr-3" />
                                    History
                                </TabsTrigger>
                                <TabsTrigger
                                    value="stats"
                                    className="w-full justify-start px-4 py-3 data-[state=active]:bg-blue-50 data-[state=active]:text-blue-600 hover:bg-gray-50 transition-colors"
                                >
                                    <BarChart className="w-4 h-4 mr-3" />
                                    Stats
                                </TabsTrigger>
                            </TabsList>
                        </Tabs>
                    </div>
                </aside>

                <main className="flex-1 min-w-0">
                    <div className="mb-6">
                        <h1 className="text-2xl font-bold capitalize">{tab}</h1>
                        <p className="text-gray-500">Manage your {tab} settings and view details.</p>
                    </div>

                    <Tabs value={tab} className="w-full">
                        <TabsContent value="profile" className="mt-0 focus-visible:outline-none animate-in fade-in slide-in-from-bottom-4 duration-500">
                            <Profile />
                        </TabsContent>
                        <TabsContent value="history" className="mt-0 focus-visible:outline-none animate-in fade-in slide-in-from-bottom-4 duration-500">
                            <History />
                        </TabsContent>
                        <TabsContent value="stats" className="mt-0 focus-visible:outline-none animate-in fade-in slide-in-from-bottom-4 duration-500">
                            <Stats />
                        </TabsContent>
                    </Tabs>
                </main>
            </div>
        </div>
    )
}
