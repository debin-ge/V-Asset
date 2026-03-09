import type { ReactNode } from "react";

import { Header } from "@/components/layout/Header";
import { Sidebar } from "@/components/layout/Sidebar";

export function AppShell({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen">
      <Header />
      <div className="mx-auto flex w-full max-w-7xl flex-col px-4 pb-10 pt-24 sm:px-6 lg:px-8">
        <Sidebar />
        <main className="mt-6 flex-1">{children}</main>
      </div>
    </div>
  );
}
