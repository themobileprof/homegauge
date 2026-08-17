"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";
import { CountrySwitcher } from "@/lib/country";
import { homeForRole, roleLabel, useAuth, type Role } from "@/lib/auth";

type NavItem = { href: string; label: string };

function itemsForRole(role?: Role | null): NavItem[] {
  if (role === "ADMIN") {
    return [
      { href: "/admin", label: "Overview" },
      { href: "/admin/users", label: "Users" },
      { href: "/mortgages", label: "Products" },
      { href: "/advisor", label: "Cases" },
      { href: "/calculator", label: "Calculator" },
    ];
  }
  if (role === "ADVISOR") {
    return [
      { href: "/advisor", label: "Case queue" },
      { href: "/mortgages", label: "Products" },
      { href: "/learn", label: "Guides" },
    ];
  }
  if (role === "LENDER_USER") {
    return [
      { href: "/lender", label: "Pipeline" },
      { href: "/mortgages", label: "Your products" },
      { href: "/learn", label: "Guides" },
    ];
  }
  if (role === "CUSTOMER") {
    return [
      { href: "/app", label: "Dashboard" },
      { href: "/app/assessment", label: "Eligibility" },
      { href: "/app/documents", label: "Documents" },
      { href: "/mortgages", label: "Options" },
      { href: "/calculator", label: "Calculator" },
      { href: "/learn", label: "Learn" },
    ];
  }
  return [
    { href: "/#how", label: "How it works" },
    { href: "/calculator", label: "Calculator" },
    { href: "/mortgages", label: "Mortgage options" },
    { href: "/learn", label: "Learn" },
  ];
}

function workspaceFromPath(pathname: string, role?: Role | null): Role | "PUBLIC" {
  if (pathname.startsWith("/admin")) return "ADMIN";
  if (pathname.startsWith("/advisor")) return "ADVISOR";
  if (pathname.startsWith("/lender")) return "LENDER_USER";
  if (pathname.startsWith("/app")) return "CUSTOMER";
  return role || "PUBLIC";
}

const workspaceTheme: Record<string, { bar: string; badge: string; label: string }> = {
  PUBLIC: { bar: "border-[color:var(--line)] bg-white/70", badge: "", label: "" },
  CUSTOMER: { bar: "border-leaf/20 bg-[#f4faf6]", badge: "bg-leaf text-white", label: "Homebuyer workspace" },
  ADVISOR: { bar: "border-gold/40 bg-[#f7f4ea]", badge: "bg-[#8a6d28] text-white", label: "Advisor workspace" },
  ADMIN: { bar: "border-ink/20 bg-[#0c1f17] text-paper", badge: "bg-gold text-ink", label: "Admin console" },
  LENDER_USER: { bar: "border-[#1f4d6b]/25 bg-[#eef5f8]", badge: "bg-[#1f4d6b] text-white", label: "Lender portal" },
};

export function SiteHeader() {
  const pathname = usePathname() || "/";
  const router = useRouter();
  const { user, loading, logout } = useAuth();
  const [open, setOpen] = useState(false);
  const workspace = workspaceFromPath(pathname, user?.role);
  const theme = workspaceTheme[workspace] || workspaceTheme.PUBLIC;
  const nav = itemsForRole(user?.role);
  const isAdminChrome = workspace === "ADMIN";

  async function onSignOut() {
    await logout();
    setOpen(false);
    router.push("/");
  }

  return (
    <header className={`sticky top-0 z-40 border-b backdrop-blur ${theme.bar}`}>
      <div className="mx-auto flex w-full max-w-6xl items-center justify-between gap-3 px-5 py-3 md:px-8">
        <div className="flex min-w-0 items-center gap-3">
          <Link href={user ? homeForRole(user.role) : "/"} className="shrink-0 font-[family-name:var(--font-display)] text-xl font-semibold tracking-tight">
            Home<span className={isAdminChrome ? "text-gold" : "text-leaf"}>Gauge</span>
          </Link>
          {theme.label && (
            <span className={`hidden rounded-full px-2.5 py-0.5 text-[11px] font-semibold uppercase tracking-wide sm:inline ${theme.badge}`}>
              {theme.label}
            </span>
          )}
        </div>

        <nav className={`hidden items-center gap-4 text-sm font-medium md:flex xl:gap-6 ${isAdminChrome ? "text-paper/80" : "text-ink-soft"}`}>
          {nav.map((item) => {
            const active = item.href !== "/" && (pathname === item.href || pathname.startsWith(item.href + "/"));
            return (
              <Link key={item.href} href={item.href} className={active ? (isAdminChrome ? "text-gold" : "text-leaf") : "hover:opacity-80"}>
                {item.label}
              </Link>
            );
          })}
        </nav>

        <div className="flex items-center gap-2 sm:gap-3">
          <div className="hidden sm:block">
            <CountrySwitcher />
          </div>
          {!loading && user ? (
            <>
              <span className={`hidden max-w-[14rem] truncate text-xs lg:inline ${isAdminChrome ? "text-paper/70" : "text-muted"}`}>
                {user.email}
                <span className="ml-1 font-semibold">{roleLabel(user.role)}</span>
              </span>
              <button
                type="button"
                onClick={onSignOut}
                className={`rounded-md px-3 py-1.5 text-sm font-semibold ${
                  isAdminChrome ? "border border-paper/25 text-paper hover:bg-white/10" : "border border-[color:var(--line)] hover:bg-white"
                }`}
              >
                Sign out
              </button>
            </>
          ) : (
            !loading && (
              <>
                <Link href="/login" className={`text-sm font-semibold ${isAdminChrome ? "text-paper/80" : "text-ink-soft"}`}>
                  Sign in
                </Link>
                <Link href="/register" className="rounded-md bg-leaf px-3 py-2 text-sm font-semibold text-white hover:bg-leaf-deep">
                  Get started
                </Link>
              </>
            )
          )}
          <button
            type="button"
            className={`rounded-md px-2 py-1.5 text-sm font-semibold md:hidden ${isAdminChrome ? "border border-paper/25" : "border border-[color:var(--line)]"}`}
            onClick={() => setOpen((v) => !v)}
            aria-expanded={open}
            aria-label="Open menu"
          >
            Menu
          </button>
        </div>
      </div>

      {open && (
        <div className={`border-t px-5 py-4 md:hidden ${isAdminChrome ? "border-paper/15" : "border-[color:var(--line)]"}`}>
          {user && (
            <p className={`mb-3 text-xs ${isAdminChrome ? "text-paper/70" : "text-muted"}`}>
              {user.email} · {roleLabel(user.role)}
            </p>
          )}
          <nav className="flex flex-col gap-3 text-sm font-medium">
            {nav.map((item) => (
              <Link key={item.href} href={item.href} onClick={() => setOpen(false)}>
                {item.label}
              </Link>
            ))}
          </nav>
          <div className="mt-4 flex flex-wrap items-center gap-3">
            <CountrySwitcher />
            {user ? (
              <button type="button" onClick={onSignOut} className="text-sm font-semibold underline">
                Sign out
              </button>
            ) : (
              <Link href="/login" className="text-sm font-semibold" onClick={() => setOpen(false)}>
                Sign in
              </Link>
            )}
          </div>
        </div>
      )}
    </header>
  );
}

export function SiteShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen">
      <SiteHeader />
      {children}
    </div>
  );
}
