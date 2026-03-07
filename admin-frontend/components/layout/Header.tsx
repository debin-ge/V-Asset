"use client";

import { useRouter } from "next/navigation";

import { useAuth } from "@/hooks/use-auth";

export function Header() {
  const router = useRouter();
  const { user, logout } = useAuth();

  const handleLogout = async () => {
    await logout();
    router.push("/login");
  };

  return (
    <div className="toolbar">
      <div>
        <p className="muted" style={{ margin: 0 }}>Administrator</p>
        <strong>{user?.nickname || user?.email || "Unknown"}</strong>
      </div>
      <button className="button secondary" onClick={handleLogout}>Logout</button>
    </div>
  );
}

