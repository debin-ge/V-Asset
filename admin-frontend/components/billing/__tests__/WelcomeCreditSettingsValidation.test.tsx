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

describe("WelcomeCreditSettingsValidation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("blocks invalid amount and does not call update API", async () => {
    vi.mocked(billingApi.getWelcomeCreditSettings).mockResolvedValue({
      enabled: true,
      amount_yuan: "1.00",
    });

    const mockUpdate = vi.mocked(billingApi.updateWelcomeCreditSettings);

    render(<WelcomeCreditSettings />);

    await waitFor(() => {
      expect(screen.getByTestId("welcome-credit-form")).toBeInTheDocument();
    });

    const input = screen.getByTestId("welcome-credit-amount");
    const saveButton = screen.getByTestId("welcome-credit-save");

    await userEvent.clear(input);
    await userEvent.type(input, "-10.00");
    
    await userEvent.click(saveButton);

    expect(mockUpdate).not.toHaveBeenCalled();
    expect(screen.queryByTestId("welcome-credit-saved-banner")).not.toBeInTheDocument();
  });
});
