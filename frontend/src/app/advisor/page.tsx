"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { QUEUE_BUCKETS, fileRef, queueBucket, relativeTime, statusLabel } from "@/lib/cases";

type Case = {
  id: string;
  customer_name: string;
  customer_email: string;
  status: string;
  next_action_text: string;
  updated_at: string;
};

export default function AdvisorDeskPage() {
  const [cases, setCases] = useState<Case[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api<{ cases: Case[] }>("/api/v1/advisor/cases")
      .then((d) => setCases(d.cases || []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const grouped = useMemo(() => {
    const map: Record<string, Case[]> = { review: [], waiting: [], ready: [], other: [] };
    for (const c of cases) map[queueBucket(c.status)].push(c);
    return map;
  }, [cases]);

  const reviewCount = grouped.review.length;
  const waitingCount = grouped.waiting.length;

  return (
    <div className="advisor-desk min-h-screen">
      <div className="mx-auto max-w-6xl px-5 py-10 md:px-8">
        <p className="text-xs font-semibold uppercase tracking-[0.18em] text-[#8a6d28]">Advisor desk</p>
        <div className="mt-2 flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 className="font-[family-name:var(--font-display)] text-4xl font-semibold tracking-tight">Working files</h1>
            <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted">
              Help a salaried buyer get document-ready and product-fit. You work the file; admin assigns it and records the top-level outcome.
            </p>
          </div>
          {!loading && cases.length > 0 && (
            <p className="font-[family-name:var(--font-display)] text-sm text-[#8a6d28]">
              {reviewCount} for you · {waitingCount} on the buyer · {cases.length} open
            </p>
          )}
        </div>

        {error && <p className="mt-6 text-sm text-[color:var(--danger)]">{error}</p>}

        {loading && <p className="mt-10 text-sm text-muted">Opening your desk…</p>}

        {!loading && cases.length === 0 && !error && (
          <div className="file-jacket mt-10 rounded-sm border border-[#c4a35a]/35 px-6 py-12 md:px-10">
            <p className="text-xs font-semibold uppercase tracking-[0.16em] text-[#8a6d28]">Empty tray</p>
            <h2 className="mt-2 font-[family-name:var(--font-display)] text-2xl font-semibold">No files on your desk yet</h2>
            <p className="mt-3 max-w-lg text-sm leading-relaxed text-muted">
              An admin assigns cases from the console. When a buyer is yours, their file lands here — situation, documents, product fit, then a clear next step.
            </p>
          </div>
        )}

        {!loading && cases.length > 0 && (
          <div className="mt-10 grid gap-8 lg:grid-cols-2">
            {QUEUE_BUCKETS.map((bucket) => {
              const list = grouped[bucket.key] || [];
              if (list.length === 0 && bucket.key === "other") return null;
              return (
                <section key={bucket.key} className={bucket.key === "review" ? "lg:col-span-2" : ""}>
                  <div className="mb-3 flex items-baseline justify-between gap-3">
                    <h2 className="font-[family-name:var(--font-display)] text-xl font-semibold">{bucket.title}</h2>
                    <span className="text-xs font-semibold uppercase tracking-wider text-[#8a6d28]">{list.length}</span>
                  </div>
                  <p className="mb-4 text-sm text-muted">{bucket.hint}</p>
                  {list.length === 0 ? (
                    <p className="rounded-sm border border-dashed border-[color:var(--line)] px-4 py-6 text-sm text-muted">Nothing in this tray.</p>
                  ) : (
                    <ul className={bucket.key === "review" ? "grid gap-3 md:grid-cols-2" : "space-y-3"}>
                      {list.map((c) => (
                        <li key={c.id}>
                          <Link
                            href={`/advisor/cases/${c.id}`}
                            className="file-jacket group flex overflow-hidden rounded-sm border border-[#c4a35a]/30 transition hover:-translate-y-0.5 hover:border-[#8a6d28]/50"
                          >
                            <span className="file-spine w-1.5 shrink-0" aria-hidden />
                            <div className="flex min-w-0 flex-1 flex-wrap items-start justify-between gap-3 px-5 py-4">
                              <div className="min-w-0">
                                <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[#8a6d28]">
                                  {fileRef(c.id)}
                                </p>
                                <h3 className="mt-1 truncate font-[family-name:var(--font-display)] text-lg font-semibold">
                                  {c.customer_name || c.customer_email}
                                </h3>
                                <p className="truncate text-sm text-muted">{c.customer_email}</p>
                                <p className="mt-2 text-sm leading-snug">{c.next_action_text || "Open the file to set the next action."}</p>
                              </div>
                              <div className="flex flex-col items-end gap-2">
                                <span className="rounded-sm bg-[#8a6d28]/10 px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wide text-[#8a6d28]">
                                  {statusLabel(c.status)}
                                </span>
                                <span className="text-xs text-muted">{relativeTime(c.updated_at)}</span>
                                <span className="text-xs font-semibold text-leaf opacity-0 transition group-hover:opacity-100">
                                  Open file →
                                </span>
                              </div>
                            </div>
                          </Link>
                        </li>
                      ))}
                    </ul>
                  )}
                </section>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
