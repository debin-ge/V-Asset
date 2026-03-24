"use client"

import * as React from "react"
import { Loader2, Sparkles, AlertCircle } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { billingApi, BillingStatementItem } from "@/lib/api/billing"
import { formatCurrencyYuan, formatDate, formatFileSize } from "@/lib/format"
import { useAuth } from "@/hooks/use-auth"

const statementTypeOptions = [
    { value: 0, label: "All" },
    { value: 2, label: "Downloads" },
    { value: 1, label: "Recharges" },
    { value: 3, label: "Adjustments" },
]

const statementStatusOptions = [
    { value: 0, label: "Any status" },
    { value: 3, label: "Completed" },
    { value: 5, label: "Pending" },
    { value: 4, label: "Released" },
]

export function BillingStatements() {
    const { billingAccount } = useAuth()
    const [statements, setStatements] = React.useState<BillingStatementItem[]>([])
    const [isLoading, setIsLoading] = React.useState(true)
    const [error, setError] = React.useState<string | null>(null)
    const [statementType, setStatementType] = React.useState(0)
    const [statementStatus, setStatementStatus] = React.useState(0)

    const loadData = React.useCallback(async (silent = false) => {
        if (!silent) {
            setIsLoading(true)
        }
        setError(null)

        try {
            const statementResp = await billingApi.listStatements({
                page: 1,
                page_size: 20,
                ...(statementType ? { type: statementType } : {}),
                ...(statementStatus ? { status: statementStatus } : {}),
            })
            setStatements(statementResp.items || [])
        } catch (err) {
            console.error("Failed to load billing stats", err)
            setError("Failed to load billing statements")
        } finally {
            setIsLoading(false)
        }
    }, [statementStatus, statementType])

    // Fetch on mount or filter change
    React.useEffect(() => {
        void loadData()
    }, [loadData])

    // Refetch when billingAccount changes (e.g., after download completes)
    React.useEffect(() => {
        if (billingAccount) {
            void loadData(true)
        }
    }, [billingAccount, loadData])

    if (isLoading) {
        return (
            <div className="flex items-center justify-center py-12 text-gray-500">
                <Loader2 className="mr-2 h-6 w-6 animate-spin" />
                Loading...
            </div>
        )
    }

    if (error) {
        return (
            <div data-testid="billing-statements-error" className="py-12 text-center text-red-500 flex flex-col items-center gap-2">
                <AlertCircle className="h-8 w-8 text-red-400" />
                <p>{error}</p>
            </div>
        )
    }

    return (
        <div data-testid="billing-statements-panel" className="space-y-6">
            <Card className="border-slate-200/80 shadow-sm">
                <CardHeader className="gap-4">
                    <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                        <div className="space-y-1">
                            <CardTitle>Billing Statements</CardTitle>
                            <p className="text-sm text-slate-500">
                                Recent balance movements, download settlements, and manual adjustments.
                            </p>
                        </div>
                        <div className="rounded-2xl border border-slate-100 bg-slate-50 px-4 py-3 text-sm text-slate-600">
                            <div className="flex items-center gap-2 font-medium text-slate-900">
                                <Sparkles className="h-4 w-4 text-sky-500" />
                                {statements.length} recent records
                            </div>
                            <p className="mt-1 text-xs text-slate-500">Showing up to the latest 20 entries for your current filters.</p>
                        </div>
                    </div>

                    <div className="space-y-3">
                        <div className="flex flex-wrap items-center gap-2">
                            {statementTypeOptions.map((option) => (
                                <FilterPill
                                    key={`type-${option.value}`}
                                    active={statementType === option.value}
                                    onClick={() => setStatementType(option.value)}
                                >
                                    {option.label}
                                </FilterPill>
                            ))}
                        </div>
                        <div className="flex flex-wrap items-center gap-2">
                            {statementStatusOptions.map((option) => (
                                <FilterPill
                                    key={`status-${option.value}`}
                                    active={statementStatus === option.value}
                                    onClick={() => setStatementStatus(option.value)}
                                >
                                    {option.label}
                                </FilterPill>
                            ))}
                        </div>
                    </div>
                </CardHeader>
                <CardContent>
                    {statements.length === 0 ? (
                        <div data-testid="billing-statements-empty" className="rounded-2xl border border-dashed border-slate-200 bg-slate-50/70 px-4 py-10 text-center text-sm text-slate-500">
                            No billing records match the current filters.
                        </div>
                    ) : (
                        <div data-testid="billing-statements-table" className="space-y-3">
                            {statements.map((item) => (
                                <div
                                    key={item.statement_id}
                                    className="grid gap-3 rounded-2xl border border-slate-100 bg-white px-4 py-4 shadow-sm md:grid-cols-[1.3fr_0.9fr_0.9fr_0.8fr_1.3fr]"
                                >
                                    <div className="space-y-2">
                                        <div className="flex flex-wrap items-center gap-2">
                                            <Badge className={statementTypeTone(item.type)}>{getStatementTypeLabel(item.type)}</Badge>
                                            <Badge variant="outline" className={statementStatusTone(item.status)}>
                                                {getStatementStatusLabel(item.status)}
                                            </Badge>
                                        </div>
                                        <div className="text-xs text-slate-500">
                                            <p>{formatDate(item.created_at)}</p>
                                            {item.history_id > 0 ? <p>History #{item.history_id}</p> : null}
                                        </div>
                                    </div>
                                    <div>
                                        <p className="text-xs uppercase tracking-wide text-slate-400">Traffic</p>
                                        <p className="mt-1 text-sm font-medium text-slate-900">{formatFileSize(item.traffic_bytes)}</p>
                                    </div>
                                    <div>
                                        <p className="text-xs uppercase tracking-wide text-slate-400">Amount</p>
                                        <p className="mt-1 text-sm font-semibold text-slate-950">{formatCurrencyYuan(item.amount_fen)}</p>
                                    </div>
                                    <div>
                                        <p className="text-xs uppercase tracking-wide text-slate-400">Statement ID</p>
                                        <p className="mt-1 truncate font-mono text-xs text-slate-600">{item.statement_id}</p>
                                    </div>
                                    <div>
                                        <p className="text-xs uppercase tracking-wide text-slate-400">Remark</p>
                                        <p className="mt-1 text-sm text-slate-700">{item.remark || "-"}</p>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </CardContent>
            </Card>
        </div>
    )
}

function FilterPill({
    active,
    onClick,
    children,
}: {
    active: boolean
    onClick: () => void
    children: React.ReactNode
}) {
    return (
        <button
            type="button"
            onClick={onClick}
            className={[
                "rounded-full border px-3 py-1.5 text-sm transition-colors",
                active
                    ? "border-sky-500 bg-sky-500 text-white shadow-sm"
                    : "border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:bg-slate-50",
            ].join(" ")}
        >
            {children}
        </button>
    )
}

function getStatementTypeLabel(type: number): string {
    switch (type) {
        case 1:
            return "Recharge"
        case 2:
            return "Download"
        case 3:
            return "Adjustment"
        default:
            return "Billing"
    }
}

function getStatementStatusLabel(status: number): string {
    switch (status) {
        case 1:
            return "Held"
        case 2:
            return "Partial"
        case 3:
            return "Completed"
        case 4:
            return "Released"
        case 5:
            return "Pending"
        default:
            return "Unknown"
    }
}

function statementTypeTone(type: number): string {
    switch (type) {
        case 1:
            return "bg-emerald-100 text-emerald-700"
        case 2:
            return "bg-sky-100 text-sky-700"
        case 3:
            return "bg-violet-100 text-violet-700"
        default:
            return "bg-slate-100 text-slate-700"
    }
}

function statementStatusTone(status: number): string {
    switch (status) {
        case 3:
            return "border-emerald-200 bg-emerald-50 text-emerald-700"
        case 5:
            return "border-amber-200 bg-amber-50 text-amber-700"
        case 4:
            return "border-slate-200 bg-slate-50 text-slate-700"
        case 2:
            return "border-sky-200 bg-sky-50 text-sky-700"
        default:
            return "border-slate-200 bg-white text-slate-600"
    }
}
