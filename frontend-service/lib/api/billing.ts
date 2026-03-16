import apiClient from "../api-client"
import type { SelectedFormatPayload } from "./download"

export interface BillingAccount {
    user_id: string
    currency_code: string
    available_balance_fen: string
    reserved_balance_fen: string
    total_recharged_fen: string
    total_spent_fen: string
    total_traffic_bytes: number
    status: number
    version: number
    created_at: string
    updated_at: string
}

export interface BillingStatementItem {
    statement_id: string
    type: number
    history_id: number
    traffic_bytes: number
    amount_fen: string
    status: number
    remark: string
    created_at: string
}

export interface BillingStatementListResponse {
    total: number
    page: number
    page_size: number
    items: BillingStatementItem[]
}

export interface BillingEstimateRequest {
    url: string
    platform?: string
    mode?: string
    selected_format?: SelectedFormatPayload
}

export interface BillingEstimateResponse {
    estimated_traffic_bytes: number
    estimated_cost_fen: string
    pricing_version: number
    is_estimated: boolean
    estimate_reason?: string
}

export const billingApi = {
    getAccount: async (): Promise<BillingAccount> => {
        const response = await apiClient.get("/api/v1/user/account")
        return response.data as BillingAccount
    },

    listStatements: async (params?: { page?: number; page_size?: number; type?: number; status?: number }): Promise<BillingStatementListResponse> => {
        const response = await apiClient.get("/api/v1/user/billing/ledger", { params })
        return response.data as BillingStatementListResponse
    },

    estimateDownload: async (payload: BillingEstimateRequest): Promise<BillingEstimateResponse> => {
        const response = await apiClient.post("/api/v1/user/billing/estimate", payload)
        return response.data as BillingEstimateResponse
    },
}
