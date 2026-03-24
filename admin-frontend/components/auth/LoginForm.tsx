"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { LockKeyhole, ShieldCheck } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
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
    <Card className="w-full max-w-xl rounded-[32px] border-border/60 bg-card/95 shadow-2xl shadow-slate-950/10">
      <CardHeader className="gap-3 border-b border-border/60 pb-6">
        <div className="flex size-12 items-center justify-center rounded-2xl bg-primary/10 text-primary">
          <ShieldCheck className="size-5" />
        </div>
        <div className="space-y-2">
          <p className="text-xs font-semibold uppercase tracking-[0.22em] text-primary/70">YouDLP Admin</p>
          <CardTitle className="text-3xl tracking-tight">管理员登录</CardTitle>
          <CardDescription>统一管理平台流量、代理资源和会话资产。</CardDescription>
        </div>
      </CardHeader>
      <CardContent className="pt-6">
        <form className="grid gap-5" onSubmit={handleSubmit}>
          <label className="grid gap-2">
            <span className="text-sm font-medium text-foreground">邮箱</span>
            <Input value={email} onChange={(e) => setEmail(e.target.value)} placeholder="admin@youdlp.local" />
          </label>
          <label className="grid gap-2">
            <span className="text-sm font-medium text-foreground">密码</span>
            <Input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="输入管理员密码"
            />
          </label>
          {error ? <p className="text-sm text-destructive">{error}</p> : null}
          <Button size="lg" type="submit" disabled={submitting}>
            <LockKeyhole data-icon="inline-start" />
            {submitting ? "登录中..." : "登录控制台"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
