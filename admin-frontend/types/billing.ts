export interface BillingAccount {
  user_id: string;
  email?: string;
  nickname?: string;
  available_balance_fen: string;
  reserved_balance_fen: string;
  total_recharged_fen: string;
  total_spent_fen: string;
  total_traffic_bytes: number;
  status: number;
  version: number;
  updated_at: string;
}

export interface BillingAccountListResponse {
  total: number;
  page: number;
  page_size: number;
  items: BillingAccount[];
}

export interface BillingLedgerEntry {
  entry_no: string;
  user_id: string;
  email?: string;
  nickname?: string;
  order_no: string;
  hold_no: string;
  history_id: number;
  task_id: string;
  transfer_id: string;
  operation_id: string;
  entry_type: number;
  scene: number;
  action_amount_fen: string;
  available_delta_fen: string;
  reserved_delta_fen: string;
  balance_after_available_fen: string;
  balance_after_reserved_fen: string;
  operator_user_id: string;
  remark: string;
  created_at: string;
}

export interface BillingLedgerListResponse {
  total: number;
  page: number;
  page_size: number;
  items: BillingLedgerEntry[];
}

export interface BillingShortfallOrder {
  order_no: string;
  user_id: string;
  email?: string;
  nickname?: string;
  history_id: number;
  task_id: string;
  scene: number;
  status: number;
  pricing_version: number;
  actual_ingress_bytes: number;
  actual_egress_bytes: number;
  actual_traffic_bytes: number;
  held_amount_fen: string;
  captured_amount_fen: string;
  released_amount_fen: string;
  shortfall_fen: string;
  remark: string;
  created_at: string;
  updated_at: string;
}

export interface BillingShortfallListResponse {
  total: number;
  page: number;
  page_size: number;
  items: BillingShortfallOrder[];
}

export interface BillingUsageRecord {
  usage_no: string;
  order_no: string;
  user_id: string;
  email?: string;
  nickname?: string;
  history_id: number;
  task_id: string;
  transfer_id: string;
  direction: number;
  traffic_bytes: number;
  unit_price_fen_per_gib: string;
  amount_fen: string;
  pricing_version: number;
  source_service: string;
  status: number;
  created_at: string;
  confirmed_at?: string;
}

export interface BillingUsageListResponse {
  total: number;
  page: number;
  page_size: number;
  items: BillingUsageRecord[];
}

export interface BillingPricing {
  version: number;
  ingress_price_fen_per_gib: string;
  egress_price_fen_per_gib: string;
  enabled: boolean;
  remark: string;
  updated_by_user_id: string;
  effective_at: string;
  created_at: string;
}

export interface BillingBalanceAdjustmentPayload {
  operation_id?: string;
  amount_fen: string;
  remark: string;
}

export interface BillingPricingUpdatePayload {
  ingress_price_fen_per_gib: string;
  egress_price_fen_per_gib: string;
  remark?: string;
}

export interface BillingShortfallReconcilePayload {
  remark?: string;
}
