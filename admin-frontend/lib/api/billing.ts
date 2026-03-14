import apiClient from "@/lib/api-client";
import type {
  BillingAccount,
  BillingAccountListResponse,
  BillingBalanceAdjustmentPayload,
  BillingLedgerListResponse,
  BillingPricing,
  BillingShortfallListResponse,
  BillingShortfallReconcilePayload,
  BillingPricingUpdatePayload,
  BillingUsageListResponse,
} from "@/types/billing";

export const billingApi = {
  listAccounts: async (params?: { query?: string; page?: number; page_size?: number; status?: number }): Promise<BillingAccountListResponse> => {
    const response = await apiClient.get("/api/v1/admin/billing/accounts", { params });
    return response.data as BillingAccountListResponse;
  },

  getAccountDetail: async (userId: string): Promise<BillingAccount> => {
    const response = await apiClient.get(`/api/v1/admin/billing/accounts/${userId}`);
    return response.data as BillingAccount;
  },

  adjustBalance: async (userId: string, payload: BillingBalanceAdjustmentPayload): Promise<{ success: boolean; entry_no: string; account: BillingAccount }> => {
    const response = await apiClient.post(`/api/v1/admin/billing/accounts/${userId}/adjustments`, payload);
    return response.data as { success: boolean; entry_no: string; account: BillingAccount };
  },

  listLedger: async (params?: { user_id?: string; page?: number; page_size?: number; entry_type?: number }): Promise<BillingLedgerListResponse> => {
    const response = await apiClient.get("/api/v1/admin/billing/ledger", { params });
    return response.data as BillingLedgerListResponse;
  },

  listShortfalls: async (params?: { user_id?: string; page?: number; page_size?: number }): Promise<BillingShortfallListResponse> => {
    const response = await apiClient.get("/api/v1/admin/billing/shortfalls", { params });
    return response.data as BillingShortfallListResponse;
  },

  reconcileShortfall: async (orderNo: string, payload?: BillingShortfallReconcilePayload): Promise<{ success: boolean; entry_no: string }> => {
    const response = await apiClient.post(`/api/v1/admin/billing/shortfalls/${orderNo}/reconcile`, payload ?? {});
    return response.data as { success: boolean; entry_no: string };
  },

  listUsageRecords: async (params?: { user_id?: string; page?: number; page_size?: number; direction?: number }): Promise<BillingUsageListResponse> => {
    const response = await apiClient.get("/api/v1/admin/billing/usage-records", { params });
    return response.data as BillingUsageListResponse;
  },

  getPricing: async (): Promise<BillingPricing> => {
    const response = await apiClient.get("/api/v1/admin/billing/pricing");
    return response.data as BillingPricing;
  },

  updatePricing: async (payload: BillingPricingUpdatePayload): Promise<BillingPricing> => {
    const response = await apiClient.put("/api/v1/admin/billing/pricing", payload);
    return response.data as BillingPricing;
  },
};
