"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { api, outcomeLabel } from "@/lib/api";
import { buyerStatusLabel, documentStatusLabel } from "@/lib/cases";
import { useCountry } from "@/lib/country";
import type { DocItem, Note } from "@/lib/advisor-file";

type Assessment = {
  id: string;
  status: string;
  results?: { product_id: string; outcome: string; product_name: string; lender_name: string; estimated_monthly_repayment: number | null }[];
  readiness?: { total: number; narrative: string };
  best_fit_why?: string;
};

type Application = {
  id: string;
  status: string;
  next_action_text: string;
  preferred_product_name?: string;
  lender_name?: string;
  lender_has_account?: boolean;
  assigned_advisor_name?: string;
  assigned_advisor_email?: string;
};

export default function AppHome() {
  const { money } = useCountry();
  const [assessment, setAssessment] = useState<Assessment | null>(null);
  const [app, setApp] = useState<Application | null>(null);
  const [docs, setDocs] = useState<DocItem[]>([]);
  const [notes, setNotes] = useState<Note[]>([]);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    Promise.all([
      api<{ assessment: Assessment }>("/api/v1/assessments/latest")
        .then((d) => d.assessment)
        .catch(() => null),
      api<{ application: Application; documents: DocItem[]; notes: Note[] }>("/api/v1/applications/me")
        .then((d) => d)
        .catch(() => null),
    ])
      .then(([a, file]) => {
        setAssessment(a);
        setApp(file?.application || null);
        setDocs(file?.documents || []);
        setNotes(file?.notes || []);
      })
      .finally(() => setLoaded(true));
  }, []);

  const completed = assessment?.status === "completed";
  const likely = (assessment?.results || []).filter((r) => r.outcome === "likely_eligible" || r.outcome === "potentially_eligible").length;
  const required = docs.filter((d) => d.required);
  const accepted = required.filter((d) => d.status === "accepted").length;
  const sentBack = docs.filter((d) => d.status === "requires_replacement" || d.status === "rejected");
  const withLender = ["SUBMITTED_TO_LENDER", "LENDER_REVIEW"].includes(app?.status || "");
  const outcome = ["APPROVED", "REJECTED", "COMPLETED"].includes(app?.status || "");

  const steps = useMemo(
    () => [
      { done: completed, label: "Eligibility assessment" },
      { done: likely > 0, label: "Mortgage options identified" },
      { done: accepted > 0 && sentBack.length === 0 && required.length > 0 && accepted === required.length, label: "Documents" },
      { done: Boolean(app?.assigned_advisor_name || app?.assigned_advisor_email), label: "Advisor on your file" },
      { done: withLender || outcome, label: "With a lender" },
    ],
    [accepted, app, completed, likely, outcome, required.length, sentBack.length, withLender],
  );

  return (
    <div className="mx-auto max-w-3xl px-5 py-10">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-leaf">Homebuyer workspace</p>
      <h1 className="mt-2 font-[family-name:var(--font-display)] text-4xl font-semibold">Your mortgage journey</h1>
      <p className="mt-3 text-muted">
        Know where you stand, prepare documents, and follow what your advisor and lender need next. HomeGauge is not a bank — only a lender can approve a mortgage.
      </p>

      <ol className="mt-10 space-y-3 text-sm">
        {steps.map((s) => (
          <li key={s.label} className={s.done ? "text-leaf" : "text-muted"}>
            {s.done ? "✓" : "○"} {s.label}
          </li>
        ))}
      </ol>

      {!loaded && <p className="mt-8 text-muted">Loading…</p>}

      {loaded && app && (
        <div className="mt-8 rounded-xl border border-leaf/25 bg-white/80 p-6">
          <p className="text-xs font-semibold uppercase tracking-wide text-leaf">{buyerStatusLabel(app.status)}</p>
          <p className="mt-2 text-sm leading-relaxed">{app.next_action_text || "Continue from the next step below."}</p>
          {app.preferred_product_name && (
            <p className="mt-3 text-sm text-muted">
              Product in view: <strong>{app.preferred_product_name}</strong>
              {app.lender_name ? ` · ${app.lender_name}` : ""}
              {app.lender_has_account === false ? " (your advisor is the liaison for this lender)" : ""}
            </p>
          )}
          {(app.assigned_advisor_name || app.assigned_advisor_email) && (
            <p className="mt-2 text-sm text-muted">Advisor: {app.assigned_advisor_name || app.assigned_advisor_email}</p>
          )}
        </div>
      )}

      {loaded && sentBack.length > 0 && (
        <div className="mt-4 rounded-xl border border-[color:var(--danger)]/25 bg-white/80 p-5">
          <h2 className="font-semibold">Replace these documents</h2>
          <ul className="mt-2 space-y-2 text-sm">
            {sentBack.map((d) => (
              <li key={d.document_type_code}>
                <strong>{d.label}</strong>
                {d.review_notes ? <span className="text-muted"> — {d.review_notes}</span> : null}
              </li>
            ))}
          </ul>
          <Link href="/app/documents" className="mt-3 inline-block text-sm font-semibold text-leaf">
            Open checklist →
          </Link>
        </div>
      )}

      {notes.length > 0 && (
        <div className="mt-4 rounded-xl border border-[color:var(--line)] bg-white/70 p-5">
          <h2 className="font-semibold">Messages on your file</h2>
          <ul className="mt-3 space-y-3 text-sm">
            {notes.slice(0, 3).map((n) => (
              <li key={n.id}>
                <p className="text-xs text-muted">{new Date(n.created_at).toLocaleString("en-NG")}</p>
                <p className="mt-1">{n.body}</p>
              </li>
            ))}
          </ul>
        </div>
      )}

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
                <li key={r.product_id} className="flex justify-between gap-3 border-b border-[color:var(--line)] pb-2">
                  <Link href={`/mortgages/${r.product_id}`} className="font-medium text-leaf hover:underline">
                    {r.product_name}
                  </Link>
                  <span className="text-muted">{outcomeLabel(r.outcome)}</span>
                </li>
              ))}
            </ul>
            {assessment.best_fit_why && <p className="mt-4 text-sm text-muted">{assessment.best_fit_why}</p>}
            <p className="mt-4 text-sm">
              Estimated repayment on top match: <strong>{money(assessment.results?.[0]?.estimated_monthly_repayment)}</strong>
            </p>
            <Link href={`/app/assessment/${assessment.id}/results`} className="mt-4 inline-block text-sm font-semibold text-leaf">
              View full results and choose a product →
            </Link>
          </div>
          <div className="rounded-xl bg-[#0c1f17] p-6 text-paper">
            <h2 className="text-lg font-semibold">Next step</h2>
            <p className="mt-2 text-sm text-paper/80">
              {sentBack.length
                ? "Replace the documents your advisor sent back."
                : withLender
                  ? "Your advisor is liaising with the lender. Watch this page for anything they still need from you."
                  : `Upload your salary-account evidence. ${required.length ? `${accepted} of ${required.length} required documents accepted.` : ""}`}
            </p>
            <div className="mt-4 flex flex-wrap gap-3">
              <Link href="/app/documents" className="inline-flex rounded-md bg-gold px-4 py-2.5 text-sm font-semibold text-ink">
                Open document checklist
              </Link>
              {docs.length > 0 && (
                <p className="self-center text-xs text-paper/70">
                  {docs.filter((d) => d.status !== "not_started").length} of {docs.length} started
                  {docs[0] ? ` · latest ${documentStatusLabel(docs.find((d) => d.status !== "not_started")?.status)}` : ""}
                </p>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
