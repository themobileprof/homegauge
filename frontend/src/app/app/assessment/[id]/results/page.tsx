"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { api, outcomeLabel } from "@/lib/api";
import { useCountry } from "@/lib/country";

type Result = {
  product_id: string;
  product_name: string;
  lender_name: string;
  outcome: string;
  explanation: string;
  estimated_monthly_repayment: number | null;
  interest_rate: number | null;
  min_equity_pct: number | null;
  verification_status: string;
  last_verified_at: string | null;
};

type Assessment = {
  id: string;
  status: string;
  results?: Result[];
  readiness?: {
    total: number;
    narrative: string;
    components: { key: string; label: string; score: number; max: number; note: string }[];
  };
  best_fit_product_id?: string;
  best_fit_why?: string;
};

export default function ResultsPage() {
  const { money } = useCountry();
  const { id } = useParams<{ id: string }>();
  const [assessment, setAssessment] = useState<Assessment | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ assessment: Assessment }>(`/api/v1/assessments/${id}`)
      .then((d) => setAssessment(d.assessment))
      .catch((e) => setError(e.message));
  }, [id]);

  return (
    <div className="mx-auto min-h-screen max-w-3xl px-5 py-8">
      <Link href="/app" className="font-[family-name:var(--font-display)] text-xl font-semibold">
        Home<span className="text-leaf">Gauge</span>
      </Link>
      <h1 className="mt-8 font-[family-name:var(--font-display)] text-3xl font-semibold">Your eligibility results</h1>
      <p className="mt-2 text-sm text-muted">
        Based on the information provided, you appear to meet — or not meet — stated product criteria. This is not a bank approval.
      </p>
      {error && <p className="mt-6 text-sm text-[color:var(--danger)]">{error}</p>}

      {assessment?.readiness && (
        <section className="mt-8 rounded-2xl border border-[color:var(--line)] bg-white/80 p-6">
          <p className="text-sm font-semibold uppercase tracking-wide text-leaf">Mortgage readiness</p>
          <p className="mt-2 font-[family-name:var(--font-display)] text-5xl font-semibold">{assessment.readiness.total}<span className="text-2xl text-muted">/100</span></p>
          <p className="mt-3 text-sm leading-relaxed text-muted">{assessment.readiness.narrative}</p>
          <ul className="mt-6 space-y-3">
            {assessment.readiness.components.map((c) => (
              <li key={c.key}>
                <div className="mb-1 flex justify-between text-sm">
                  <span className="font-medium">{c.label}</span>
                  <span>{c.score}/{c.max}</span>
                </div>
                <div className="h-2 overflow-hidden rounded-full bg-paper-2">
                  <div className="h-full bg-leaf" style={{ width: `${(c.score / c.max) * 100}%` }} />
                </div>
                <p className="mt-1 text-xs text-muted">{c.note}</p>
              </li>
            ))}
          </ul>
        </section>
      )}

      {assessment?.best_fit_why && (
        <section className="mt-6 rounded-xl bg-[#0c1f17] p-5 text-paper">
          <p className="text-sm uppercase tracking-wide text-gold">Best fit for you</p>
          <p className="mt-2 text-sm leading-relaxed">{assessment.best_fit_why}</p>
        </section>
      )}

      <section className="mt-8 space-y-4">
        <h2 className="text-xl font-semibold">Product-by-product</h2>
        {(assessment?.results || []).map((r) => (
          <article key={r.product_id} className="rounded-xl border border-[color:var(--line)] bg-white/70 p-5">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <p className="text-xs font-semibold uppercase text-leaf">{r.lender_name}</p>
                <h3 className="text-lg font-semibold">{r.product_name}</h3>
              </div>
              <span className="rounded-md bg-paper-2 px-3 py-1 text-xs font-semibold">{outcomeLabel(r.outcome)}</span>
            </div>
            <p className="mt-3 text-sm text-muted">{r.explanation}</p>
            <dl className="mt-4 grid gap-2 text-sm sm:grid-cols-3">
              <div>Rate: <strong>{r.interest_rate != null ? `${r.interest_rate}%` : "—"}</strong></div>
              <div>Equity: <strong>{r.min_equity_pct != null ? `${r.min_equity_pct}%` : "—"}</strong></div>
              <div>Est. monthly: <strong>{money(r.estimated_monthly_repayment)}</strong></div>
            </dl>
            <p className="mt-3 text-xs text-muted">
              {r.verification_status}
              {r.last_verified_at ? ` · last verified ${new Date(r.last_verified_at).toLocaleDateString("en-NG")}` : ""}
            </p>
            <Link href={`/mortgages/${r.product_id}`} className="mt-3 inline-block text-sm font-semibold text-leaf">
              View product →
            </Link>
          </article>
        ))}
      </section>

      <div className="mt-10 flex flex-wrap gap-3">
        <Link href="/app" className="rounded-md bg-ink px-5 py-3 text-sm font-semibold text-paper">Go to dashboard</Link>
        <Link href="/calculator" className="rounded-md border border-[color:var(--line)] px-5 py-3 text-sm font-semibold">Open calculator</Link>
      </div>
    </div>
  );
}
