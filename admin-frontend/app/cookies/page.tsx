"use client";

import * as React from "react";

import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { CookieFilterBar } from "@/components/cookies/CookieFilterBar";
import { CookieFormDialog } from "@/components/cookies/CookieFormDialog";
import { CookieTable } from "@/components/cookies/CookieTable";
import { AppShell } from "@/components/layout/AppShell";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { cookieApi } from "@/lib/api/cookie";
import type { CookieInfo } from "@/types/cookie";
import { toast } from "sonner";

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
    try {
      await cookieApi.delete(id);
      await loadCookies();
      toast.success("Cookie deleted");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to delete cookie");
    }
  };

  const handleFreeze = async (id: number) => {
    try {
      await cookieApi.freeze(id, 1800);
      await loadCookies();
      toast.success("Cookie frozen");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to freeze cookie");
    }
  };

  const handleCreate = async (payload: {
    platform: string;
    name: string;
    content: string;
    expire_at?: string;
    freeze_seconds?: number;
  }) => {
    try {
      await cookieApi.create(payload);
      setShowCreateForm(false);
      await loadCookies();
      toast.success("Cookie created");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to create cookie");
      throw error;
    }
  };

  return (
    <ProtectedRoute>
      <AppShell>
        <div className="space-y-4">
          <CookieFilterBar
            platform={platform}
            onPlatformChange={setPlatform}
            onRefresh={() => void loadCookies()}
            onCreateToggle={() => setShowCreateForm((prev) => !prev)}
            creating={showCreateForm}
          />
          <Dialog open={showCreateForm} onOpenChange={setShowCreateForm}>
            <DialogContent className="max-w-2xl">
              <CookieFormDialog onSubmit={handleCreate} onCancel={() => setShowCreateForm(false)} />
            </DialogContent>
          </Dialog>
          <CookieTable
            items={items}
            onDelete={(id) => void handleDelete(id)}
            onFreeze={(id) => void handleFreeze(id)}
          />
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}
