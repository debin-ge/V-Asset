"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Cookie, CreditCard, Globe, LayoutDashboard } from "lucide-react";
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

const links = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard, note: "Overview" },
  { href: "/billing", label: "Billing", icon: CreditCard, note: "Accounts & pricing" },
  { href: "/proxies", label: "Proxies", icon: Globe, note: "Pool & policy" },
  { href: "/cookies", label: "Cookies", icon: Cookie, note: "Session assets" },
];

type SidebarNavProps = {
  compact?: boolean;
  className?: string;
};

export function SidebarNav({ compact = false, className }: SidebarNavProps) {
  const pathname = usePathname();

  return (
    <nav className={cn("flex gap-2 overflow-x-auto", className)}>
      {links.map((link) => {
        const Icon = link.icon;
        const routePath = pathname?.startsWith("/admin-console") ? `/admin-console${link.href}` : link.href;
        const active = pathname === routePath || pathname === link.href || pathname?.startsWith(`${routePath}/`) || pathname?.startsWith(`${link.href}/`);

        return (
          <Link
            key={link.href}
            href={link.href}
            className={cn(
              "group flex min-w-[190px] items-center gap-3 transition-all",
              compact ? "rounded-xl px-3 py-2.5" : "rounded-2xl px-4 py-3",
              active
                ? "bg-gradient-to-r from-blue-600 via-indigo-600 to-purple-600 text-white shadow-lg shadow-blue-500/20"
                : "text-slate-600 hover:bg-white hover:text-slate-950"
            )}
          >
            <div
              className={cn(
                "flex items-center justify-center transition-colors",
                compact ? "size-8 rounded-lg" : "size-10 rounded-xl",
                active ? "bg-white/15" : "bg-slate-100 text-slate-500 group-hover:bg-blue-50 group-hover:text-blue-600"
              )}
            >
              <Icon className={compact ? "size-3.5" : "size-4"} />
            </div>
            <div className="min-w-0">
              <p className={cn("font-medium", compact ? "text-sm" : "text-base")}>{link.label}</p>
              <p className={cn("text-xs transition-colors", active ? "text-white/75" : "text-slate-400 group-hover:text-slate-500")}>{link.note}</p>
            </div>
          </Link>
        );
      })}
    </nav>
  );
}

function SidebarFrame({
  actions,
}: {
  actions?: ReactNode;
}) {
  return (
    <aside className="w-full">
      <div className="overflow-hidden rounded-3xl border border-white/50 bg-white/70 shadow-lg shadow-blue-950/5 backdrop-blur-xl">
        <div className="border-b border-white/60 p-2">
          <SidebarNav />
        </div>
        {actions ? (
          <div className="flex flex-wrap items-center justify-end gap-2 p-3">
            {actions}
          </div>
        ) : null}
      </div>
    </aside>
  );
}

export function Sidebar({
  actions,
}: {
  actions?: ReactNode;
}) {
  return <SidebarFrame actions={actions} />;
}
