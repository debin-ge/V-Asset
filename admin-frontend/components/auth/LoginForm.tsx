"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { useAuth } from "@/hooks/use-auth";

export function LoginForm() {
  const router = useRouter();
  const { login } = useAuth();
  const [email, setEmail] = React.useState("");
  const [password, setPassword] = React.useState("");
  const [error, setError] = React.useState("");
  const [submitting, setSubmitting] = React.useState(false);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      await login(email, password);
      router.push("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form className="card login-card grid" onSubmit={handleSubmit}>
      <div>
        <p className="muted" style={{ marginBottom: 6 }}>V-Asset Admin</p>
        <h1 style={{ margin: 0, fontSize: 34 }}>管理员登录</h1>
      </div>
      <label className="grid" style={{ gap: 6 }}>
        <span>邮箱</span>
        <input className="field" value={email} onChange={(e) => setEmail(e.target.value)} />
      </label>
      <label className="grid" style={{ gap: 6 }}>
        <span>密码</span>
        <input className="field" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
      </label>
      {error ? <p style={{ color: "#a12c1d", margin: 0 }}>{error}</p> : null}
      <button className="button" type="submit" disabled={submitting}>
        {submitting ? "登录中..." : "登录"}
      </button>
    </form>
  );
}

