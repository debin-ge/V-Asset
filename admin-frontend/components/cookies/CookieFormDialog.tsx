"use client";

import * as React from "react";

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
}: {
  onSubmit: (payload: {
    platform: string;
    name: string;
    content: string;
    expire_at?: string;
    freeze_seconds?: number;
  }) => Promise<void>;
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
    <form className="card grid" onSubmit={handleSubmit}>
      <div>
        <p className="muted" style={{ marginBottom: 6 }}>New Cookie</p>
        <h2 style={{ margin: 0 }}>Create Platform Cookie</h2>
      </div>
      <div className="split" style={{ gridTemplateColumns: "1fr 1fr" }}>
        <label className="grid" style={{ gap: 6 }}>
          <span>Platform</span>
          <select
            className="select"
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
        <label className="grid" style={{ gap: 6 }}>
          <span>Name</span>
          <input
            className="field"
            value={form.name}
            onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
          />
        </label>
      </div>
      <label className="grid" style={{ gap: 6 }}>
        <span>Cookie Content</span>
        <textarea
          className="textarea"
          rows={6}
          value={form.content}
          onChange={(e) => setForm((prev) => ({ ...prev, content: e.target.value }))}
        />
      </label>
      <div className="split" style={{ gridTemplateColumns: "1fr 1fr" }}>
        <label className="grid" style={{ gap: 6 }}>
          <span>Expire At</span>
          <input
            className="field"
            type="datetime-local"
            value={form.expire_at}
            onChange={(e) => setForm((prev) => ({ ...prev, expire_at: e.target.value }))}
          />
        </label>
        <label className="grid" style={{ gap: 6 }}>
          <span>Freeze Seconds</span>
          <input
            className="field"
            type="number"
            min="0"
            value={form.freeze_seconds}
            onChange={(e) => setForm((prev) => ({ ...prev, freeze_seconds: e.target.value }))}
          />
        </label>
      </div>
      {error ? <p style={{ color: "#a12c1d", margin: 0 }}>{error}</p> : null}
      <div className="inline-actions">
        <button className="button" type="submit" disabled={submitting}>
          {submitting ? "Creating..." : "Create Cookie"}
        </button>
      </div>
    </form>
  );
}

