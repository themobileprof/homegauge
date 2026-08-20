"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { fileRef, relativeTime, statusLabel } from "@/lib/cases";

type LenderMe = { lender: { id: string; name: string; country_code: string; description: string }; email: string };

type Case = {
  id: string;
  customer_name: string;
  customer_email: string;
  status: string;
  next_action_text: string;
  preferred_product_name?: string;
  updated_at: string;
};

const buckets: { key: string; title: string; statuses: string[] }[] = [
  { key: "in", title: "New to you", statuses: ["SUBMITTED_TO_LENDER"] },
  { key: "review", title: "In your review", statuses: ["LENDER_REVIEW"] },
  { key: "waiting", title: "Waiting on the buyer", statuses: ["ADDITIONAL_INFORMATION_REQUIRED"] },
  { key: "closed", title: "Recorded outcomes", statuses: ["APPROVED", "REJECTED", "COMPLETED"] },
];

export default function LenderHomePage() {
  const [me, setMe] = useState<LenderMe | null>(null);
  const [cases, setCases] = useState<Case[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      api<LenderMe>("/api/v1/lender/me"),
      api<{ cases: Case[] }>("/api/v1/lender/pipeline"),
    ])
      .then(([m, p]) => {
        setMe(m);
        setCases(p.cases || []);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const grouped = useMemo(() => {
    const map: Record<string, Case[]> = {};
    for (const b of buckets) map[b.key] = [];
    for (const c of cases) {
      const b = buckets.find((x) => x.statuses.includes(c.status));
      if (b) map[b.key].push(c);
    }
    return map;
  }, [cases]);

  return (
    <div className="lender-desk min-h-screen">
      <div className="mx-auto max-w-6xl px-5 py-10 md:px-8">
        <p className="text-xs font-semibold uppercase tracking-[0.18em] text-[#1f4d6b]">Lender portal</p>
        <h1 className="mt-2 font-[family-name:var(--font-display)] text-4xl font-semibold tracking-tight">
          {me?.lender.name || "Referral pipeline"}
        </h1>
        <p className="mt-3 max-w-2xl text-sm leading-relaxed text-muted">
          HomeGauge-prepared files submitted to your products. Record that you are reviewing, or ask for more information.
          A recorded outcome here is still not a bank approval on HomeGauge — admin records the top-level case status.
        </p>
        <p className="mt-4">
          <Link href="/lender/products" className="text-sm font-semibold text-[#1f4d6b] hover:underline">
            Manage your products →
          </Link>
        </p>
        {error && <p className="mt-6 text-sm text-[color:var(--danger)]">{error}</p>}
        {loading && <p className="mt-8 text-sm text-muted">Opening pipeline…</p>}

        {!loading && !error && cases.length === 0 && (
          <div className="lender-jacket mt-10 rounded-sm border border-[#1f4d6b]/20 px-6 py-12">
            <h2 className="font-[family-name:var(--font-display)] text-2xl font-semibold">Nothing in the pipeline yet</h2>
            <p className="mt-3 max-w-lg text-sm text-muted">
              When an advisor submits a file against one of your products, it lands here. Lenders without a portal account are updated by the advisor instead.
            </p>
          </div>
        )}

        {!loading && cases.length > 0 && (
          <div className="mt-10 grid gap-8 lg:grid-cols-2">
            {buckets.map((b) => {
              const list = grouped[b.key] || [];
              if (list.length === 0 && b.key === "closed") return null;
              return (
                <section key={b.key}>
                  <div className="mb-3 flex items-baseline justify-between">
                    <h2 className="font-[family-name:var(--font-display)] text-xl font-semibold">{b.title}</h2>
                    <span className="text-xs font-semibold uppercase tracking-wider text-[#1f4d6b]">{list.length}</span>
                  </div>
                  {list.length === 0 ? (
                    <p className="rounded-sm border border-dashed border-[color:var(--line)] px-4 py-6 text-sm text-muted">Empty.</p>
                  ) : (
                    <ul className="space-y-3">
                      {list.map((c) => (
                        <li key={c.id}>
                          <Link
                            href={`/lender/cases/${c.id}`}
                            className="lender-jacket group flex overflow-hidden rounded-sm border border-[#1f4d6b]/15 hover:-translate-y-0.5"
                          >
                            <span className="lender-spine w-1.5 shrink-0" aria-hidden />
                            <div className="flex flex-1 flex-wrap items-start justify-between gap-3 px-5 py-4">
                              <div>
                                <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[#1f4d6b]">{fileRef(c.id)}</p>
                                <h3 className="mt-1 font-[family-name:var(--font-display)] text-lg font-semibold">
                                  {c.customer_name || c.customer_email}
                                </h3>
                                <p className="text-sm text-muted">{c.preferred_product_name}</p>
                                <p className="mt-2 text-sm">{c.next_action_text}</p>
                              </div>
                              <div className="text-right">
                                <span className="rounded-sm bg-[#1f4d6b]/10 px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wide text-[#1f4d6b]">
                                  {statusLabel(c.status)}
                                </span>
                                <p className="mt-2 text-xs text-muted">{relativeTime(c.updated_at)}</p>
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
