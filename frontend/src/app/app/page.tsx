"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api, outcomeLabel } from "@/lib/api";
import { useCountry } from "@/lib/country";

type Assessment = {
  id: string;
  status: string;
  results?: { outcome: string; product_name: string; lender_name: string; estimated_monthly_repayment: number | null }[];
  readiness?: { total: number; narrative: string };
  best_fit_why?: string;
};

export default function AppHome() {
  const { money } = useCountry();
  const [assessment, setAssessment] = useState<Assessment | null>(null);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    api<{ assessment: Assessment }>("/api/v1/assessments/latest")
      .then((d) => setAssessment(d.assessment))
      .catch(() => setAssessment(null))
      .finally(() => setLoaded(true));
  }, []);

  const completed = assessment?.status === "completed";
  const likely = (assessment?.results || []).filter((r) => r.outcome === "likely_eligible" || r.outcome === "potentially_eligible").length;

  return (
    <div className="mx-auto max-w-3xl px-5 py-10">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-leaf">Homebuyer workspace</p>
      <h1 className="mt-2 font-[family-name:var(--font-display)] text-4xl font-semibold">Your mortgage journey</h1>
      <p className="mt-3 text-muted">Know where you stand, prepare documents, and get help when you need it. This is your personal file — not the advisor or admin console.</p>

      <ol className="mt-10 space-y-3 text-sm">
        <li className={completed ? "text-leaf" : ""}>{completed ? "✓" : "○"} Eligibility assessment</li>
        <li className={likely > 0 ? "text-leaf" : "text-muted"}>{likely > 0 ? "✓" : "○"} Mortgage options identified</li>
        <li className="text-muted">○ Documents</li>
        <li className="text-muted">○ Application</li>
        <li className="text-muted">○ Bank review</li>
      </ol>

      {!loaded && <p className="mt-8 text-muted">Loading…</p>}

      {loaded && !assessment && (
        <div className="mt-8 rounded-xl border border-[color:var(--line)] bg-white/70 p-6">
          <h2 className="text-lg font-semibold">Next step</h2>
          <p className="mt-2 text-sm text-muted">Check your eligibility using your salary account history.</p>
          <Link href="/app/assessment" className="mt-4 inline-flex rounded-md bg-leaf px-4 py-2.5 text-sm font-semibold text-white">
            Check My Mortgage Eligibility
          </Link>
        </div>
      )}

      {completed && assessment && (
        <div className="mt-8 space-y-4">
          {assessment.readiness && (
            <div className="rounded-xl border border-[color:var(--line)] bg-white/70 p-6">
              <p className="text-sm font-semibold text-leaf">Mortgage readiness</p>
              <p className="mt-1 font-[family-name:var(--font-display)] text-4xl font-semibold">{assessment.readiness.total}/100</p>
              <p className="mt-2 text-sm text-muted">{assessment.readiness.narrative}</p>
            </div>
          )}
          <div className="rounded-xl border border-[color:var(--line)] bg-white/70 p-6">
            <h2 className="text-lg font-semibold">Eligibility summary</h2>
            <p className="mt-2 text-sm text-muted">{likely} product(s) look like a possible fit based on stated criteria.</p>
            <ul className="mt-4 space-y-2 text-sm">
              {(assessment.results || []).slice(0, 3).map((r) => (
                <li key={r.product_name} className="flex justify-between gap-3 border-b border-[color:var(--line)] pb-2">
                  <span>{r.product_name}</span>
                  <span className="text-muted">{outcomeLabel(r.outcome)}</span>
                </li>
              ))}
            </ul>
            {assessment.best_fit_why && <p className="mt-4 text-sm text-muted">{assessment.best_fit_why}</p>}
            <p className="mt-4 text-sm">
              Estimated repayment on top match:{" "}
              <strong>{money(assessment.results?.[0]?.estimated_monthly_repayment)}</strong>
            </p>
            <Link href={`/app/assessment/${assessment.id}/results`} className="mt-4 inline-block text-sm font-semibold text-leaf">
              View full results →
            </Link>
          </div>
          <div className="rounded-xl bg-[#0c1f17] p-6 text-paper">
            <h2 className="text-lg font-semibold">Next step</h2>
            <p className="mt-2 text-sm text-paper/80">Upload your latest 6-month salary account statement so we can confirm your salary pattern.</p>
            <Link href="/app/documents" className="mt-4 inline-flex rounded-md bg-gold px-4 py-2.5 text-sm font-semibold text-ink">
              Open document checklist
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}
