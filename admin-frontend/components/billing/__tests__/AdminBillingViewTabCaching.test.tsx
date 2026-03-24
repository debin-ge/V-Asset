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

vi.mock("@/lib/api/billing", () => ({
  billingApi: {
    listAccounts: vi.fn(),
    getPricing: vi.fn(),
    updatePricing: vi.fn(),
    getWelcomeCreditSettings: vi.fn(),
    updateWelcomeCreditSettings: vi.fn(),
  },
}));

describe("AdminBillingViewTabCaching", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("does not re-fetch welcome settings when paging and searching accounts", async () => {
    vi.mocked(billingApi.listAccounts).mockImplementation(async (params?: { page?: number; page_size?: number }) => ({
      total: 40,
      page: params?.page ?? 1,
      page_size: params?.page_size ?? 20,
      items: [
        {
          user_id: "user_1",
          email: "test@example.com",
          nickname: "Test User",
          available_balance_fen: "1000",
          reserved_balance_fen: "0",
          total_recharged_fen: "1000",
          total_spent_fen: "0",
          total_traffic_bytes: 0,
          status: 1,
          version: 1,
          updated_at: new Date().toISOString(),
        },
      ],
    }));

    vi.mocked(billingApi.getPricing).mockResolvedValue({
      version: 1,
      ingress_price_fen_per_gib: "100",
      egress_price_fen_per_gib: "200",
      enabled: true,
      remark: "",
      updated_by_user_id: "admin",
      effective_at: new Date().toISOString(),
      created_at: new Date().toISOString(),
    });

    vi.mocked(billingApi.getWelcomeCreditSettings).mockResolvedValue({
      enabled: true,
      amount_yuan: "1.00",
    });

    vi.mocked(billingApi.updateWelcomeCreditSettings).mockResolvedValue({
      enabled: true,
      amount_yuan: "1.00",
    });

    render(<BillingPage />);

    expect(await screen.findByTestId("welcome-credit-form")).toBeInTheDocument();

    expect(billingApi.getWelcomeCreditSettings).toHaveBeenCalledTimes(1);

    await userEvent.click(screen.getByTestId("admin-billing-page-next"));

    await waitFor(() => {
      expect(billingApi.listAccounts).toHaveBeenCalledWith(expect.objectContaining({ page: 2 }));
    });

    await userEvent.type(screen.getByTestId("admin-billing-account-search-input"), "bob");
    await userEvent.click(screen.getByTestId("admin-billing-account-search"));

    await waitFor(() => {
      expect(billingApi.listAccounts).toHaveBeenCalledWith(expect.objectContaining({ query: "bob", page: 1 }));
    });

    await userEvent.selectOptions(screen.getByTestId("admin-billing-page-size"), "50");

    await waitFor(() => {
      expect(billingApi.listAccounts).toHaveBeenCalledWith(expect.objectContaining({ page: 1, page_size: 50 }));
    });

    expect(billingApi.getWelcomeCreditSettings).toHaveBeenCalledTimes(1);
  });
});
