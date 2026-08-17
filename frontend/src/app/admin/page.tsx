"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";

type Overview = {
  users_total: number;
  users_by_role: Record<string, number>;
  active_products: number;
  active_lenders: number;
  open_cases: number;
};

export default function AdminHomePage() {
  const [data, setData] = useState<Overview | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<Overview>("/api/v1/admin/overview")
      .then(setData)
      .catch((e) => setError(e.message));
  }, []);

  return (
    <div className="mx-auto max-w-6xl px-5 py-10 md:px-8">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-gold">Admin console</p>
      <h1 className="mt-2 font-[family-name:var(--font-display)] text-4xl font-semibold">Platform overview</h1>
      <p className="mt-3 max-w-2xl text-muted">
        Operate markets, products, and staff from here. This is not the homebuyer journey and not the advisor case queue.
      </p>
      {error && <p className="mt-6 text-sm text-[color:var(--danger)]">{error}</p>}
      <div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Stat label="People on the platform" value={data?.users_total} href="/admin/users" />
        <Stat label="Active products" value={data?.active_products} href="/admin/products" />
        <Stat label="Active lenders" value={data?.active_lenders} href="/admin/products" />
        <Stat label="Open cases" value={data?.open_cases} href="/advisor" />
      </div>
      <section className="mt-8 max-w-xl rounded-xl border border-[color:var(--line)] bg-white/80 p-6">
        <div className="flex items-start justify-between gap-3">
          <h2 className="text-lg font-semibold">Users by role</h2>
          <Link href="/admin/users" className="text-sm font-semibold text-leaf hover:underline">
            Manage
          </Link>
        </div>
        <ul className="mt-4 space-y-2 text-sm">
          {Object.entries(data?.users_by_role || {}).map(([role, n]) => (
            <li key={role} className="flex justify-between border-b border-[color:var(--line)] pb-2">
              <span>{role}</span>
              <strong>{n}</strong>
            </li>
          ))}
          {!data && <li className="text-muted">Loading…</li>}
        </ul>
      </section>
    </div>
  );
}

function Stat({ label, value, href }: { label: string; value?: number; href?: string }) {
  const inner = (
    <>
      <p className="text-xs font-semibold uppercase tracking-wide text-muted">{label}</p>
      <p className="mt-2 font-[family-name:var(--font-display)] text-3xl font-semibold">{value ?? "—"}</p>
    </>
  );
  if (href) {
    return (
      <Link href={href} className="rounded-xl border border-[color:var(--line)] bg-white/80 p-5 hover:border-leaf">
        {inner}
      </Link>
    );
  }
  return <div className="rounded-xl border border-[color:var(--line)] bg-white/80 p-5">{inner}</div>;
}
