"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { LogOut, Shield } from "lucide-react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/hooks/use-auth";

export function Header() {
  const router = useRouter();
  const { user, logout } = useAuth();

  const handleLogout = async () => {
    await logout();
    router.push("/login");
  };

  return (
    <header className="fixed left-0 right-0 top-0 z-50 border-b border-white/40 bg-white/55 backdrop-blur-xl">
      <div className="mx-auto flex h-16 w-full max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <Link href="/dashboard" className="flex items-center gap-3">
          <div className="flex size-9 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-500 to-purple-600 text-white shadow-lg shadow-blue-500/30">
            <Shield className="size-4" />
          </div>
          <div>
            <p className="bg-gradient-to-r from-blue-600 via-indigo-600 to-purple-600 bg-clip-text text-lg font-bold text-transparent">
              YouDLP
            </p>
            <p className="text-[11px] font-medium uppercase tracking-[0.18em] text-slate-400">
              Admin Frontend
            </p>
          </div>
        </Link>
        <div className="flex items-center gap-3">
          <div className="hidden items-center gap-3 rounded-full border border-white/50 bg-white/70 px-3 py-1.5 shadow-sm md:flex">
            <Avatar size="default">
              <AvatarImage src={user?.avatar_url} alt={user?.nickname || user?.email || "Admin"} />
              <AvatarFallback>{(user?.nickname || user?.email || "A").slice(0, 1).toUpperCase()}</AvatarFallback>
            </Avatar>
            <div className="pr-1">
              <p className="text-sm font-medium text-slate-900">{user?.nickname || user?.email || "Unknown"}</p>
              <p className="text-xs text-slate-400">Administrator</p>
            </div>
          </div>
          <Button variant="outline" className="rounded-full bg-white/70" onClick={handleLogout}>
            <LogOut data-icon="inline-start" />
            Logout
          </Button>
        </div>
      </div>
    </header>
  );
}
