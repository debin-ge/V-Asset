import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

import { WelcomeCreditSettings } from "@/components/billing/WelcomeCreditSettings";
import { billingApi } from "@/lib/api/billing";

vi.mock("@/lib/api/billing", () => ({
  billingApi: {
    getWelcomeCreditSettings: vi.fn(),
    updateWelcomeCreditSettings: vi.fn(),
  },
}));

describe("WelcomeCreditSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("loads and displays the default settings (1.00 yuan)", async () => {
    const mockGet = vi.mocked(billingApi.getWelcomeCreditSettings).mockResolvedValue({
      enabled: true,
      amount_yuan: "1.00",
      currency_code: "CNY",
    });

    render(<WelcomeCreditSettings />);

    expect(screen.getByText("Loading settings...")).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByTestId("welcome-credit-form")).toBeInTheDocument();
    });

    const checkbox = screen.getByTestId("welcome-credit-enabled");
    expect(checkbox).toBeChecked();

    const input = screen.getByTestId("welcome-credit-amount");
    expect(input).toHaveValue("1.00");

    expect(mockGet).toHaveBeenCalledTimes(1);
  });

  it("updates settings and shows saved banner", async () => {
    vi.mocked(billingApi.getWelcomeCreditSettings).mockResolvedValue({
      enabled: true,
      amount_yuan: "1.00",
      currency_code: "CNY",
    });

    const mockUpdate = vi.mocked(billingApi.updateWelcomeCreditSettings).mockResolvedValue({
      enabled: true,
      amount_yuan: "1.50",
      currency_code: "CNY",
    });

    render(<WelcomeCreditSettings />);

    await waitFor(() => {
      expect(screen.getByTestId("welcome-credit-form")).toBeInTheDocument();
    });

    const input = screen.getByTestId("welcome-credit-amount");
    const saveButton = screen.getByTestId("welcome-credit-save");

    await userEvent.clear(input);
    await userEvent.type(input, "1.50");
    
    expect(input).toHaveValue("1.50");

    await userEvent.click(saveButton);

    await waitFor(() => {
      expect(mockUpdate).toHaveBeenCalledWith({
        enabled: true,
        amount_yuan: "1.50",
        currency_code: "CNY",
      });
    });

    expect(screen.getByTestId("welcome-credit-saved-banner")).toBeInTheDocument();
    expect(input).toHaveValue("1.50");
  });
});
