"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { useAuth } from "@/hooks/use-auth";

export default function HomePage() {
  const router = useRouter();
  const { user, isLoading } = useAuth();

  React.useEffect(() => {
    if (isLoading) {
      return;
    }

    router.replace(user ? "/dashboard" : "/login");
  }, [isLoading, router, user]);

  return <main className="login-shell">Loading...</main>;
}
