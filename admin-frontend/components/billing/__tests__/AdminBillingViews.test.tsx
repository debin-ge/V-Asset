import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

import BillingPage from "@/app/billing/page";
import { billingApi } from "@/lib/api/billing";

vi.mock("@/hooks/use-auth", () => ({
  useAuth: () => ({ user: { user_id: "admin" }, isLoading: false }),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), prefetch: vi.fn() }),
  usePathname: () => "/admin-console/billing",
}));

vi.mock("@/components/auth/ProtectedRoute", () => ({
  ProtectedRoute: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/components/billing/WelcomeCreditSettings", () => ({
  WelcomeCreditSettings: () => <div data-testid="welcome-credit-mock">WelcomeCreditSettings</div>,
}));

vi.mock("@/lib/api/billing", () => ({
  billingApi: {
    listAccounts: vi.fn(),
    getPricing: vi.fn(),
    updatePricing: vi.fn(),
  },
}));

function buildAccount(userId: string, nickname = "Test User", email = "test@example.com") {
  return {
    user_id: userId,
    email,
    nickname,
    available_balance_fen: "1000",
    reserved_balance_fen: "100",
    total_recharged_fen: "5000",
    total_spent_fen: "4000",
    total_traffic_bytes: 0,
    status: 1,
    version: 1,
    updated_at: new Date().toISOString(),
  };
}

function buildPricing() {
  return {
    version: 1,
    ingress_price_fen_per_gib: "100",
    egress_price_fen_per_gib: "200",
    enabled: true,
    remark: "",
    updated_by_user_id: "admin",
    effective_at: new Date().toISOString(),
    created_at: new Date().toISOString(),
  };
}

function buildPaged(items: unknown[], page = 1, pageSize = 20, total = items.length) {
  return {
    total,
    page,
    page_size: pageSize,
    items,
  };
}

describe("AdminBillingViews", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders billing home with accounts table, pricing and welcome credit", async () => {
    vi.mocked(billingApi.listAccounts).mockResolvedValue(buildPaged([buildAccount("user_1")], 1, 20, 1));
    vi.mocked(billingApi.getPricing).mockResolvedValue(buildPricing());

    render(<BillingPage />);

    expect(await screen.findByTestId("admin-billing-accounts-table")).toBeInTheDocument();
    expect(screen.getByText("Pricing")).toBeInTheDocument();
    expect(screen.getByTestId("welcome-credit-mock")).toBeInTheDocument();

    await waitFor(() => {
      expect(billingApi.listAccounts).toHaveBeenCalledWith({ query: "", page: 1, page_size: 20 });
      expect(billingApi.getPricing).toHaveBeenCalled();
    });

    expect(screen.queryByTestId("admin-billing-detail-tab-account")).not.toBeInTheDocument();
    expect(screen.queryByTestId("admin-billing-ledger-table")).not.toBeInTheDocument();
  });

  it("supports pagination and page size switching", async () => {
    vi.mocked(billingApi.listAccounts)
      .mockResolvedValueOnce(buildPaged([buildAccount("user_1")], 1, 20, 45))
      .mockResolvedValueOnce(buildPaged([buildAccount("user_2")], 2, 20, 45))
      .mockResolvedValueOnce(buildPaged([buildAccount("user_3")], 1, 50, 45));
    vi.mocked(billingApi.getPricing).mockResolvedValue(buildPricing());

    render(<BillingPage />);

    expect(await screen.findByTestId("admin-billing-pagination-info")).toBeInTheDocument();

    await userEvent.click(screen.getByTestId("admin-billing-page-next"));

    await waitFor(() => {
      expect(billingApi.listAccounts).toHaveBeenCalledWith({ query: "", page: 2, page_size: 20 });
    });

    await userEvent.selectOptions(screen.getByTestId("admin-billing-page-size"), "50");

    await waitFor(() => {
      expect(billingApi.listAccounts).toHaveBeenCalledWith({ query: "", page: 1, page_size: 50 });
    });

    expect(screen.getByTestId("admin-billing-pagination-info")).toHaveTextContent("Page 1 / 1");
  });

  it("supports search and provides detail entry link", async () => {
    vi.mocked(billingApi.listAccounts)
      .mockResolvedValueOnce(buildPaged([buildAccount("user_1", "Alice", "alice@example.com")], 1, 20, 2))
      .mockResolvedValueOnce(buildPaged([buildAccount("user_2", "Bob", "bob@example.com")], 1, 20, 1))
      .mockResolvedValueOnce(buildPaged([buildAccount("user_1", "Alice", "alice@example.com")], 1, 20, 2));
    vi.mocked(billingApi.getPricing).mockResolvedValue(buildPricing());

    render(<BillingPage />);

    expect(await screen.findByTestId("admin-billing-account-search-input")).toBeInTheDocument();

    await userEvent.clear(screen.getByTestId("admin-billing-account-search-input"));
    await userEvent.type(screen.getByTestId("admin-billing-account-search-input"), "bob");
    await userEvent.click(screen.getByTestId("admin-billing-account-search"));

    await waitFor(() => {
      expect(billingApi.listAccounts).toHaveBeenCalledWith({ query: "bob", page: 1, page_size: 20 });
    });

    const viewLink = screen.getByRole("link", { name: "View" });
    expect(viewLink).toHaveAttribute("href", "/billing/accounts/user_2");

    await userEvent.click(screen.getByTestId("admin-billing-account-search-reset"));

    await waitFor(() => {
      expect(billingApi.listAccounts).toHaveBeenCalledWith({ query: "", page: 1, page_size: 20 });
    });
  });
});
