"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useCountry } from "@/lib/country";

type Product = {
  id: string;
  country_code: string;
  currency_code: string;
  lender_name: string;
  name: string;
  description: string;
  interest_rate: number | null;
  interest_rate_type: string;
  max_tenor_years: number | null;
  min_equity_pct: number | null;
  min_income: number | null;
  min_loan_amount: number | null;
  max_loan_amount: number | null;
  verification_status: string;
  last_verified_at: string | null;
  mortgage_type: string;
};

export default function MortgagesPage() {
  const { country, countryCode, money } = useCountry();
  const [products, setProducts] = useState<Product[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    setSelected([]);
    api<{ products: Product[] }>(`/api/v1/mortgage-products?country=${encodeURIComponent(countryCode)}`)
      .then((d) => setProducts(d.products || []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [countryCode]);

  const compareHref = useMemo(() => {
    if (selected.length < 2) return null;
    return `/mortgages/compare?ids=${selected.join(",")}`;
  }, [selected]);

  function toggle(id: string) {
    setSelected((prev) => {
      if (prev.includes(id)) return prev.filter((x) => x !== id);
      if (prev.length >= 4) return prev;
      return [...prev, id];
    });
  }

  return (
    <div>
      <main className="mx-auto max-w-6xl px-5 py-10 pb-20">
        <h1 className="font-[family-name:var(--font-display)] text-4xl font-semibold">Mortgage options</h1>
        <p className="mt-3 max-w-2xl text-muted">
          Compare mortgage products in {country?.name || "your market"}. Every rate shows when it was last verified. HomeGauge is not a lender.
        </p>

        <div className="mt-6 flex flex-wrap items-center gap-3">
          <p className="text-sm text-muted">Select 2–4 products to compare.</p>
          {compareHref ? (
            <Link href={compareHref} className="rounded-md bg-ink px-4 py-2 text-sm font-semibold text-paper">
              Compare selected ({selected.length})
            </Link>
          ) : (
            <span className="rounded-md bg-paper-2 px-4 py-2 text-sm text-muted">Compare selected ({selected.length})</span>
          )}
        </div>

        {loading && <p className="mt-10 text-muted">Loading products…</p>}
        {error && <p className="mt-10 text-sm text-[color:var(--danger)]">{error}</p>}
        {!loading && !error && products.length === 0 && (
          <p className="mt-10 text-muted">No active products in this market yet. Switch country or check back soon.</p>
        )}

        <div className="mt-10 grid gap-5 md:grid-cols-2 lg:grid-cols-3">
          {products.map((p) => {
            const on = selected.includes(p.id);
            return (
              <article key={p.id} className={`rounded-xl border p-5 ${on ? "border-leaf bg-white" : "border-[color:var(--line)] bg-white/70"}`}>
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-xs font-semibold uppercase tracking-wide text-leaf">{p.lender_name}</p>
                    <h2 className="mt-1 text-lg font-semibold">{p.name}</h2>
                  </div>
                  <button
                    type="button"
                    onClick={() => toggle(p.id)}
                    className={`rounded-md px-3 py-1.5 text-xs font-semibold ${on ? "bg-leaf text-white" : "bg-paper-2 text-ink"}`}
                  >
                    {on ? "Selected" : "Select"}
                  </button>
                </div>
                <p className="mt-3 line-clamp-3 text-sm text-muted">{p.description}</p>
                <dl className="mt-4 space-y-2 text-sm">
                  <Row label="Interest rate" value={p.interest_rate != null ? `${p.interest_rate}% ${p.interest_rate_type}` : "—"} />
                  <Row label="Max years to repay" value={p.max_tenor_years ? `${p.max_tenor_years} years` : "—"} />
                  <Row label="Typical deposit (equity)" value={p.min_equity_pct != null ? `${p.min_equity_pct}%` : "—"} />
                  <Row label="Min income" value={money(p.min_income)} />
                  <Row label="Loan range" value={`${money(p.min_loan_amount)} – ${money(p.max_loan_amount)}`} />
                </dl>
                <p className="mt-4 text-xs text-muted">
                  {p.verification_status === "verified" ? "Verified" : "Needs verification"}
                  {p.last_verified_at ? ` · ${new Date(p.last_verified_at).toLocaleDateString(country?.locale || "en")}` : ""}
                </p>
                <Link href={`/mortgages/${p.id}`} className="mt-3 inline-block text-sm font-semibold text-leaf">
                  View details →
                </Link>
              </article>
            );
          })}
        </div>
      </main>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3 border-b border-[color:var(--line)] pb-2">
      <dt className="text-muted">{label}</dt>
      <dd className="text-right font-medium">{value}</dd>
    </div>
  );
}
