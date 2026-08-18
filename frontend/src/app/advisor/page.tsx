"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";

type Case = {
  id: string;
  customer_name: string;
  customer_email: string;
  status: string;
  next_action_text: string;
  updated_at: string;
};

export default function AdvisorPage() {
  const [cases, setCases] = useState<Case[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ cases: Case[] }>("/api/v1/advisor/cases")
      .then((d) => setCases(d.cases || []))
      .catch((e) => setError(e.message));
  }, []);

  return (
    <div className="mx-auto max-w-5xl px-5 py-10 md:px-8">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-[#8a6d28]">Advisor workspace</p>
      <h1 className="mt-2 font-[family-name:var(--font-display)] text-3xl font-semibold">Your cases</h1>
      <p className="mt-2 text-sm text-muted">
        Files assigned to you. Review eligibility, documents, and next actions. Assignment and top-level status sit with admin.
      </p>
      {error && <p className="mt-4 text-sm text-[color:var(--danger)]">{error}</p>}
      <div className="mt-8 space-y-3">
        {cases.map((c) => (
          <Link
            key={c.id}
            href={`/advisor/cases/${c.id}`}
            className="block rounded-xl border border-[color:var(--line)] bg-white/70 p-5 hover:border-leaf"
          >
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h2 className="font-semibold">{c.customer_name || c.customer_email}</h2>
                <p className="text-sm text-muted">{c.customer_email}</p>
                <p className="mt-2 text-sm">{c.next_action_text}</p>
              </div>
              <span className="rounded-md bg-paper-2 px-3 py-1 text-xs font-semibold">{c.status}</span>
            </div>
          </Link>
        ))}
        {!error && cases.length === 0 && <p className="text-muted">No cases assigned to you yet. An admin will assign work from the console.</p>}
      </div>
    </div>
  );
}
