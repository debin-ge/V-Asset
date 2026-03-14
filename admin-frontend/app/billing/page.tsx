"use client";

import * as React from "react";
import { Search, Wallet } from "lucide-react";
import { toast } from "sonner";

import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { StatusBadge } from "@/components/common/StatusBadge";
import { AppShell } from "@/components/layout/AppShell";
import { PageHeader } from "@/components/layout/PageHeader";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { billingApi } from "@/lib/api/billing";
import type { BillingAccount, BillingLedgerEntry, BillingPricing, BillingShortfallOrder, BillingUsageRecord } from "@/types/billing";

export default function BillingPage() {
  const [query, setQuery] = React.useState("");
  const [accounts, setAccounts] = React.useState<BillingAccount[]>([]);
  const [selectedUserId, setSelectedUserId] = React.useState("");
  const [selectedAccount, setSelectedAccount] = React.useState<BillingAccount | null>(null);
  const [ledgerItems, setLedgerItems] = React.useState<BillingLedgerEntry[]>([]);
  const [shortfallItems, setShortfallItems] = React.useState<BillingShortfallOrder[]>([]);
  const [usageItems, setUsageItems] = React.useState<BillingUsageRecord[]>([]);
  const [pricing, setPricing] = React.useState<BillingPricing | null>(null);
  const [isLoading, setIsLoading] = React.useState(true);
  const [adjustAmount, setAdjustAmount] = React.useState("");
  const [adjustRemark, setAdjustRemark] = React.useState("");
  const [reconcileRemark, setReconcileRemark] = React.useState("");
  const [reconcilingOrderNo, setReconcilingOrderNo] = React.useState("");
  const [pricingForm, setPricingForm] = React.useState({
    ingress_price_fen_per_gib: "0",
    egress_price_fen_per_gib: "0",
    default_estimate_bytes: "104857600",
    remark: "",
  });

  const shortfallTotalFen = React.useMemo(
    () => shortfallItems.reduce((sum, item) => sum + item.shortfall_fen, 0),
    [shortfallItems]
  );
  const usageTrafficBytes = React.useMemo(
    () => usageItems.reduce((sum, item) => sum + item.traffic_bytes, 0),
    [usageItems]
  );

  const loadAccounts = React.useCallback(async (targetQuery = query) => {
    const response = await billingApi.listAccounts({ query: targetQuery, page: 1, page_size: 20 });
    setAccounts(response.items || []);

    if (response.items?.length) {
      const nextUserId = response.items.some((item) => item.user_id === selectedUserId)
        ? selectedUserId
        : response.items[0].user_id;
      setSelectedUserId(nextUserId);
      return nextUserId;
    }

    setSelectedUserId("");
    setSelectedAccount(null);
    setLedgerItems([]);
    setShortfallItems([]);
    setUsageItems([]);
    return "";
  }, [query, selectedUserId]);

  const loadAccountDetail = React.useCallback(async (userId: string) => {
    if (!userId) {
      return;
    }

    const [account, shortfalls, ledger, usage] = await Promise.all([
      billingApi.getAccountDetail(userId),
      billingApi.listShortfalls({ user_id: userId, page: 1, page_size: 20 }),
      billingApi.listLedger({ user_id: userId, page: 1, page_size: 20 }),
      billingApi.listUsageRecords({ user_id: userId, page: 1, page_size: 20 }),
    ]);

    setSelectedAccount(account);
    setShortfallItems(shortfalls.items || []);
    setLedgerItems(ledger.items || []);
    setUsageItems(usage.items || []);
  }, []);

  const loadPricing = React.useCallback(async () => {
    const pricingResponse = await billingApi.getPricing();
    setPricing(pricingResponse);
    setPricingForm({
      ingress_price_fen_per_gib: pricingResponse.ingress_price_fen_per_gib,
      egress_price_fen_per_gib: pricingResponse.egress_price_fen_per_gib,
      default_estimate_bytes: String(pricingResponse.default_estimate_bytes),
      remark: pricingResponse.remark || "",
    });
  }, []);

  const loadPage = React.useCallback(async () => {
    setIsLoading(true);
    try {
      const userId = await loadAccounts();
      await Promise.all([
        loadPricing(),
        userId ? loadAccountDetail(userId) : Promise.resolve(),
      ]);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to load billing data");
    } finally {
      setIsLoading(false);
    }
  }, [loadAccountDetail, loadAccounts, loadPricing]);

  React.useEffect(() => {
    void loadPage();
  }, [loadPage]);

  const handleSearch = async () => {
    try {
      setIsLoading(true);
      const userId = await loadAccounts(query);
      if (userId) {
        await loadAccountDetail(userId);
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to search accounts");
    } finally {
      setIsLoading(false);
    }
  };

  const handleSelectAccount = async (userId: string) => {
    setSelectedUserId(userId);
    try {
      await loadAccountDetail(userId);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to load account detail");
    }
  };

  const handleAdjustBalance = async () => {
    if (!selectedUserId) {
      toast.error("Select an account first");
      return;
    }
    if (!adjustAmount || !adjustRemark.trim()) {
      toast.error("Amount and remark are required");
      return;
    }

    const amountFen = Math.round(Number(adjustAmount) * 100);
    if (!Number.isFinite(amountFen) || amountFen === 0) {
      toast.error("Enter a valid amount");
      return;
    }

    try {
      await billingApi.adjustBalance(selectedUserId, {
        amount_fen: amountFen,
        remark: adjustRemark.trim(),
      });
      setAdjustAmount("");
      setAdjustRemark("");
      await Promise.all([loadAccounts(), loadAccountDetail(selectedUserId)]);
      toast.success("Balance updated");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update balance");
    }
  };

  const handleUpdatePricing = async () => {
    try {
      await billingApi.updatePricing({
        ingress_price_fen_per_gib: pricingForm.ingress_price_fen_per_gib,
        egress_price_fen_per_gib: pricingForm.egress_price_fen_per_gib,
        default_estimate_bytes: Number(pricingForm.default_estimate_bytes),
        remark: pricingForm.remark,
      });
      await loadPricing();
      toast.success("Pricing updated");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update pricing");
    }
  };

  const handleReconcileShortfall = async (orderNo: string) => {
    if (!selectedUserId) {
      toast.error("Select an account first");
      return;
    }

    if (!window.confirm(`Reconcile shortfall order ${orderNo}?`)) {
      return;
    }

    try {
      setReconcilingOrderNo(orderNo);
      await billingApi.reconcileShortfall(orderNo, {
        remark: reconcileRemark.trim() || undefined,
      });
      setReconcileRemark("");
      await Promise.all([loadAccounts(), loadAccountDetail(selectedUserId)]);
      toast.success("Shortfall reconciled");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to reconcile shortfall");
    } finally {
      setReconcilingOrderNo("");
    }
  };

  return (
    <ProtectedRoute>
      <AppShell>
        <div className="space-y-4">
          <PageHeader
            eyebrow="Commercial Billing"
            title="Billing"
            description="搜索用户、调整余额、核查流水，并维护平台统一费率。"
            actions={<Button onClick={() => void loadPage()}>Refresh</Button>}
          />

          <div className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
            <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
              <CardHeader>
                <CardTitle>User Accounts</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex gap-3">
                  <div className="relative flex-1">
                    <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-400" />
                    <Input
                      className="pl-9"
                      placeholder="Search by email, nickname, or user ID"
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") {
                          void handleSearch();
                        }
                      }}
                    />
                  </div>
                  <Button onClick={() => void handleSearch()}>Search</Button>
                </div>

                <div className="overflow-hidden rounded-2xl border border-slate-100">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>User</TableHead>
                        <TableHead>Available</TableHead>
                        <TableHead>Reserved</TableHead>
                        <TableHead>Total Spent</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {accounts.map((account) => (
                        <TableRow
                          key={account.user_id}
                          className={selectedUserId === account.user_id ? "bg-slate-50" : ""}
                          onClick={() => void handleSelectAccount(account.user_id)}
                        >
                          <TableCell>
                            <div>
                              <p className="font-medium text-slate-900">{account.nickname || "Unknown user"}</p>
                              <p className="text-xs text-slate-500">{account.email || account.user_id}</p>
                            </div>
                          </TableCell>
                          <TableCell>{formatCurrencyFen(account.available_balance_fen)}</TableCell>
                          <TableCell>{formatCurrencyFen(account.reserved_balance_fen)}</TableCell>
                          <TableCell>{formatCurrencyFen(account.total_spent_fen)}</TableCell>
                        </TableRow>
                      ))}
                      {!accounts.length && !isLoading ? (
                        <TableRow>
                          <TableCell colSpan={4} className="py-8 text-center text-sm text-slate-500">
                            No billing accounts found.
                          </TableCell>
                        </TableRow>
                      ) : null}
                    </TableBody>
                  </Table>
                </div>
              </CardContent>
            </Card>

            <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
              <CardHeader>
                <CardTitle>Pricing</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-3 md:grid-cols-2">
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-slate-700">Ingress Price (fen / GiB)</label>
                    <Input
                      value={pricingForm.ingress_price_fen_per_gib}
                      onChange={(e) => setPricingForm((prev) => ({ ...prev, ingress_price_fen_per_gib: e.target.value }))}
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-slate-700">Egress Price (fen / GiB)</label>
                    <Input
                      value={pricingForm.egress_price_fen_per_gib}
                      onChange={(e) => setPricingForm((prev) => ({ ...prev, egress_price_fen_per_gib: e.target.value }))}
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700">Default Estimate Bytes</label>
                  <Input
                    value={pricingForm.default_estimate_bytes}
                    onChange={(e) => setPricingForm((prev) => ({ ...prev, default_estimate_bytes: e.target.value }))}
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700">Remark</label>
                  <Textarea
                    value={pricingForm.remark}
                    onChange={(e) => setPricingForm((prev) => ({ ...prev, remark: e.target.value }))}
                    placeholder="Pricing change note"
                  />
                </div>
                {pricing ? (
                  <div className="rounded-2xl bg-slate-50 p-4 text-sm text-slate-600">
                    Active version #{pricing.version} · updated {formatDateTime(pricing.effective_at)}
                  </div>
                ) : null}
                <Button onClick={() => void handleUpdatePricing()}>Update Pricing</Button>
              </CardContent>
            </Card>
          </div>

          <div className="grid gap-4 xl:grid-cols-[0.8fr_1.2fr]">
            <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
              <CardHeader>
                <CardTitle>Account Detail</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {!selectedAccount ? (
                  <div className="rounded-2xl border border-dashed border-slate-200 px-4 py-10 text-center text-sm text-slate-500">
                    Select a billing account to inspect details.
                  </div>
                ) : (
                  <>
                    <div className="grid gap-3 sm:grid-cols-2">
                      <Metric label="Available" value={formatCurrencyFen(selectedAccount.available_balance_fen)} />
                      <Metric label="Reserved" value={formatCurrencyFen(selectedAccount.reserved_balance_fen)} />
                      <Metric label="Total Traffic" value={formatFileSize(selectedAccount.total_traffic_bytes)} />
                      <Metric label="Total Recharged" value={formatCurrencyFen(selectedAccount.total_recharged_fen)} />
                      <Metric label="Total Spent" value={formatCurrencyFen(selectedAccount.total_spent_fen)} />
                      <Metric label="Shortfalls" value={formatCurrencyFen(shortfallTotalFen)} />
                    </div>

                    <div className="rounded-2xl border border-slate-100 bg-slate-50/80 p-4">
                      <p className="text-xs uppercase tracking-wide text-slate-400">Selected account</p>
                      <div className="mt-2 flex flex-wrap items-center gap-2">
                        <p className="text-base font-semibold text-slate-950">
                          {selectedAccount.nickname || "Unknown user"}
                        </p>
                        <StatusBadge
                          label={accountStatusLabel(selectedAccount.status)}
                          tone={accountStatusTone(selectedAccount.status)}
                        />
                      </div>
                      <p className="mt-1 text-sm text-slate-600">
                        {selectedAccount.email || selectedAccount.user_id}
                      </p>
                      <p className="mt-1 text-xs text-slate-500">User ID: {selectedAccount.user_id}</p>
                    </div>

                    <div className="rounded-2xl border border-slate-100 p-4">
                      <div className="mb-3 flex items-center gap-2 text-slate-900">
                        <Wallet className="size-4" />
                        <p className="font-medium">Manual Adjustment</p>
                      </div>
                      <div className="space-y-3">
                        <div className="space-y-2">
                          <label className="text-sm font-medium text-slate-700">Amount (CNY)</label>
                          <Input
                            placeholder="10.00 or -5.00"
                            value={adjustAmount}
                            onChange={(e) => setAdjustAmount(e.target.value)}
                          />
                        </div>
                        <div className="space-y-2">
                          <label className="text-sm font-medium text-slate-700">Remark</label>
                          <Textarea
                            placeholder="Why are you adjusting this account?"
                            value={adjustRemark}
                            onChange={(e) => setAdjustRemark(e.target.value)}
                          />
                        </div>
                        <Button onClick={() => void handleAdjustBalance()}>Submit Adjustment</Button>
                      </div>
                    </div>
                  </>
                )}
              </CardContent>
            </Card>

            <div className="space-y-4">
              <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
                <CardHeader>
                  <CardTitle>Shortfalls</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid gap-3 md:grid-cols-3">
                    <Metric label="Pending Orders" value={String(shortfallItems.length)} />
                    <Metric label="Shortfall Total" value={formatCurrencyFen(shortfallTotalFen)} />
                    <Metric label="Usage Loaded" value={formatFileSize(usageTrafficBytes)} />
                  </div>

                  <div className="space-y-2">
                    <label className="text-sm font-medium text-slate-700">Reconcile Remark</label>
                    <Textarea
                      placeholder="Optional note for this reconciliation"
                      value={reconcileRemark}
                      onChange={(e) => setReconcileRemark(e.target.value)}
                    />
                  </div>

                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Time</TableHead>
                        <TableHead>Scene</TableHead>
                        <TableHead>Traffic</TableHead>
                        <TableHead>Shortfall</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Remark</TableHead>
                        <TableHead className="text-right">Action</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {shortfallItems.map((item) => (
                        <TableRow key={item.order_no}>
                          <TableCell>{formatDateTime(item.updated_at || item.created_at)}</TableCell>
                          <TableCell>
                            <StatusBadge label={sceneLabel(item.scene)} tone={sceneTone(item.scene)} />
                          </TableCell>
                          <TableCell>{formatFileSize(totalTrafficBytes(item))}</TableCell>
                          <TableCell className="font-medium text-amber-700">{formatCurrencyFen(item.shortfall_fen)}</TableCell>
                          <TableCell>
                            <StatusBadge label={orderStatusLabel(item.status)} tone={orderStatusTone(item.status)} />
                          </TableCell>
                          <TableCell>
                            <div className="space-y-1">
                              <p className="text-sm text-slate-700">{item.remark || "-"}</p>
                              <p className="font-mono text-[11px] text-slate-400">{item.order_no}</p>
                            </div>
                          </TableCell>
                          <TableCell className="text-right">
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={item.shortfall_fen <= 0 || reconcilingOrderNo === item.order_no}
                              onClick={() => void handleReconcileShortfall(item.order_no)}
                            >
                              {reconcilingOrderNo === item.order_no ? "Reconciling..." : "Reconcile"}
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                      {!shortfallItems.length ? (
                        <TableRow>
                          <TableCell colSpan={7} className="py-8 text-center text-sm text-slate-500">
                            No pending shortfalls.
                          </TableCell>
                        </TableRow>
                      ) : null}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>

              <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
                <CardHeader>
                  <CardTitle>Ledger</CardTitle>
                </CardHeader>
                <CardContent>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Time</TableHead>
                        <TableHead>Type</TableHead>
                        <TableHead>Amount</TableHead>
                        <TableHead>Balances</TableHead>
                        <TableHead>Remark</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {ledgerItems.map((item) => (
                        <TableRow key={item.entry_no}>
                          <TableCell>{formatDateTime(item.created_at)}</TableCell>
                          <TableCell>
                            <StatusBadge label={ledgerTypeLabel(item.entry_type)} tone={ledgerTypeTone(item.entry_type)} />
                          </TableCell>
                          <TableCell>{formatCurrencyFen(item.action_amount_fen)}</TableCell>
                          <TableCell>
                            <div className="text-xs text-slate-600">
                              <div>Avail {formatCurrencyFen(item.balance_after_available_fen)}</div>
                              <div>Reserved {formatCurrencyFen(item.balance_after_reserved_fen)}</div>
                            </div>
                          </TableCell>
                          <TableCell>{item.remark || "-"}</TableCell>
                        </TableRow>
                      ))}
                      {!ledgerItems.length ? (
                        <TableRow>
                          <TableCell colSpan={5} className="py-8 text-center text-sm text-slate-500">
                            No ledger records.
                          </TableCell>
                        </TableRow>
                      ) : null}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>

              <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
                <CardHeader>
                  <CardTitle>Traffic Usage</CardTitle>
                </CardHeader>
                <CardContent>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Time</TableHead>
                        <TableHead>Direction</TableHead>
                        <TableHead>Traffic</TableHead>
                        <TableHead>Amount</TableHead>
                        <TableHead>Source</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {usageItems.map((item) => (
                        <TableRow key={item.usage_no}>
                          <TableCell>{formatDateTime(item.created_at)}</TableCell>
                          <TableCell>
                            <StatusBadge label={item.direction === 1 ? "Ingress" : "Egress"} tone={directionTone(item.direction)} />
                          </TableCell>
                          <TableCell>{formatFileSize(item.traffic_bytes)}</TableCell>
                          <TableCell>{formatCurrencyFen(item.amount_fen)}</TableCell>
                          <TableCell>{item.source_service}</TableCell>
                        </TableRow>
                      ))}
                      {!usageItems.length ? (
                        <TableRow>
                          <TableCell colSpan={5} className="py-8 text-center text-sm text-slate-500">
                            No usage records.
                          </TableCell>
                        </TableRow>
                      ) : null}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            </div>
          </div>
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-slate-100 bg-slate-50 p-4">
      <p className="text-xs uppercase tracking-wide text-slate-400">{label}</p>
      <p className="mt-2 text-lg font-semibold text-slate-900">{value}</p>
    </div>
  );
}

function formatCurrencyFen(amountFen: number) {
  return new Intl.NumberFormat("zh-CN", {
    style: "currency",
    currency: "CNY",
    minimumFractionDigits: 2,
  }).format(amountFen / 100);
}

function formatFileSize(bytes: number) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const unitIndex = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / Math.pow(1024, unitIndex);
  return `${value.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

function formatDateTime(value: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function ledgerTypeLabel(entryType: number) {
  switch (entryType) {
    case 1:
      return "Top-up";
    case 2:
      return "Adjustment";
    case 3:
      return "Hold";
    case 4:
      return "Capture";
    case 5:
      return "Release";
    default:
      return "Ledger";
  }
}

function accountStatusLabel(status: number) {
  switch (status) {
    case 1:
      return "Active";
    case 2:
      return "Frozen";
    case 3:
      return "Closed";
    default:
      return "Unknown";
  }
}

function accountStatusTone(status: number): "success" | "warning" | "danger" | "neutral" {
  switch (status) {
    case 1:
      return "success";
    case 2:
      return "warning";
    case 3:
      return "danger";
    default:
      return "neutral";
  }
}

function orderStatusLabel(status: number) {
  switch (status) {
    case 1:
      return "Held";
    case 2:
      return "Partial";
    case 3:
      return "Captured";
    case 4:
      return "Released";
    case 5:
      return "Awaiting Shortfall";
    default:
      return "Unknown";
  }
}

function orderStatusTone(status: number): "success" | "warning" | "danger" | "info" | "neutral" {
  switch (status) {
    case 3:
      return "success";
    case 5:
      return "warning";
    case 4:
      return "neutral";
    case 2:
      return "info";
    default:
      return "neutral";
  }
}

function sceneLabel(scene: number) {
  switch (scene) {
    case 1:
      return "Download";
    case 2:
      return "Redownload";
    case 3:
      return "Admin";
    default:
      return "Unknown";
  }
}

function sceneTone(scene: number): "success" | "warning" | "info" | "neutral" {
  switch (scene) {
    case 1:
      return "info";
    case 2:
      return "success";
    case 3:
      return "neutral";
    default:
      return "neutral";
  }
}

function ledgerTypeTone(entryType: number): "success" | "warning" | "danger" | "info" | "neutral" {
  switch (entryType) {
    case 1:
      return "success";
    case 2:
      return "warning";
    case 3:
      return "info";
    case 4:
      return "success";
    case 5:
      return "neutral";
    default:
      return "neutral";
  }
}

function directionTone(direction: number): "info" | "success" {
  return direction === 1 ? "info" : "success";
}

function totalTrafficBytes(item: BillingShortfallOrder) {
  if (item.actual_traffic_bytes > 0) {
    return item.actual_traffic_bytes;
  }
  return item.actual_ingress_bytes + item.actual_egress_bytes;
}
