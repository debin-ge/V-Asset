"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Cookie, CreditCard, Globe, LayoutDashboard } from "lucide-react";

import { cn } from "@/lib/utils";

const links = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard, note: "Overview" },
  { href: "/billing", label: "Billing", icon: CreditCard, note: "Accounts & pricing" },
  { href: "/proxies", label: "Proxies", icon: Globe, note: "Pool & policy" },
  { href: "/cookies", label: "Cookies", icon: Cookie, note: "Session assets" },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-full">
      <div className="overflow-hidden rounded-3xl border border-white/50 bg-white/70 p-2 shadow-lg shadow-blue-950/5 backdrop-blur-xl">
        <nav className="flex gap-2 overflow-x-auto">
          {links.map((link) => {
            const Icon = link.icon;
            const active = pathname === link.href;

            return (
              <Link
                key={link.href}
                href={link.href}
                className={cn(
                  "group flex min-w-[190px] items-center gap-3 rounded-2xl px-4 py-3 transition-all",
                  active
                    ? "bg-gradient-to-r from-blue-600 via-indigo-600 to-purple-600 text-white shadow-lg shadow-blue-500/20"
                    : "text-slate-600 hover:bg-white hover:text-slate-950"
                )}
              >
                <div
                  className={cn(
                    "flex size-10 items-center justify-center rounded-xl transition-colors",
                    active ? "bg-white/15" : "bg-slate-100 text-slate-500 group-hover:bg-blue-50 group-hover:text-blue-600"
                  )}
                >
                  <Icon className="size-4" />
                </div>
                <div className="min-w-0">
                  <p className="font-medium">{link.label}</p>
                  <p className={cn("text-xs transition-colors", active ? "text-white/75" : "text-slate-400 group-hover:text-slate-500")}>{link.note}</p>
                </div>
              </Link>
            );
          })}
        </nav>
      </div>
    </aside>
  );
}
