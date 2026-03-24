import type { ReactNode } from "react";

import { Header } from "@/components/layout/Header";
import { Sidebar } from "@/components/layout/Sidebar";

export function AppShell({
  children,
  actions,
}: {
  children: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <div className="min-h-screen">
      <Header />
      <div className="mx-auto flex w-full max-w-7xl flex-col px-4 pb-10 pt-24 sm:px-6 lg:px-8">
        <Sidebar actions={actions} />
        <main className="mt-6 flex-1">{children}</main>
      </div>
    </div>
  );
}
