"use client";

import * as React from "react";

import { Button } from "@/components/ui/button";
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

type CookieFormState = {
  platform: string;
  name: string;
  content: string;
  expire_at: string;
  freeze_seconds: string;
};

const emptyState: CookieFormState = {
  platform: "youtube",
  name: "",
  content: "",
  expire_at: "",
  freeze_seconds: "0",
};

export function CookieFormDialog({
  onSubmit,
  onCancel,
}: {
  onSubmit: (payload: {
    platform: string;
    name: string;
    content: string;
    expire_at?: string;
    freeze_seconds?: number;
  }) => Promise<void>;
  onCancel: () => void;
}) {
  const [form, setForm] = React.useState<CookieFormState>(emptyState);
  const [submitting, setSubmitting] = React.useState(false);
  const [error, setError] = React.useState("");

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitting(true);
    setError("");

    try {
      await onSubmit({
        platform: form.platform,
        name: form.name,
        content: form.content,
        expire_at: form.expire_at ? form.expire_at.replace("T", " ") + ":00" : undefined,
        freeze_seconds: Number(form.freeze_seconds || "0"),
      });
      setForm(emptyState);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create cookie");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form className="grid gap-4" onSubmit={handleSubmit}>
      <DialogHeader>
        <DialogTitle>Create Platform Cookie</DialogTitle>
        <DialogDescription>录入平台会话内容、冻结时长与过期时间。</DialogDescription>
      </DialogHeader>
      <div className="grid gap-4 md:grid-cols-2">
        <label className="grid gap-2">
          <span className="text-sm font-medium text-foreground">Platform</span>
          <select
            className="h-8 w-full rounded-lg border border-input bg-background px-2.5 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
            value={form.platform}
            onChange={(e) => setForm((prev) => ({ ...prev, platform: e.target.value }))}
          >
            <option value="youtube">YouTube</option>
            <option value="bilibili">Bilibili</option>
            <option value="tiktok">TikTok</option>
            <option value="twitter">Twitter</option>
            <option value="instagram">Instagram</option>
          </select>
        </label>
        <label className="grid gap-2">
          <span className="text-sm font-medium text-foreground">Name</span>
          <Input
            value={form.name}
            onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
            placeholder="Account label"
          />
        </label>
      </div>
      <label className="grid gap-2">
        <span className="text-sm font-medium text-foreground">Cookie Content</span>
        <Textarea
          rows={6}
          value={form.content}
          onChange={(e) => setForm((prev) => ({ ...prev, content: e.target.value }))}
          placeholder="Paste raw cookie content"
        />
      </label>
      <div className="grid gap-4 md:grid-cols-2">
        <label className="grid gap-2">
          <span className="text-sm font-medium text-foreground">Expire At</span>
          <Input
            type="datetime-local"
            value={form.expire_at}
            onChange={(e) => setForm((prev) => ({ ...prev, expire_at: e.target.value }))}
          />
        </label>
        <label className="grid gap-2">
          <span className="text-sm font-medium text-foreground">Freeze Seconds</span>
          <Input
            type="number"
            min="0"
            value={form.freeze_seconds}
            onChange={(e) => setForm((prev) => ({ ...prev, freeze_seconds: e.target.value }))}
          />
        </label>
      </div>
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      <DialogFooter>
        <Button variant="outline" type="button" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={submitting}>
          {submitting ? "Creating..." : "Create Cookie"}
        </Button>
      </DialogFooter>
    </form>
  );
}
