"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useCountry } from "@/lib/country";
import { formatRate } from "@/lib/rates";

type Product = {
  id: string;
  lender_name: string;
  name: string;
  interest_rate: number | null;
  interest_rate_min: number | null;
  interest_rate_max: number | null;
  interest_rate_type: string;
  max_tenor_years: number | null;
  min_equity_pct: number | null;
  min_income: number | null;
  processing_fee: number | null;
  valuation_fee: number | null;
  legal_fee: number | null;
  verification_status: string;
  last_verified_at: string | null;
  documents?: { label: string; required: boolean }[];
};

function CompareInner() {
  const { money, country } = useCountry();
  const params = useSearchParams();
  const ids = (params.get("ids") || "").split(",").filter(Boolean);
  const [products, setProducts] = useState<Product[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    if (ids.length < 2) {
      setError("Select at least two products to compare.");
      return;
    }
    api<{ products: Product[] }>("/api/v1/mortgage-products/compare", {
      method: "POST",
      body: JSON.stringify({ product_ids: ids }),
    })
      .then((d) => setProducts(d.products || []))
      .catch((e) => setError(e.message));
  }, [params]);

  return (
    <div>
      <main className="mx-auto max-w-6xl overflow-x-auto px-5 py-10 pb-20">
        <Link href="/mortgages" className="text-sm font-semibold text-leaf">← Mortgage options</Link>
        <h1 className="mt-6 font-[family-name:var(--font-display)] text-3xl font-semibold">Compare mortgages</h1>
        <p className="mt-2 text-sm text-muted">Plain-language comparison. Confirm current terms with the lender.</p>
        {error && <p className="mt-6 text-sm text-[color:var(--danger)]">{error}</p>}
        {products.length > 0 && (
          <table className="mt-8 w-full min-w-[720px] border-collapse text-sm">
            <thead>
              <tr className="border-b border-[color:var(--line)] text-left">
                <th className="py-3 pr-4 font-medium text-muted">Detail</th>
                {products.map((p) => (
                  <th key={p.id} className="px-3 py-3 font-semibold">{p.name}<div className="text-xs font-normal text-muted">{p.lender_name}</div></th>
                ))}
              </tr>
            </thead>
            <tbody>
              <CmpRow label="Interest rate" values={products.map((p) => formatRate(p))} />
              <CmpRow label="Years to repay" values={products.map((p) => (p.max_tenor_years ? `${p.max_tenor_years}` : "—"))} />
              <CmpRow label="Deposit needed" values={products.map((p) => (p.min_equity_pct != null ? `${p.min_equity_pct}%` : "—"))} />
              <CmpRow label="Minimum income" values={products.map((p) => money(p.min_income))} />
              <CmpRow label="Processing fee" values={products.map((p) => money(p.processing_fee))} />
              <CmpRow label="Valuation fee" values={products.map((p) => money(p.valuation_fee))} />
              <CmpRow label="Legal fee" values={products.map((p) => money(p.legal_fee))} />
              <CmpRow label="Verification" values={products.map((p) => `${p.verification_status}${p.last_verified_at ? ` · ${new Date(p.last_verified_at).toLocaleDateString(country?.locale || "en")}` : ""}`)} />
              <CmpRow label="Key documents" values={products.map((p) => (p.documents || []).filter((d) => d.required).map((d) => d.label).slice(0, 4).join("; ") || "—")} />
            </tbody>
          </table>
        )}
      </main>
    </div>
  );
}

function CmpRow({ label, values }: { label: string; values: string[] }) {
  return (
    <tr className="border-b border-[color:var(--line)] align-top">
      <td className="py-3 pr-4 text-muted">{label}</td>
      {values.map((v, i) => (
        <td key={i} className="px-3 py-3 font-medium">{v}</td>
      ))}
    </tr>
  );
}

export default function ComparePage() {
  return (
    <Suspense fallback={<p className="p-8 text-muted">Loading comparison…</p>}>
      <CompareInner />
    </Suspense>
  );
}
