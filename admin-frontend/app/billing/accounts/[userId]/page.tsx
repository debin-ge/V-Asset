"use client";

import * as React from "react";
import Link from "next/link";
import { AlertTriangle, ArrowDownUp, Gauge, ReceiptText, UserRound, Wallet } from "lucide-react";
import { useParams } from "next/navigation";
import { toast } from "sonner";

import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { StatusBadge } from "@/components/common/StatusBadge";
import { AppShell } from "@/components/layout/AppShell";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { billingApi } from "@/lib/api/billing";
import { cn } from "@/lib/utils";
import type { BillingAccount, BillingLedgerEntry, BillingShortfallOrder, BillingUsageRecord } from "@/types/billing";

type DetailTab = "account" | "ledger" | "shortfalls" | "usage";

type LoadedTabs = {
  account: boolean;
  ledger: boolean;
  shortfalls: boolean;
  usage: boolean;
};

const INITIAL_LOADED_TABS: LoadedTabs = {
  account: false,
  ledger: false,
  shortfalls: false,
  usage: false,
};

export default function BillingAccountDetailPage() {
  const params = useParams<{ userId: string }>();
  const userId = React.useMemo(() => {
    if (!params?.userId) {
      return "";
    }
    return decodeURIComponent(params.userId);
  }, [params?.userId]);

  const [activeTab, setActiveTab] = React.useState<DetailTab>("account");
  const [loadedTabs, setLoadedTabs] = React.useState<LoadedTabs>(INITIAL_LOADED_TABS);

  const [account, setAccount] = React.useState<BillingAccount | null>(null);
  const [ledgerItems, setLedgerItems] = React.useState<BillingLedgerEntry[]>([]);
  const [shortfallItems, setShortfallItems] = React.useState<BillingShortfallOrder[]>([]);
  const [usageItems, setUsageItems] = React.useState<BillingUsageRecord[]>([]);

  const [isAccountLoading, setIsAccountLoading] = React.useState(true);
  const [isTabLoading, setIsTabLoading] = React.useState(false);

  const [adjustAmount, setAdjustAmount] = React.useState("");
  const [adjustRemark, setAdjustRemark] = React.useState("");
  const [reconcileRemark, setReconcileRemark] = React.useState("");
  const [reconcilingOrderNo, setReconcilingOrderNo] = React.useState("");

  const shortfallTotalYuan = React.useMemo(
    () => shortfallItems.reduce((sum, item) => sum + parseCurrency(item.shortfall_yuan), 0),
    [shortfallItems]
  );
  const usageTrafficBytes = React.useMemo(
    () => usageItems.reduce((sum, item) => sum + item.traffic_bytes, 0),
    [usageItems]
  );
  const usageTotalAmountYuan = React.useMemo(
    () => usageItems.reduce((sum, item) => sum + parseCurrency(item.amount_yuan), 0),
    [usageItems]
  );

  const loadAccountDetail = React.useCallback(async () => {
    if (!userId) {
      setAccount(null);
      return;
    }
    const response = await billingApi.getAccountDetail(userId);
    setAccount(response);
  }, [userId]);

  const loadLedger = React.useCallback(async () => {
    if (!userId) {
      setLedgerItems([]);
      return;
    }
    const response = await billingApi.listLedger({ user_id: userId, page: 1, page_size: 20 });
    setLedgerItems(response.items || []);
  }, [userId]);

  const loadShortfalls = React.useCallback(async () => {
    if (!userId) {
      setShortfallItems([]);
      return;
    }
    const response = await billingApi.listShortfalls({ user_id: userId, page: 1, page_size: 20 });
    setShortfallItems(response.items || []);
  }, [userId]);

  const loadUsage = React.useCallback(async () => {
    if (!userId) {
      setUsageItems([]);
      return;
    }
    const response = await billingApi.listUsageRecords({ user_id: userId, page: 1, page_size: 20 });
    setUsageItems(response.items || []);
  }, [userId]);

  const loadTabData = React.useCallback(async (tab: DetailTab) => {
    if (tab === "ledger") {
      await loadLedger();
      return;
    }
    if (tab === "shortfalls") {
      await loadShortfalls();
      return;
    }
    if (tab === "usage") {
      await loadUsage();
    }
  }, [loadLedger, loadShortfalls, loadUsage]);

  const refreshAllData = React.useCallback(async () => {
    await Promise.all([
      loadAccountDetail(),
      loadLedger(),
      loadShortfalls(),
      loadUsage(),
    ]);

    setLoadedTabs({
      account: true,
      ledger: true,
      shortfalls: true,
      usage: true,
    });
  }, [loadAccountDetail, loadLedger, loadShortfalls, loadUsage]);

  React.useEffect(() => {
    setLoadedTabs(INITIAL_LOADED_TABS);
    setActiveTab("account");
  }, [userId]);

  React.useEffect(() => {
    if (!userId) {
      setIsAccountLoading(false);
      setAccount(null);
      return;
    }

    setIsAccountLoading(true);
    void loadAccountDetail()
      .then(() => {
        setLoadedTabs((prev) => ({ ...prev, account: true }));
      })
      .catch((error) => {
        toast.error(error instanceof Error ? error.message : "Failed to load account detail");
      })
      .finally(() => {
        setIsAccountLoading(false);
      });
  }, [loadAccountDetail, userId]);

  React.useEffect(() => {
    if (!userId || activeTab === "account" || loadedTabs[activeTab]) {
      return;
    }

    setIsTabLoading(true);
    void loadTabData(activeTab)
      .then(() => {
        setLoadedTabs((prev) => ({ ...prev, [activeTab]: true }));
      })
      .catch((error) => {
        toast.error(error instanceof Error ? error.message : "Failed to load billing detail");
      })
      .finally(() => {
        setIsTabLoading(false);
      });
  }, [activeTab, loadTabData, loadedTabs, userId]);

  const handleRefresh = async () => {
    if (!userId) {
      return;
    }

    try {
      setIsAccountLoading(true);
      setIsTabLoading(true);
      await refreshAllData();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to refresh billing detail");
    } finally {
      setIsAccountLoading(false);
      setIsTabLoading(false);
    }
  };

  const handleAdjustBalance = async () => {
    if (!userId) {
      toast.error("Invalid user");
      return;
    }
    if (!adjustAmount || !adjustRemark.trim()) {
      toast.error("Amount and remark are required");
      return;
    }

    const amountYuan = adjustAmount.trim();
    if (!amountYuan || !Number.isFinite(Number(amountYuan)) || Number(amountYuan) === 0) {
      toast.error("Enter a valid amount");
      return;
    }

    try {
      await billingApi.adjustBalance(userId, {
        amount_yuan: amountYuan,
        remark: adjustRemark.trim(),
      });

      setAdjustAmount("");
      setAdjustRemark("");
      await refreshAllData();
      toast.success("Balance updated");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update balance");
    }
  };

  const handleReconcileShortfall = async (orderNo: string) => {
    if (!window.confirm(`Reconcile shortfall order ${orderNo}?`)) {
      return;
    }

    try {
      setReconcilingOrderNo(orderNo);
      await billingApi.reconcileShortfall(orderNo, {
        remark: reconcileRemark.trim() || undefined,
      });
      setReconcileRemark("");

      await refreshAllData();
      toast.success("Shortfall reconciled");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to reconcile shortfall");
    } finally {
      setReconcilingOrderNo("");
    }
  };

  const renderAccountTab = () => {
    if (isAccountLoading && !account) {
      return (
        <div className="rounded-2xl border border-dashed border-slate-200 px-4 py-10 text-center text-sm text-slate-500">
          Loading account detail...
        </div>
      );
    }

    if (!account) {
      return (
        <div className="rounded-2xl border border-dashed border-slate-200 px-4 py-10 text-center text-sm text-slate-500">
          User account not found.
        </div>
      );
    }

    return (
      <div data-testid="admin-billing-detail-account" className="space-y-5">
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
          <Metric label="Available" value={formatCurrencyYuan(account.available_balance_yuan)} tone="success" />
          <Metric label="Reserved" value={formatCurrencyYuan(account.reserved_balance_yuan)} tone="warning" />
          <Metric label="Total Traffic" value={formatFileSize(account.total_traffic_bytes)} tone="info" />
          <Metric label="Total Recharged" value={formatCurrencyYuan(account.total_recharged_yuan)} />
          <Metric label="Total Spent" value={formatCurrencyYuan(account.total_spent_yuan)} tone="danger" />
        </div>

        <div className="grid gap-4 lg:grid-cols-[1.05fr_0.95fr]">
          <Card className="rounded-2xl border-slate-200/80 bg-gradient-to-br from-slate-50 to-white">
            <CardContent className="p-5">
              <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-slate-500">Selected Account</p>
              <div className="mt-3 flex flex-wrap items-center gap-2">
                <p className="text-lg font-semibold text-slate-950">
                  {account.nickname || "Unknown user"}
                </p>
                <StatusBadge
                  label={accountStatusLabel(account.status)}
                  tone={accountStatusTone(account.status)}
                />
              </div>
              <p className="mt-1 text-sm text-slate-600">{account.email || account.user_id}</p>
              <p className="mt-1 font-mono text-xs text-slate-400">User ID: {account.user_id}</p>

              <div className="mt-5 grid gap-2 sm:grid-cols-2">
                <InfoChip label="Available" value={formatCurrencyYuan(account.available_balance_yuan)} />
                <InfoChip label="Reserved" value={formatCurrencyYuan(account.reserved_balance_yuan)} />
                <InfoChip label="Total Recharged" value={formatCurrencyYuan(account.total_recharged_yuan)} />
                <InfoChip label="Total Spent" value={formatCurrencyYuan(account.total_spent_yuan)} />
              </div>
            </CardContent>
          </Card>

          <Card className="rounded-2xl border-slate-200/80 shadow-sm">
            <CardContent className="space-y-4 p-5">
              <div className="flex items-center gap-2 text-slate-900">
                <div className="rounded-lg bg-emerald-100 p-1.5 text-emerald-700">
                  <Wallet className="size-4" />
                </div>
                <div>
                  <p className="font-medium">Manual Adjustment</p>
                  <p className="text-xs text-slate-500">Apply one-off credit or debit with auditable remark.</p>
                </div>
              </div>

              <div className="space-y-3">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700">Amount (yuan)</label>
                  <Input
                    data-testid="admin-billing-adjust-amount"
                    placeholder="1.00 or -5.00"
                    value={adjustAmount}
                    onChange={(e) => setAdjustAmount(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700">Remark</label>
                  <Textarea
                    data-testid="admin-billing-adjust-remark"
                    placeholder="Why are you adjusting this account?"
                    value={adjustRemark}
                    onChange={(e) => setAdjustRemark(e.target.value)}
                  />
                </div>
                <Button data-testid="admin-billing-adjust-submit" onClick={() => void handleAdjustBalance()}>
                  Submit Adjustment
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    );
  };

  const renderLedgerTab = () => {
    return (
      <div className="space-y-4">
        <div className="grid gap-3 md:grid-cols-3">
          <Metric label="Entries Loaded" value={String(ledgerItems.length)} tone="info" />
          <Metric label="Current Available" value={account ? formatCurrencyYuan(account.available_balance_yuan) : "-"} tone="success" />
          <Metric
            label="Latest Entry"
            value={ledgerItems.length ? formatDateTime(ledgerItems[0].created_at) : "-"}
            caption={ledgerItems.length ? "Most recent timeline event" : "No records yet"}
          />
        </div>

        <Card className="overflow-hidden rounded-2xl border-slate-200/80">
          <CardContent className="p-0">
            <Table data-testid="admin-billing-ledger-table">
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
                    <TableCell>{formatCurrencyYuan(item.action_amount_yuan)}</TableCell>
                    <TableCell>
                      <div className="text-xs text-slate-600">
                        <div>Avail {formatCurrencyYuan(item.balance_after_available_yuan)}</div>
                        <div>Reserved {formatCurrencyYuan(item.balance_after_reserved_yuan)}</div>
                      </div>
                    </TableCell>
                    <TableCell>{item.remark || "-"}</TableCell>
                  </TableRow>
                ))}

                {!ledgerItems.length && !isTabLoading ? (
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
      </div>
    );
  };

  const renderShortfallsTab = () => {
    return (
      <div className="space-y-4">
        <div className="grid gap-3 md:grid-cols-3">
          <Metric label="Pending Orders" value={String(shortfallItems.length)} tone="warning" />
          <Metric label="Shortfall Total" value={formatCurrencyYuan(shortfallTotalYuan)} tone="danger" />
          <Metric
            label="Latest Update"
            value={shortfallItems.length ? formatDateTime(shortfallItems[0].updated_at || shortfallItems[0].created_at) : "-"}
          />
        </div>

        <Card className="rounded-2xl border-amber-200/70 bg-amber-50/35">
          <CardContent className="space-y-2 p-4">
            <p className="text-sm font-medium text-slate-900">Reconcile Settings</p>
            <p className="text-xs text-slate-600">This remark will be attached to the next reconcile action.</p>
            <Textarea
              data-testid="admin-billing-reconcile-remark"
              placeholder="Optional note for this reconciliation"
              value={reconcileRemark}
              onChange={(e) => setReconcileRemark(e.target.value)}
            />
          </CardContent>
        </Card>

        <Card className="overflow-hidden rounded-2xl border-slate-200/80">
          <CardContent className="p-0">
            <Table data-testid="admin-billing-shortfalls-table">
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
                    <TableCell className="font-medium text-amber-700">{formatCurrencyYuan(item.shortfall_yuan)}</TableCell>
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
                        data-testid={`admin-billing-reconcile-${item.order_no}`}
                        disabled={parseCurrency(item.shortfall_yuan) <= 0 || reconcilingOrderNo === item.order_no}
                        onClick={() => void handleReconcileShortfall(item.order_no)}
                      >
                        {reconcilingOrderNo === item.order_no ? "Reconciling..." : "Reconcile"}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}

                {!shortfallItems.length && !isTabLoading ? (
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
      </div>
    );
  };

  const renderUsageTab = () => {
    return (
      <div className="space-y-4">
        <div className="grid gap-3 md:grid-cols-3">
          <Metric label="Records Loaded" value={String(usageItems.length)} tone="info" />
          <Metric label="Traffic Loaded" value={formatFileSize(usageTrafficBytes)} tone="success" />
          <Metric label="Total Amount" value={formatCurrencyYuan(usageTotalAmountYuan)} />
        </div>

        <Card className="overflow-hidden rounded-2xl border-slate-200/80">
          <CardContent className="p-0">
            <Table data-testid="admin-billing-usage-table">
              <TableHeader>
                <TableRow>
                  <TableHead>Direction</TableHead>
                  <TableHead>Traffic</TableHead>
                  <TableHead>Amount</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {usageItems.map((item) => (
                  <TableRow key={item.usage_no}>
                    <TableCell>
                      <StatusBadge label={item.direction === 1 ? "Ingress" : "Egress"} tone={directionTone(item.direction)} />
                    </TableCell>
                    <TableCell>{formatFileSize(item.traffic_bytes)}</TableCell>
                    <TableCell>{formatCurrencyYuan(item.amount_yuan)}</TableCell>
                  </TableRow>
                ))}

                {!usageItems.length && !isTabLoading ? (
                  <TableRow>
                    <TableCell colSpan={3} className="py-8 text-center text-sm text-slate-500">
                      No usage records.
                    </TableCell>
                  </TableRow>
                ) : null}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    );
  };

  return (
    <ProtectedRoute>
      <AppShell
        actions={(
          <div className="flex gap-2">
            <Link href="/billing" className={buttonVariants({ variant: "outline" })}>
              Back to Billing
            </Link>
            <Button onClick={() => void handleRefresh()} disabled={!userId}>Refresh</Button>
          </div>
        )}
      >
        <div className="space-y-4">
          {!userId ? (
            <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
              <CardContent className="py-10 text-center text-sm text-slate-500">
                Invalid user ID.
              </CardContent>
            </Card>
          ) : (
            <>
              <Card className="rounded-2xl border-slate-200/80 bg-gradient-to-r from-slate-50 via-white to-slate-50">
                <CardContent className="grid gap-3 p-5 sm:grid-cols-[1fr_auto] sm:items-center">
                  <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500">Billing Profile</p>
                    <p className="mt-1 text-lg font-semibold text-slate-950">{account?.nickname || account?.email || userId}</p>
                    <p className="text-sm text-slate-600">{account?.email || account?.user_id || userId}</p>
                  </div>

                  <div className="flex gap-2">
                    <InfoChip label="Available" value={account ? formatCurrencyYuan(account.available_balance_yuan) : "-"} />
                    <InfoChip label="Reserved" value={account ? formatCurrencyYuan(account.reserved_balance_yuan) : "-"} />
                  </div>
                </CardContent>
              </Card>

              <div className="rounded-2xl border border-slate-200/80 bg-slate-50/80 p-2">
                <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
                  <DetailTabButton
                    active={activeTab === "account"}
                    icon={UserRound}
                    label="Account Detail"
                    description="Balance and manual adjustment"
                    data-testid="admin-billing-detail-tab-account"
                    onClick={() => setActiveTab("account")}
                  />
                  <DetailTabButton
                    active={activeTab === "ledger"}
                    icon={ReceiptText}
                    label="Ledger"
                    description="Credits, holds and captures"
                    data-testid="admin-billing-detail-tab-ledger"
                    onClick={() => setActiveTab("ledger")}
                  />
                  <DetailTabButton
                    active={activeTab === "shortfalls"}
                    icon={AlertTriangle}
                    label="Shortfalls"
                    description="Pending debt and reconcile"
                    data-testid="admin-billing-detail-tab-shortfalls"
                    onClick={() => setActiveTab("shortfalls")}
                  />
                  <DetailTabButton
                    active={activeTab === "usage"}
                    icon={Gauge}
                    label="Traffic Usage"
                    description="Ingress and egress details"
                    data-testid="admin-billing-detail-tab-usage"
                    onClick={() => setActiveTab("usage")}
                  />
                </div>
              </div>

              <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
                <CardContent className="space-y-4 py-6">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-xs font-semibold uppercase tracking-[0.14em] text-slate-500">
                        {activeTab === "account" ? "Account Operations" : activeTab === "ledger" ? "Ledger Timeline" : activeTab === "shortfalls" ? "Shortfall Management" : "Traffic Usage Analytics"}
                      </p>
                      <p className="text-sm text-slate-600">
                        {activeTab === "account" ? "Review balance state and perform manual adjustment." : activeTab === "ledger" ? "Audit billing events in chronological order." : activeTab === "shortfalls" ? "Reconcile outstanding deficits with optional remarks." : "Inspect charged usage records for the current user."}
                      </p>
                    </div>
                    {isTabLoading ? (
                      <div className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-1.5 text-xs text-slate-500">
                        <ArrowDownUp className="size-3.5 animate-pulse" />
                        Syncing
                      </div>
                    ) : null}
                  </div>
                  {activeTab === "account" ? renderAccountTab() : null}
                  {activeTab === "ledger" ? renderLedgerTab() : null}
                  {activeTab === "shortfalls" ? renderShortfallsTab() : null}
                  {activeTab === "usage" ? renderUsageTab() : null}
                </CardContent>
              </Card>
            </>
          )}
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}

function Metric({
  label,
  value,
  caption,
  tone = "neutral",
}: {
  label: string;
  value: string;
  caption?: string;
  tone?: "neutral" | "success" | "warning" | "danger" | "info";
}) {
  const toneClasses = {
    neutral: "border-slate-200 bg-slate-50 text-slate-900",
    success: "border-emerald-200 bg-emerald-50 text-emerald-900",
    warning: "border-amber-200 bg-amber-50 text-amber-900",
    danger: "border-rose-200 bg-rose-50 text-rose-900",
    info: "border-sky-200 bg-sky-50 text-sky-900",
  } as const;

  return (
    <div className={cn("rounded-2xl border p-4", toneClasses[tone])}>
      <p className="text-xs uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-1 text-lg font-semibold">{value}</p>
      {caption ? <p className="mt-1 text-xs text-slate-500">{caption}</p> : null}
    </div>
  );
}

function InfoChip({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white/80 px-3 py-2">
      <p className="text-[11px] uppercase tracking-wide text-slate-500">{label}</p>
      <p className="text-sm font-semibold text-slate-900">{value}</p>
    </div>
  );
}

function DetailTabButton({
  active,
  icon: Icon,
  label,
  description,
  onClick,
  "data-testid": dataTestId,
}: {
  active: boolean;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  description: string;
  onClick: () => void;
  "data-testid"?: string;
}) {
  return (
    <button
      type="button"
      data-testid={dataTestId}
      onClick={onClick}
      className={cn(
        "flex w-full items-start gap-3 rounded-xl border px-3 py-2.5 text-left transition",
        active
          ? "border-slate-900 bg-slate-900 text-white shadow-sm"
          : "border-slate-200 bg-white text-slate-700 hover:border-slate-300 hover:bg-slate-100/70"
      )}
    >
      <div className={cn("rounded-lg p-1.5", active ? "bg-white/15" : "bg-slate-100")}>
        <Icon className={cn("size-4", active ? "text-white" : "text-slate-700")} />
      </div>
      <div>
        <p className={cn("text-sm font-medium", active ? "text-white" : "text-slate-900")}>{label}</p>
        <p className={cn("text-xs", active ? "text-slate-200" : "text-slate-500")}>{description}</p>
      </div>
    </button>
  );
}

function parseCurrency(amount: string | number) {
  const parsed = typeof amount === "number" ? amount : Number(amount);
  return Number.isFinite(parsed) ? parsed : 0;
}

function formatCurrencyYuan(amountYuan: string | number) {
  const normalized = parseCurrency(amountYuan);
  return new Intl.NumberFormat("zh-CN", {
    style: "currency",
    currency: "CNY",
    minimumFractionDigits: 2,
  }).format(normalized);
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

function directionTone(direction: number): "info" | "success" {
  return direction === 1 ? "info" : "success";
}

function totalTrafficBytes(item: BillingShortfallOrder) {
  if (item.actual_traffic_bytes > 0) {
    return item.actual_traffic_bytes;
  }
  return item.actual_ingress_bytes + item.actual_egress_bytes;
}
