"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const links = [
  { href: "/dashboard", label: "Dashboard" },
  { href: "/proxies", label: "Proxies" },
  { href: "/cookies", label: "Cookies" },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="sidebar">
      <div>
        <p className="muted">V-Asset</p>
        <h2 style={{ margin: "4px 0 0", fontSize: 28 }}>Admin</h2>
      </div>
      <nav>
        {links.map((link) => (
          <Link
            key={link.href}
            href={link.href}
            className="nav-link"
            style={{
              borderColor: pathname === link.href ? "var(--accent)" : undefined,
              color: pathname === link.href ? "var(--accent-dark)" : undefined,
            }}
          >
            {link.label}
          </Link>
        ))}
      </nav>
    </aside>
  );
}

