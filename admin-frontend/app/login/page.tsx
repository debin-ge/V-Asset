"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { LoginForm } from "@/components/auth/LoginForm";
import { useAuth } from "@/hooks/use-auth";

export default function LoginPage() {
  const router = useRouter();
  const { user, isLoading } = useAuth();

  React.useEffect(() => {
    if (!isLoading && user) {
      router.replace("/dashboard");
    }
  }, [isLoading, router, user]);

  if (isLoading) {
    return <main className="flex min-h-screen items-center justify-center px-4 text-sm text-muted-foreground">Loading...</main>;
  }

  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden px-4 py-10">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(21,94,239,0.18),transparent_22%),radial-gradient(circle_at_80%_20%,rgba(14,165,233,0.14),transparent_20%),linear-gradient(180deg,#f8fbff_0%,#eef4fb_100%)]" />
      <div className="relative grid w-full max-w-6xl gap-6 lg:grid-cols-[1.1fr_0.9fr]">
        <div className="hidden rounded-[36px] border border-border/60 bg-slate-950 p-8 text-slate-100 shadow-2xl shadow-slate-950/10 lg:flex lg:flex-col lg:justify-between">
          <div className="space-y-4">
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-200/80">V-Asset Control Plane</p>
            <h1 className="max-w-lg text-5xl font-semibold leading-tight tracking-tight">
              把代理资源、Cookies 和平台请求统一放进一个可操作的控制台。
            </h1>
            <p className="max-w-xl text-base leading-7 text-slate-300">
              新版后台强调密度、层级和可读性，让运维指标和资产管理在一个视图里更容易判断。
            </p>
          </div>
          <div className="grid gap-3 sm:grid-cols-3">
            <InfoTile label="Proxy Pools" value="Manual + Dynamic" />
            <InfoTile label="Cookie Assets" value="Lifecycle aware" />
            <InfoTile label="Ops Metrics" value="Realtime overview" />
          </div>
        </div>
        <div className="flex items-center justify-center">
          <LoginForm />
        </div>
      </div>
    </main>
  );
}

function InfoTile({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[24px] border border-white/10 bg-white/5 p-4">
      <p className="text-xs uppercase tracking-[0.16em] text-slate-400">{label}</p>
      <p className="mt-2 text-sm font-medium text-white">{value}</p>
    </div>
  );
}
