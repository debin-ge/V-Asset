"use client";

import * as React from "react";

import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { CookieFilterBar } from "@/components/cookies/CookieFilterBar";
import { CookieFormDialog } from "@/components/cookies/CookieFormDialog";
import { CookieTable } from "@/components/cookies/CookieTable";
import { Header } from "@/components/layout/Header";
import { Sidebar } from "@/components/layout/Sidebar";
import { cookieApi } from "@/lib/api/cookie";
import type { CookieInfo } from "@/types/cookie";

export default function CookiesPage() {
  const [items, setItems] = React.useState<CookieInfo[]>([]);
  const [platform, setPlatform] = React.useState("");
  const [showCreateForm, setShowCreateForm] = React.useState(false);

  const loadCookies = React.useCallback(async () => {
    const response = await cookieApi.list({
      page: 1,
      page_size: 20,
      ...(platform ? { platform } : {}),
    });
    setItems(response.items || []);
  }, [platform]);

  React.useEffect(() => {
    void loadCookies();
  }, [loadCookies]);

  const handleDelete = async (id: number) => {
    await cookieApi.delete(id);
    await loadCookies();
  };

  const handleFreeze = async (id: number) => {
    await cookieApi.freeze(id, 1800);
    await loadCookies();
  };

  const handleCreate = async (payload: {
    platform: string;
    name: string;
    content: string;
    expire_at?: string;
    freeze_seconds?: number;
  }) => {
    await cookieApi.create(payload);
    setShowCreateForm(false);
    await loadCookies();
  };

  return (
    <ProtectedRoute>
      <div className="layout shell">
        <Sidebar />
        <main className="content">
          <Header />
          <div style={{ marginBottom: 16 }}>
            <div>
              <h1 className="page-title">Cookies</h1>
              <p className="muted">管理平台 Cookies 资源。</p>
            </div>
          </div>
          <CookieFilterBar
            platform={platform}
            onPlatformChange={setPlatform}
            onRefresh={() => void loadCookies()}
            onCreateToggle={() => setShowCreateForm((prev) => !prev)}
            creating={showCreateForm}
          />
          {showCreateForm ? <CookieFormDialog onSubmit={handleCreate} /> : null}
          <CookieTable
            items={items}
            onDelete={(id) => void handleDelete(id)}
            onFreeze={(id) => void handleFreeze(id)}
          />
        </main>
      </div>
    </ProtectedRoute>
  );
}
