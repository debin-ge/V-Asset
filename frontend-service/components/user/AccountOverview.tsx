"use client"

import * as React from "react"
import { Loader2, RefreshCw, Shield, Wallet, Waves, Receipt } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { formatCurrencyYuan, formatDate, formatFileSize } from "@/lib/format"
import { useAuth } from "@/hooks/use-auth"

export function AccountOverview() {
    const { billingAccount, refreshBillingAccount, isLoading: isAuthLoading } = useAuth()
    const [isRefreshing, setIsRefreshing] = React.useState(false)
    const [error, setError] = React.useState<string | null>(null)

    const handleRefresh = async () => {
        setIsRefreshing(true)
        setError(null)
        try {
            await refreshBillingAccount()
        } catch (err) {
            console.error("Failed to refresh account", err)
            setError("Failed to refresh account overview")
        } finally {
            setIsRefreshing(false)
        }
    }

    if (isAuthLoading) {
        return (
            <div className="flex items-center justify-center py-12 text-gray-500">
                <Loader2 className="mr-2 h-6 w-6 animate-spin" />
                Loading...
            </div>
        )
    }

    if (error) {
        return <div className="py-12 text-center text-red-500">{error}</div>
    }

    if (!billingAccount) {
        return (
            <Card data-testid="account-overview-empty" className="border-slate-200/80 shadow-sm">
                <CardContent className="py-10 text-center text-slate-500">
                    No account information available.
                </CardContent>
            </Card>
        )
    }

    const accountCards = [
        {
            title: "Available Balance",
            value: formatCurrencyYuan(billingAccount.available_balance_yuan),
            description: "Ready for your next download request",
            icon: Wallet,
        },
        {
            title: "Reserved Amount",
            value: formatCurrencyYuan(billingAccount.reserved_balance_yuan),
            description: "Held for active download and transfer flows",
            icon: Shield,
        },
        {
            title: "Total Traffic",
            value: formatFileSize(billingAccount.total_traffic_bytes ?? 0),
            description: "Combined settled traffic across completed billing flows",
            icon: Waves,
        },
        {
            title: "Total Spent",
            value: formatCurrencyYuan(billingAccount.total_spent_yuan),
            description: "Settled billing amount already consumed",
            icon: Receipt,
        },
    ]

    return (
        <div className="space-y-6">
            <Card data-testid="account-overview-card" className="overflow-hidden border-blue-100 bg-gradient-to-br from-sky-50 via-white to-blue-50 shadow-sm">
                <CardContent className="flex flex-col gap-5 p-6 lg:flex-row lg:items-end lg:justify-between">
                    <div className="space-y-3">
                        <div className="flex flex-wrap items-center gap-2">
                            <Badge variant="outline" className="border-sky-200 bg-sky-100 text-sky-700">
                                Account Overview
                            </Badge>
                            <Badge className={accountStatusTone(billingAccount.status ?? 0)}>
                                {accountStatusLabel(billingAccount.status ?? 0)}
                            </Badge>
                        </div>
                        <div>
                            <p className="text-sm text-slate-600">Available balance</p>
                            <p data-testid="account-balance-value" className="mt-1 text-3xl font-semibold text-slate-950">
                                {formatCurrencyYuan(billingAccount.available_balance_yuan)}
                            </p>
                        </div>
                        <p className="max-w-2xl text-sm text-slate-600">
                            Download billing now settles against your account balance and aggregates traffic into a single user-facing total.
                        </p>
                    </div>

                    <div className="flex flex-col items-start gap-3 lg:items-end">
                        <div className="rounded-2xl border border-white/70 bg-white/80 px-4 py-3 text-sm text-slate-600 shadow-sm">
                            Last synced {formatDate(billingAccount.updated_at ?? "") || "just now"}
                        </div>
                        <Button variant="outline" onClick={handleRefresh} disabled={isRefreshing}>
                            {isRefreshing ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
                            Refresh
                        </Button>
                    </div>
                </CardContent>
            </Card>

            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                {accountCards.map((card) => {
                    const Icon = card.icon
                    return (
                        <Card key={card.title} className="border-slate-200/80 shadow-sm">
                            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                                <CardTitle className="text-sm font-medium text-slate-700">{card.title}</CardTitle>
                                <Icon className="h-4 w-4 text-slate-400" />
                            </CardHeader>
                            <CardContent>
                                <div className="text-2xl font-bold text-slate-950">{card.value}</div>
                                <p className="text-xs text-slate-500">{card.description}</p>
                            </CardContent>
                        </Card>
                    )
                })}
            </div>
        </div>
    )
}

function accountStatusLabel(status: number): string {
    switch (status) {
        case 1:
            return "Active"
        case 2:
            return "Frozen"
        case 3:
            return "Closed"
        default:
            return "Unknown"
    }
}

function accountStatusTone(status: number): string {
    switch (status) {
        case 1:
            return "bg-emerald-500 text-white"
        case 2:
            return "bg-amber-500 text-white"
        case 3:
            return "bg-slate-700 text-white"
        default:
            return "bg-slate-500 text-white"
    }
}
