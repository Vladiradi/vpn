"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuthStore } from "@/lib/store";
import { Button } from "@/components/ui/button";

const links = [
  { href: "/users", label: "Пользователи" },
  { href: "/subscriptions", label: "Подписки" },
  { href: "/nodes", label: "Ноды" },
  { href: "/offerings", label: "Тарифы" },
];

export function Nav() {
  const pathname = usePathname();
  const router = useRouter();
  const { user, clearAuth } = useAuthStore();

  return (
    <header className="border-b bg-white dark:bg-zinc-900">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3">
        <nav className="flex items-center gap-6">
          <span className="font-semibold">VPN Admin</span>
          {links.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className={`text-sm ${
                pathname === link.href ? "font-medium text-foreground" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {link.label}
            </Link>
          ))}
        </nav>
        <div className="flex items-center gap-4">
          <span className="text-sm text-muted-foreground hidden sm:inline">{user?.email}</span>
          <Button variant="outline" size="sm" onClick={() => { clearAuth(); router.push("/login"); }}>
            Выйти
          </Button>
        </div>
      </div>
    </header>
  );
}
