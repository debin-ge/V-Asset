"use client";

import * as React from "react";
import Link from "next/link";
import { Search } from "lucide-react";
import { toast } from "sonner";

import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { WelcomeCreditSettings } from "@/components/billing/WelcomeCreditSettings";
import { AppShell } from "@/components/layout/AppShell";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { billingApi } from "@/lib/api/billing";
import type { BillingAccount, BillingPricing } from "@/types/billing";

const PAGE_SIZE_OPTIONS = [10, 20, 50] as const;

export default function BillingPage() {
  const [accounts, setAccounts] = React.useState<BillingAccount[]>([]);
  const [queryInput, setQueryInput] = React.useState("");
  const [query, setQuery] = React.useState("");
  const [page, setPage] = React.useState(1);
  const [pageSize, setPageSize] = React.useState<number>(20);
  const [total, setTotal] = React.useState(0);

  const [pricing, setPricing] = React.useState<BillingPricing | null>(null);
  const [isLoading, setIsLoading] = React.useState(true);
  const [pricingForm, setPricingForm] = React.useState({
    ingress_price_yuan_per_gb: "0",
    egress_price_yuan_per_gb: "0",
    remark: "",
  });

  const totalPages = React.useMemo(() => {
    if (total <= 0) {
      return 1;
    }
    return Math.max(1, Math.ceil(total / pageSize));
  }, [pageSize, total]);

  const loadAccounts = React.useCallback(async (
    target: { query?: string; page?: number; pageSize?: number } = {}
  ) => {
    const targetQuery = target.query ?? query;
    const targetPage = target.page ?? page;
    const targetPageSize = target.pageSize ?? pageSize;

    const response = await billingApi.listAccounts({
      query: targetQuery,
      page: targetPage,
      page_size: targetPageSize,
    });

    setAccounts(response.items || []);
    setTotal(response.total || 0);
    setPage(response.page || targetPage);
    setPageSize(response.page_size || targetPageSize);
  }, [page, pageSize, query]);

  const loadPricing = React.useCallback(async () => {
    const pricingResponse = await billingApi.getPricing();
    setPricing(pricingResponse);
    setPricingForm({
      ingress_price_yuan_per_gb: pricingResponse.ingress_price_yuan_per_gb,
      egress_price_yuan_per_gb: pricingResponse.egress_price_yuan_per_gb,
      remark: pricingResponse.remark || "",
    });
  }, []);

  const loadPage = React.useCallback(async () => {
    setIsLoading(true);
    try {
      await Promise.all([
        loadAccounts({ query, page, pageSize }),
        loadPricing(),
      ]);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to load billing data");
    } finally {
      setIsLoading(false);
    }
  }, [loadAccounts, loadPricing, page, pageSize, query]);

  const hasInitializedRef = React.useRef(false);
  React.useEffect(() => {
    if (hasInitializedRef.current) {
      return;
    }
    hasInitializedRef.current = true;
    void loadPage();
  }, [loadPage]);

  const handleSearch = async () => {
    const nextQuery = queryInput.trim();
    setQueryInput(nextQuery);
    setQuery(nextQuery);

    try {
      setIsLoading(true);
      await loadAccounts({ query: nextQuery, page: 1 });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to search accounts");
    } finally {
      setIsLoading(false);
    }
  };

  const handleResetSearch = async () => {
    setQueryInput("");
    setQuery("");

    try {
      setIsLoading(true);
      await loadAccounts({ query: "", page: 1 });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to reset account list");
    } finally {
      setIsLoading(false);
    }
  };

  const handleChangePage = async (nextPage: number) => {
    const clampedPage = Math.min(Math.max(1, nextPage), totalPages);
    try {
      setIsLoading(true);
      await loadAccounts({ page: clampedPage });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to change page");
    } finally {
      setIsLoading(false);
    }
  };

  const handleChangePageSize = async (nextPageSize: number) => {
    try {
      setIsLoading(true);
      await loadAccounts({ page: 1, pageSize: nextPageSize });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to change page size");
    } finally {
      setIsLoading(false);
    }
  };

  const handleUpdatePricing = async () => {
    try {
      await billingApi.updatePricing({
        ingress_price_yuan_per_gb: pricingForm.ingress_price_yuan_per_gb,
        egress_price_yuan_per_gb: pricingForm.egress_price_yuan_per_gb,
        remark: pricingForm.remark,
      });
      await loadPricing();
      toast.success("Pricing updated");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update pricing");
    }
  };

  return (
    <ProtectedRoute>
      <AppShell
        actions={<Button onClick={() => void loadPage()}>Refresh</Button>}
      >
        <div className="space-y-4">
          <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
            <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
              <CardHeader>
                <CardTitle>User Accounts</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex flex-col gap-3 lg:flex-row">
                  <div className="relative flex-1">
                    <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-400" />
                    <Input
                      data-testid="admin-billing-account-search-input"
                      className="pl-9"
                      placeholder="Search by email, nickname, or user ID"
                      value={queryInput}
                      onChange={(e) => setQueryInput(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") {
                          void handleSearch();
                        }
                      }}
                    />
                  </div>
                  <Button data-testid="admin-billing-account-search" onClick={() => void handleSearch()}>Search</Button>
                  <Button data-testid="admin-billing-account-search-reset" variant="outline" onClick={() => void handleResetSearch()}>
                    Reset
                  </Button>
                </div>

                <div className="overflow-hidden rounded-2xl border border-slate-100">
                  <Table data-testid="admin-billing-accounts-table">
                    <TableHeader>
                      <TableRow>
                        <TableHead>User</TableHead>
                        <TableHead>Available</TableHead>
                        <TableHead>Reserved</TableHead>
                        <TableHead>Total Spent</TableHead>
                        <TableHead className="text-right">Action</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {accounts.map((account) => (
                        <TableRow key={account.user_id}>
                          <TableCell>
                            <UserIdentity
                              userId={account.user_id}
                              email={account.email}
                              nickname={account.nickname}
                            />
                          </TableCell>
                          <TableCell>{formatCurrencyYuan(account.available_balance_yuan)}</TableCell>
                          <TableCell>{formatCurrencyYuan(account.reserved_balance_yuan)}</TableCell>
                          <TableCell>{formatCurrencyYuan(account.total_spent_yuan)}</TableCell>
                          <TableCell className="text-right">
                            <Link
                              href={`/billing/accounts/${encodeURIComponent(account.user_id)}`}
                              className={buttonVariants({
                                size: "sm",
                                variant: "outline",
                              })}
                              data-testid={`admin-billing-view-${account.user_id}`}
                            >
                              View
                            </Link>
                          </TableCell>
                        </TableRow>
                      ))}

                      {!accounts.length && !isLoading ? (
                        <TableRow>
                          <TableCell colSpan={5} className="py-8 text-center text-sm text-slate-500">
                            No billing accounts found.
                          </TableCell>
                        </TableRow>
                      ) : null}
                    </TableBody>
                  </Table>
                </div>

                <div className="flex flex-col gap-3 rounded-2xl border border-slate-100 bg-slate-50/70 p-3 md:flex-row md:items-center md:justify-between">
                  <p data-testid="admin-billing-pagination-info" className="text-sm text-slate-600">
                    Page {page} / {totalPages} · {total} users
                  </p>

                  <div className="flex items-center gap-2">
                    <label htmlFor="admin-billing-page-size" className="text-sm text-slate-600">Per page</label>
                    <select
                      id="admin-billing-page-size"
                      data-testid="admin-billing-page-size"
                      className="h-8 rounded-lg border border-input bg-background px-2.5 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                      value={String(pageSize)}
                      onChange={(e) => void handleChangePageSize(Number(e.target.value))}
                    >
                      {PAGE_SIZE_OPTIONS.map((size) => (
                        <option key={size} value={size}>{size}</option>
                      ))}
                    </select>
                    <Button
                      size="sm"
                      variant="outline"
                      data-testid="admin-billing-page-prev"
                      disabled={isLoading || page <= 1}
                      onClick={() => void handleChangePage(page - 1)}
                    >
                      Prev
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      data-testid="admin-billing-page-next"
                      disabled={isLoading || page >= totalPages}
                      onClick={() => void handleChangePage(page + 1)}
                    >
                      Next
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>

            <div className="space-y-4">
              <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
                <CardHeader>
                  <CardTitle>Pricing</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid gap-3 md:grid-cols-2">
                    <div className="space-y-2">
                      <label className="text-sm font-medium text-slate-700">Ingress Price (yuan / GB)</label>
                      <Input
                        value={pricingForm.ingress_price_yuan_per_gb}
                        onChange={(e) => setPricingForm((prev) => ({ ...prev, ingress_price_yuan_per_gb: e.target.value }))}
                      />
                    </div>
                    <div className="space-y-2">
                      <label className="text-sm font-medium text-slate-700">Egress Price (yuan / GB)</label>
                      <Input
                        value={pricingForm.egress_price_yuan_per_gb}
                        onChange={(e) => setPricingForm((prev) => ({ ...prev, egress_price_yuan_per_gb: e.target.value }))}
                      />
                    </div>
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
                      <div>Active version #{pricing.version} · updated {formatDateTime(pricing.effective_at)}</div>
                      <div className="mt-1 text-xs text-slate-500">
                        Billing uses MB-based traffic and applies a minimum billable size of 100 MB per capture.
                      </div>
                    </div>
                  ) : null}

                  <Button onClick={() => void handleUpdatePricing()}>Update Pricing</Button>
                </CardContent>
              </Card>

              <WelcomeCreditSettings />
            </div>
          </div>
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}

function UserIdentity({ userId, email, nickname }: { userId: string; email?: string; nickname?: string }) {
  return (
    <div>
      <p className="font-medium text-slate-900">{nickname || "Unknown user"}</p>
      <p className="text-xs text-slate-500">{email || userId}</p>
    </div>
  );
}

function formatCurrencyYuan(amountYuan: string | number) {
  const normalized = parseCurrency(amountYuan);
  return new Intl.NumberFormat("zh-CN", {
    style: "currency",
    currency: "CNY",
    minimumFractionDigits: 2,
  }).format(normalized);
}

function parseCurrency(amount: string | number) {
  const parsed = typeof amount === "number" ? amount : Number(amount);
  return Number.isFinite(parsed) ? parsed : 0;
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
