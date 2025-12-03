"use client"

import { useAuth } from "@/hooks/use-auth"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts"

const data = [
    { name: "Mon", downloads: 4 },
    { name: "Tue", downloads: 7 },
    { name: "Wed", downloads: 3 },
    { name: "Thu", downloads: 8 },
    { name: "Fri", downloads: 12 },
    { name: "Sat", downloads: 15 },
    { name: "Sun", downloads: 10 },
]

export function Stats() {
    const { user } = useAuth()

    if (!user) return null

    return (
        <div className="space-y-6">
            <div className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Total Downloads</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">328</div>
                        <p className="text-xs text-muted-foreground">+12% from last month</p>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Remaining Quota</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">{user.quota}</div>
                        <p className="text-xs text-muted-foreground">Resets in 8 hours</p>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Plan</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">Free</div>
                        <p className="text-xs text-muted-foreground">Upgrade for unlimited</p>
                    </CardContent>
                </Card>
            </div>

            <Card className="col-span-4">
                <CardHeader>
                    <CardTitle>Download Activity</CardTitle>
                </CardHeader>
                <CardContent className="pl-2">
                    <div className="h-[300px]">
                        <ResponsiveContainer width="100%" height="100%">
                            <BarChart data={data}>
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
        </div>
    )
}
