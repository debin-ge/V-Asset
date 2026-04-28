import { describe, it, expect, vi, beforeEach } from "vitest";

const { mockApiClient } = vi.hoisted(() => ({
  mockApiClient: {
    get: vi.fn(),
  },
}));

vi.mock("@/lib/api-client", () => ({
  default: mockApiClient,
}));

import { proxyApi } from "@/lib/api/proxy";

describe("proxy usage events API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApiClient.get.mockResolvedValue({
      data: {
        events: [],
        pagination: { page: 1, page_size: 20, total: 0 },
        summary: {
          success_count: 0,
          failure_count: 0,
          failure_rate: 0,
          category_counts: [],
          stage_counts: [],
          platform_counts: [],
        },
      },
    });
  });

  it("routes usage event queries through the admin basePath proxy with params", async () => {
    await proxyApi.listUsageEvents({
      proxy_id: 42,
      stage: "parse",
      success: "failed",
      error_category: "rate_limited",
      page: 2,
      page_size: 50,
    });

    expect(mockApiClient.get).toHaveBeenCalledWith("/admin-console/api/v1/admin/proxy-usage-events", {
      params: {
        proxy_id: 42,
        stage: "parse",
        success: "failed",
        error_category: "rate_limited",
        page: 2,
        page_size: 50,
      },
    });
  });
});
