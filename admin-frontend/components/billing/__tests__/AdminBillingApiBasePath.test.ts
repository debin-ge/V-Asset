import { describe, it, expect, vi, beforeEach } from "vitest";

const { mockApiClient } = vi.hoisted(() => ({
  mockApiClient: {
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
  },
}));

vi.mock("@/lib/api-client", () => ({
  default: mockApiClient,
}));

import { authApi } from "@/lib/api/auth";
import { billingApi } from "@/lib/api/billing";
import { buildAdminApiPath } from "@/lib/admin-api-path";

describe("Admin billing basePath-safe API paths", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApiClient.get.mockResolvedValue({ data: {} });
    mockApiClient.post.mockResolvedValue({ data: {} });
    mockApiClient.put.mockResolvedValue({ data: {} });
    mockApiClient.patch.mockResolvedValue({ data: {} });
    mockApiClient.delete.mockResolvedValue({ data: {} });
  });

  it("prefixes admin API paths with the admin basePath once", () => {
    expect(buildAdminApiPath("/api/v1/admin/auth/me")).toBe("/admin-console/api/v1/admin/auth/me");
    expect(buildAdminApiPath("/admin-console/api/v1/admin/auth/me")).toBe("/admin-console/api/v1/admin/auth/me");
    expect(buildAdminApiPath("/api/v1/health")).toBe("/api/v1/health");
  });

  it("routes auth bootstrap and login through the basePath proxy", async () => {
    await authApi.me();
    await authApi.login("admin@example.com", "password");

    expect(mockApiClient.get).toHaveBeenCalledWith("/admin-console/api/v1/admin/auth/me");
    expect(mockApiClient.post).toHaveBeenCalledWith("/admin-console/api/v1/admin/auth/login", {
      email: "admin@example.com",
      password: "password",
    });
  });

  it("routes billing requests through the basePath proxy", async () => {
    await billingApi.listAccounts({ query: "alice", page: 1, page_size: 20 });
    await billingApi.adjustBalance("user_123", { amount_yuan: "100", remark: "manual" });

    expect(mockApiClient.get).toHaveBeenCalledWith("/admin-console/api/v1/admin/billing/accounts", {
      params: { query: "alice", page: 1, page_size: 20 },
    });
    expect(mockApiClient.post).toHaveBeenCalledWith(
      "/admin-console/api/v1/admin/billing/accounts/user_123/adjustments",
      { amount_yuan: "100", remark: "manual" }
    );
  });
});
