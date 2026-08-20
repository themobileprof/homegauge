"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useCountry } from "@/lib/country";
import { readinessCostsFromProduct } from "@/lib/journey";
import { formatRate, isNegotiableRate } from "@/lib/rates";

type Product = {
  id: string;
  lender_name: string;
  name: string;
  description: string;
  interest_rate: number | null;
  interest_rate_min: number | null;
  interest_rate_max: number | null;
  interest_rate_type: string;
  max_tenor_years: number | null;
  min_equity_pct: number | null;
  min_income: number | null;
  min_loan_amount: number | null;
  max_loan_amount: number | null;
  processing_fee: number | null;
  valuation_fee: number | null;
  legal_fee: number | null;
  source: string | null;
  source_url: string | null;
  verification_status: string;
  last_verified_at: string | null;
  documents?: { label: string; category: string; required: boolean; instructions: string }[];
  rules?: { field: string; message_template: string; severity: string }[];
};

export default function ProductDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { money, country } = useCountry();
  const [product, setProduct] = useState<Product | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ product: Product }>(`/api/v1/mortgage-products/${id}`)
      .then((d) => setProduct(d.product))
      .catch((e) => setError(e.message));
  }, [id]);

  return (
    <div className="mx-auto min-h-screen max-w-3xl px-5 py-8">
      <Link href="/mortgages" className="text-sm font-semibold text-leaf">← All options</Link>
      {error && <p className="mt-6 text-sm text-[color:var(--danger)]">{error}</p>}
      {product && (
        <>
          <p className="mt-8 text-sm font-semibold uppercase tracking-wide text-leaf">{product.lender_name}</p>
          <h1 className="mt-2 font-[family-name:var(--font-display)] text-4xl font-semibold">{product.name}</h1>
          <p className="mt-4 text-muted leading-relaxed">{product.description}</p>
          <dl className="mt-8 grid gap-3 sm:grid-cols-2">
            <Stat label="Interest rate" value={formatRate(product)} />
            <Stat label="Max tenor" value={product.max_tenor_years ? `${product.max_tenor_years} years` : "—"} />
            <Stat label="Min equity" value={product.min_equity_pct != null ? `${product.min_equity_pct}%` : "—"} />
            <Stat label="Min income" value={money(product.min_income)} />
            <Stat label="Loan size" value={`${money(product.min_loan_amount)} – ${money(product.max_loan_amount)}`} />
            <Stat label="Verification" value={`${product.verification_status}${product.last_verified_at ? ` · ${new Date(product.last_verified_at).toLocaleDateString(country?.locale || "en")}` : ""}`} />
          </dl>
          {isNegotiableRate(product) && (
            <p className="mt-3 text-xs text-muted">
              This is a stated band, not a personal offer. The lender sets the actual rate after underwriting.
            </p>
          )}
          {product.source_url && (
            <p className="mt-4 text-xs text-muted">
              Source: <a className="underline" href={product.source_url} target="_blank" rel="noreferrer">{product.source || product.source_url}</a>
            </p>
          )}
          <h2 className="mt-10 text-xl font-semibold">Get ready — known costs</h2>
          <p className="mt-2 text-sm text-muted">
            What salaried buyers typically need to budget before disbursement. HomeGauge does not collect these yet — confirm with the lender.
          </p>
          <ul className="mt-4 space-y-3 text-sm">
            {readinessCostsFromProduct(product).map((c) => (
              <li key={c.key} className="rounded-md bg-paper-2/80 px-3 py-3">
                <div className="flex justify-between gap-3">
                  <span className="font-medium">{c.label}</span>
                  <span>{c.amount != null ? money(c.amount) : "Confirm"}</span>
                </div>
                <p className="mt-1 text-xs text-muted">{c.note}</p>
              </li>
            ))}
          </ul>
          <h2 className="mt-10 text-xl font-semibold">Documents usually required</h2>
          <ul className="mt-4 space-y-2 text-sm">
            {(product.documents || []).map((d) => (
              <li key={d.label} className="rounded-md bg-paper-2/80 px-3 py-2">
                <span className="font-medium">{d.label}</span>
                {d.required ? " · required" : " · optional"}
                <span className="text-muted"> · {d.category}</span>
              </li>
            ))}
          </ul>
          <Link href="/register?next=/app/assessment" className="mt-10 inline-flex rounded-md bg-leaf px-5 py-3 text-sm font-semibold text-white">
            Check if you may qualify
          </Link>
          <p className="mt-4 text-xs text-muted">Eligibility estimates are educational and are not a bank approval.</p>
        </>
      )}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-[color:var(--line)] bg-white/70 p-4">
      <dt className="text-xs uppercase tracking-wide text-muted">{label}</dt>
      <dd className="mt-1 font-semibold">{value}</dd>
    </div>
  );
}
