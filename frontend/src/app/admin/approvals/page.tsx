"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { advisorName, statusLabel } from "@/lib/cases";

type CaseRow = {
  id: string;
  customer_name: string;
  customer_email: string;
  status: string;
  assigned_advisor_name?: string;
  assigned_advisor_email?: string;
  next_action_text: string;
  updated_at: string;
};

export default function AdminApprovalsPage() {
  const [cases, setCases] = useState<CaseRow[]>([]);
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState<string | null>(null);

  async function load() {
    const d = await api<{ cases: CaseRow[]; note: string }>("/api/v1/admin/approvals");
    setCases(d.cases || []);
    setNote(d.note || "");
  }

  useEffect(() => {
    load().catch((e) => setError(e.message));
  }, []);

  async function setStatus(id: string, status: string) {
    setBusy(id + status);
    setError("");
    try {
      await api(`/api/v1/admin/cases/${id}/status`, {
        method: "PATCH",
        body: JSON.stringify({ status }),
      });
      setMessage(`Marked ${statusLabel(status).toLowerCase()}.`);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not update.");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="mx-auto max-w-4xl px-5 py-10 md:px-8">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-gold">Admin console</p>
      <h1 className="mt-2 font-[family-name:var(--font-display)] text-3xl font-semibold">Top-level approvals</h1>
      <p className="mt-2 max-w-2xl text-sm text-muted">
        {note ||
          "When an advisor marks a case ready for submission, it appears here. Exception and lender-offer approvals will be added later."}{" "}
        Recording a case outcome here is not a bank approval.
      </p>
      {error && <p className="mt-4 text-sm text-[color:var(--danger)]">{error}</p>}
      {message && <p className="mt-4 text-sm text-leaf">{message}</p>}

      <div className="mt-8 space-y-3">
        {cases.map((c) => (
          <article key={c.id} className="rounded-xl border border-[color:var(--line)] bg-white/80 p-5">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h2 className="font-semibold">{c.customer_name || c.customer_email}</h2>
                <p className="text-sm text-muted">{c.customer_email}</p>
                <p className="mt-2 text-sm">{c.next_action_text}</p>
                <p className="mt-1 text-xs text-muted">Advisor: {advisorName(c)}</p>
              </div>
              <span className="rounded-md bg-paper-2 px-3 py-1 text-xs font-semibold">{statusLabel(c.status)}</span>
            </div>
            <div className="mt-4 flex flex-wrap gap-2">
              <button
                type="button"
                disabled={busy !== null}
                onClick={() => setStatus(c.id, "SUBMITTED_TO_LENDER")}
                className="rounded-md bg-ink px-3 py-2 text-xs font-semibold text-paper disabled:opacity-60"
              >
                {busy === c.id + "SUBMITTED_TO_LENDER" ? "Saving…" : "Release to lender"}
              </button>
              <button
                type="button"
                disabled={busy !== null}
                onClick={() => setStatus(c.id, "APPROVED")}
                className="rounded-md bg-leaf px-3 py-2 text-xs font-semibold text-white disabled:opacity-60"
              >
                {busy === c.id + "APPROVED" ? "Saving…" : "Record case approved"}
              </button>
              <button
                type="button"
                disabled={busy !== null}
                onClick={() => setStatus(c.id, "REJECTED")}
                className="rounded-md border border-[color:var(--danger)]/30 px-3 py-2 text-xs font-semibold text-[color:var(--danger)] disabled:opacity-60"
              >
                {busy === c.id + "REJECTED" ? "Saving…" : "Reject"}
              </button>
              <Link href="/admin/cases" className="rounded-md px-3 py-2 text-xs font-semibold text-leaf hover:underline">
                Open in assignment
              </Link>
            </div>
          </article>
        ))}
        {cases.length === 0 && <p className="text-sm text-muted">Nothing waiting for a top-level decision.</p>}
      </div>
    </div>
  );
}
