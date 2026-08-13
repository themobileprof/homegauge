"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";
import { api } from "@/lib/api";

type CaseDetail = {
  case: {
    id: string;
    customer_name: string;
    customer_email: string;
    status: string;
    next_action_text: string;
    user_id: string;
  };
  notes: { id: string; author_email: string; body: string; visibility: string; created_at: string }[];
  suggestions: { id: string; suggestion_type: string; payload: { message?: string }; status: string; confidence?: number }[];
};

const statuses = [
  "DOCUMENTS_PENDING",
  "DOCUMENTS_UNDER_REVIEW",
  "READY_FOR_SUBMISSION",
  "SUBMITTED_TO_LENDER",
  "LENDER_REVIEW",
  "ADDITIONAL_INFORMATION_REQUIRED",
  "APPROVED",
  "REJECTED",
  "COMPLETED",
  "CANCELLED",
];

export default function AdvisorCasePage() {
  const { id } = useParams<{ id: string }>();
  const [data, setData] = useState<CaseDetail | null>(null);
  const [note, setNote] = useState("");
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  async function load() {
    const d = await api<CaseDetail>(`/api/v1/advisor/cases/${id}`);
    setData(d);
    setStatus(d.case.status);
  }

  useEffect(() => {
    load().catch((e) => setError(e.message));
  }, [id]);

  async function saveStatus(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await api(`/api/v1/advisor/cases/${id}/status`, {
        method: "PATCH",
        body: JSON.stringify({ status, next_action_text: data?.case.next_action_text }),
      });
      setMessage("Status updated.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed");
    }
  }

  async function addNote(e: FormEvent) {
    e.preventDefault();
    try {
      await api(`/api/v1/advisor/cases/${id}/notes`, {
        method: "POST",
        body: JSON.stringify({ body: note, visibility: "internal" }),
      });
      setNote("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed");
    }
  }

  return (
    <div className="mx-auto min-h-screen max-w-3xl px-5 py-10">
      <Link href="/advisor" className="text-sm font-semibold text-leaf">← Case queue</Link>
      {error && <p className="mt-4 text-sm text-[color:var(--danger)]">{error}</p>}
      {message && <p className="mt-4 text-sm text-leaf">{message}</p>}
      {data && (
        <>
          <h1 className="mt-6 font-[family-name:var(--font-display)] text-3xl font-semibold">
            {data.case.customer_name || data.case.customer_email}
          </h1>
          <p className="text-sm text-muted">{data.case.customer_email}</p>
          <p className="mt-3 text-sm">{data.case.next_action_text}</p>

          <form onSubmit={saveStatus} className="mt-8 flex flex-wrap items-end gap-3">
            <label className="text-sm">
              <span className="mb-1 block font-medium">Status</span>
              <select value={status} onChange={(e) => setStatus(e.target.value)} className="rounded-md border border-[color:var(--line)] bg-white px-3 py-2">
                {statuses.map((s) => <option key={s}>{s}</option>)}
              </select>
            </label>
            <button type="submit" className="rounded-md bg-ink px-4 py-2 text-sm font-semibold text-paper">Update</button>
          </form>

          <section className="mt-10">
            <h2 className="text-lg font-semibold">AI / concierge suggestions</h2>
            <ul className="mt-4 space-y-3">
              {(data.suggestions || []).map((s) => (
                <li key={s.id} className="rounded-lg border border-[color:var(--line)] bg-white/70 p-4 text-sm">
                  <p className="font-semibold">{s.suggestion_type} · {s.status}</p>
                  <p className="mt-1 text-muted">{s.payload?.message || JSON.stringify(s.payload)}</p>
                  {s.status === "pending" && (
                    <div className="mt-3 flex gap-2">
                      <button
                        type="button"
                        className="rounded-md bg-leaf px-3 py-1.5 text-xs font-semibold text-white"
                        onClick={() =>
                          api(`/api/v1/advisor/suggestions/${s.id}/resolve`, {
                            method: "POST",
                            body: JSON.stringify({ status: "approved" }),
                          }).then(load)
                        }
                      >
                        Approve
                      </button>
                      <button
                        type="button"
                        className="rounded-md bg-paper-2 px-3 py-1.5 text-xs font-semibold"
                        onClick={() =>
                          api(`/api/v1/advisor/suggestions/${s.id}/resolve`, {
                            method: "POST",
                            body: JSON.stringify({ status: "rejected" }),
                          }).then(load)
                        }
                      >
                        Reject
                      </button>
                    </div>
                  )}
                </li>
              ))}
              {(data.suggestions || []).length === 0 && <p className="text-sm text-muted">No suggestions yet.</p>}
            </ul>
          </section>

          <section className="mt-10">
            <h2 className="text-lg font-semibold">Notes</h2>
            <form onSubmit={addNote} className="mt-3 space-y-3">
              <textarea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                required
                rows={3}
                className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2 text-sm"
                placeholder="Internal note…"
              />
              <button type="submit" className="rounded-md bg-leaf px-4 py-2 text-sm font-semibold text-white">Add note</button>
            </form>
            <ul className="mt-4 space-y-3">
              {(data.notes || []).map((n) => (
                <li key={n.id} className="rounded-lg bg-paper-2/80 p-3 text-sm">
                  <p className="text-xs text-muted">{n.author_email} · {new Date(n.created_at).toLocaleString("en-NG")}</p>
                  <p className="mt-1">{n.body}</p>
                </li>
              ))}
            </ul>
          </section>
        </>
      )}
    </div>
  );
}
