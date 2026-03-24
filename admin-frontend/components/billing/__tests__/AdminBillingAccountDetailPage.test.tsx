import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

import BillingAccountDetailPage from "@/app/billing/accounts/[userId]/page";
import { billingApi } from "@/lib/api/billing";

const { toastMock } = vi.hoisted(() => ({
  toastMock: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock("sonner", () => ({
  toast: toastMock,
}));

vi.mock("@/hooks/use-auth", () => ({
  useAuth: () => ({ user: { user_id: "admin" }, isLoading: false }),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), prefetch: vi.fn() }),
  usePathname: () => "/admin-console/billing/accounts/user_1",
  useParams: () => ({ userId: "user_1" }),
}));

vi.mock("@/components/auth/ProtectedRoute", () => ({
  ProtectedRoute: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/lib/api/billing", () => ({
  billingApi: {
    getAccountDetail: vi.fn(),
    listLedger: vi.fn(),
    listShortfalls: vi.fn(),
    listUsageRecords: vi.fn(),
    adjustBalance: vi.fn(),
    reconcileShortfall: vi.fn(),
  },
}));

function buildAccount(userId = "user_1") {
  return {
    user_id: userId,
    email: "test@example.com",
    nickname: "Test User",
    available_balance_fen: "1000",
    reserved_balance_fen: "0",
    total_recharged_fen: "1000",
    total_spent_fen: "0",
    total_traffic_bytes: 1024,
    status: 1,
    version: 1,
    updated_at: new Date().toISOString(),
  };
}

function buildShortfall(orderNo = "order_1") {
  return {
    order_no: orderNo,
    user_id: "user_1",
    email: "test@example.com",
    nickname: "Test User",
    history_id: 1,
    task_id: "task_1",
    scene: 1,
    status: 5,
    pricing_version: 1,
    actual_ingress_bytes: 1024,
    actual_egress_bytes: 0,
    actual_traffic_bytes: 1024,
    held_amount_fen: "0",
    captured_amount_fen: "0",
    released_amount_fen: "0",
    shortfall_fen: "500",
    remark: "awaiting",
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };
}

describe("AdminBillingAccountDetailPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();

    vi.mocked(billingApi.getAccountDetail).mockResolvedValue(buildAccount());
    vi.mocked(billingApi.listLedger).mockResolvedValue({ total: 0, page: 1, page_size: 20, items: [] });
    vi.mocked(billingApi.listShortfalls).mockResolvedValue({ total: 0, page: 1, page_size: 20, items: [] });
    vi.mocked(billingApi.listUsageRecords).mockResolvedValue({ total: 0, page: 1, page_size: 20, items: [] });
    vi.mocked(billingApi.adjustBalance).mockResolvedValue({ success: true, entry_no: "entry_1", account: buildAccount() });
    vi.mocked(billingApi.reconcileShortfall).mockResolvedValue({ success: true, entry_no: "entry_2" });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads account detail by default", async () => {
    render(<BillingAccountDetailPage />);

    expect(await screen.findByTestId("admin-billing-detail-tab-account")).toBeInTheDocument();

    await waitFor(() => {
      expect(billingApi.getAccountDetail).toHaveBeenCalledWith("user_1");
    });

    expect(screen.getByTestId("admin-billing-detail-account")).toBeInTheDocument();
    expect(billingApi.listLedger).not.toHaveBeenCalled();
    expect(billingApi.listShortfalls).not.toHaveBeenCalled();
    expect(billingApi.listUsageRecords).not.toHaveBeenCalled();
  });

  it("loads user-scoped ledger, shortfalls and usage on tab switches", async () => {
    render(<BillingAccountDetailPage />);

    expect(await screen.findByTestId("admin-billing-detail-tab-ledger")).toBeInTheDocument();

    await userEvent.click(screen.getByTestId("admin-billing-detail-tab-ledger"));
    await waitFor(() => {
      expect(billingApi.listLedger).toHaveBeenCalledWith({ user_id: "user_1", page: 1, page_size: 20 });
    });

    await userEvent.click(screen.getByTestId("admin-billing-detail-tab-shortfalls"));
    await waitFor(() => {
      expect(billingApi.listShortfalls).toHaveBeenCalledWith({ user_id: "user_1", page: 1, page_size: 20 });
    });

    await userEvent.click(screen.getByTestId("admin-billing-detail-tab-usage"));
    await waitFor(() => {
      expect(billingApi.listUsageRecords).toHaveBeenCalledWith({ user_id: "user_1", page: 1, page_size: 20 });
    });
  });

  it("adjusts balance and refreshes all detail datasets", async () => {
    render(<BillingAccountDetailPage />);

    expect(await screen.findByTestId("admin-billing-adjust-amount")).toBeInTheDocument();

    await userEvent.type(screen.getByTestId("admin-billing-adjust-amount"), "100");
    await userEvent.type(screen.getByTestId("admin-billing-adjust-remark"), "manual adjustment");
    await userEvent.click(screen.getByTestId("admin-billing-adjust-submit"));

    await waitFor(() => {
      expect(billingApi.adjustBalance).toHaveBeenCalledWith("user_1", {
        amount_fen: "100",
        remark: "manual adjustment",
      });
    });

    await waitFor(() => {
      expect(billingApi.getAccountDetail).toHaveBeenCalledTimes(2);
      expect(billingApi.listLedger).toHaveBeenCalledWith({ user_id: "user_1", page: 1, page_size: 20 });
      expect(billingApi.listShortfalls).toHaveBeenCalledWith({ user_id: "user_1", page: 1, page_size: 20 });
      expect(billingApi.listUsageRecords).toHaveBeenCalledWith({ user_id: "user_1", page: 1, page_size: 20 });
    });
  });

  it("reconciles shortfall and refreshes all detail datasets", async () => {
    const confirmMock = vi.fn().mockReturnValue(true);
    vi.stubGlobal("confirm", confirmMock);

    vi.mocked(billingApi.listShortfalls)
      .mockResolvedValueOnce({ total: 1, page: 1, page_size: 20, items: [buildShortfall("order_1")] })
      .mockResolvedValueOnce({ total: 1, page: 1, page_size: 20, items: [buildShortfall("order_1")] });

    render(<BillingAccountDetailPage />);

    expect(await screen.findByTestId("admin-billing-detail-tab-shortfalls")).toBeInTheDocument();

    await userEvent.click(screen.getByTestId("admin-billing-detail-tab-shortfalls"));

    await waitFor(() => {
      expect(billingApi.listShortfalls).toHaveBeenCalledWith({ user_id: "user_1", page: 1, page_size: 20 });
    });

    await userEvent.type(screen.getByTestId("admin-billing-reconcile-remark"), "reconcile now");
    await userEvent.click(screen.getByTestId("admin-billing-reconcile-order_1"));

    await waitFor(() => {
      expect(confirmMock).toHaveBeenCalledWith("Reconcile shortfall order order_1?");
      expect(billingApi.reconcileShortfall).toHaveBeenCalledWith("order_1", { remark: "reconcile now" });
    });

    await waitFor(() => {
      expect(billingApi.getAccountDetail).toHaveBeenCalledTimes(2);
      expect(billingApi.listLedger).toHaveBeenCalledWith({ user_id: "user_1", page: 1, page_size: 20 });
      expect(billingApi.listShortfalls).toHaveBeenCalledTimes(2);
      expect(billingApi.listUsageRecords).toHaveBeenCalledWith({ user_id: "user_1", page: 1, page_size: 20 });
    });
  });

  it("keeps current detail visible and shows toast when adjustment fails", async () => {
    vi.mocked(billingApi.adjustBalance).mockRejectedValue(new Error("adjust failed"));

    render(<BillingAccountDetailPage />);

    expect(await screen.findByTestId("admin-billing-detail-account")).toBeInTheDocument();
    expect(screen.getAllByText("Test User").length).toBeGreaterThan(0);

    await userEvent.type(screen.getByTestId("admin-billing-adjust-amount"), "100");
    await userEvent.type(screen.getByTestId("admin-billing-adjust-remark"), "manual adjustment");
    await userEvent.click(screen.getByTestId("admin-billing-adjust-submit"));

    await waitFor(() => {
      expect(toastMock.error).toHaveBeenCalledWith("adjust failed");
    });

    expect(billingApi.getAccountDetail).toHaveBeenCalledTimes(1);
    expect(billingApi.listLedger).not.toHaveBeenCalled();
    expect(screen.getAllByText("Test User").length).toBeGreaterThan(0);
  });
});
