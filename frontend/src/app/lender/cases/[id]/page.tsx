"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";
import { api, outcomeLabel } from "@/lib/api";
import { documentStatusLabel, fileRef, statusLabel } from "@/lib/cases";
import { useCountry } from "@/lib/country";
import type { Assessment, DocItem, Note } from "@/lib/advisor-file";

type CaseRow = {
  id: string;
  customer_name: string;
  customer_email: string;
  status: string;
  next_action_text: string;
  preferred_product_name?: string;
  lender_name?: string;
};

type File = { case: CaseRow; notes: Note[]; documents: DocItem[]; assessment: Assessment | null };

export default function LenderCasePage() {
  const { money } = useCountry();
  const { id } = useParams<{ id: string }>();
  const [data, setData] = useState<File | null>(null);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [note, setNote] = useState("");
  const [shareBuyer, setShareBuyer] = useState(true);
  const [infoAsk, setInfoAsk] = useState("");
  const [busy, setBusy] = useState("");

  async function load() {
    const d = await api<File>(`/api/v1/lender/cases/${id}`);
    setData(d);
  }

  useEffect(() => {
    load().catch((e) => setError(e.message));
  }, [id]);

  async function setStatus(status: string, next: string) {
    setBusy(status);
    setError("");
    try {
      await api(`/api/v1/lender/cases/${id}/status`, {
        method: "PATCH",
        body: JSON.stringify({ status, next_action_text: next }),
      });
      setMessage("Pipeline updated.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not update.");
    } finally {
      setBusy("");
    }
  }

  async function addNote(e: FormEvent) {
    e.preventDefault();
    setBusy("note");
    try {
      await api(`/api/v1/lender/cases/${id}/notes`, {
        method: "POST",
        body: JSON.stringify({ body: note, visibility: shareBuyer ? "customer" : "lender" }),
      });
      setNote("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not add note.");
    } finally {
      setBusy("");
    }
  }

  async function openDoc(doc: DocItem) {
    if (!doc.document_id) return;
    const d = await api<{ url: string }>(`/api/v1/lender/documents/${doc.document_id}/download-url`);
    window.open(d.url, "_blank", "noopener,noreferrer");
  }

  if (!data) {
    return (
      <div className="lender-desk min-h-screen px-5 py-10">
        <div className="mx-auto max-w-4xl">
          <Link href="/lender" className="text-sm font-semibold text-[#1f4d6b]">
            ← Pipeline
          </Link>
          {error ? <p className="mt-6 text-sm text-[color:var(--danger)]">{error}</p> : <p className="mt-8 text-sm text-muted">Opening file…</p>}
        </div>
      </div>
    );
  }

  const c = data.case;
  const in_ = data.assessment?.input_snapshot;

  return (
    <div className="lender-desk min-h-screen pb-16">
      <div className="mx-auto max-w-4xl px-5 py-8 md:px-8">
        <div className="flex items-center justify-between">
          <Link href="/lender" className="text-sm font-semibold text-[#1f4d6b]">
            ← Pipeline
          </Link>
          <p className="font-[family-name:var(--font-display)] text-sm text-[#1f4d6b]">{fileRef(c.id)}</p>
        </div>

        <header className="lender-jacket mt-5 rounded-sm border border-[#1f4d6b]/20 px-6 py-6">
          <p className="text-xs font-semibold uppercase tracking-[0.16em] text-[#1f4d6b]">Referred file</p>
          <h1 className="mt-1 font-[family-name:var(--font-display)] text-3xl font-semibold">{c.customer_name || c.customer_email}</h1>
          <p className="mt-1 text-sm text-muted">{c.customer_email}</p>
          <p className="mt-3 text-sm">
            {c.preferred_product_name} · {statusLabel(c.status)}
          </p>
          <p className="mt-2 text-sm text-muted">{c.next_action_text}</p>
          <p className="mt-3 text-xs text-muted">This is a HomeGauge-prepared file. Recording an update here is not a bank approval.</p>
        </header>

        {error && <p className="mt-4 text-sm text-[color:var(--danger)]">{error}</p>}
        {message && <p className="mt-4 text-sm text-leaf">{message}</p>}

        <section className="lender-jacket mt-8 rounded-sm border border-[#1f4d6b]/15 px-6 py-6">
          <h2 className="font-[family-name:var(--font-display)] text-xl font-semibold">Your update</h2>
          <div className="mt-4 flex flex-wrap gap-2">
            <button
              type="button"
              disabled={busy !== ""}
              onClick={() => setStatus("LENDER_REVIEW", "Lender is reviewing the file.")}
              className="rounded-sm bg-[#1f4d6b] px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
            >
              Mark in review
            </button>
          </div>
          <form
            className="mt-5 space-y-3"
            onSubmit={(e) => {
              e.preventDefault();
              setStatus("ADDITIONAL_INFORMATION_REQUIRED", infoAsk || "Lender needs more information.");
            }}
          >
            <label className="block text-sm">
              <span className="mb-1 block font-medium">Ask the buyer for more</span>
              <textarea
                value={infoAsk}
                onChange={(e) => setInfoAsk(e.target.value)}
                rows={3}
                required
                className="w-full rounded-sm border border-[color:var(--line)] bg-white px-3 py-2 text-sm"
                placeholder="What is missing? This becomes the next action the buyer and advisor see."
              />
            </label>
            <button type="submit" disabled={busy !== ""} className="rounded-sm border border-[color:var(--line)] px-4 py-2 text-sm font-semibold">
              Request more information
            </button>
          </form>
        </section>

        {in_ && (
          <section className="mt-8">
            <h2 className="font-[family-name:var(--font-display)] text-xl font-semibold">Situation</h2>
            <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-2">
              <div>Income: <strong>{money(in_.monthly_net_income)}</strong></div>
              <div>Deposit: <strong>{money(in_.available_deposit)}</strong></div>
              <div>Property: <strong>{money(in_.desired_property_price)}</strong></div>
              <div>Salary months: <strong>{in_.salary_months ?? "—"}</strong></div>
              <div>Employer: <strong>{in_.employer_name || "—"}</strong></div>
            </dl>
          </section>
        )}

        {data.assessment?.results && (
          <section className="mt-8">
            <h2 className="font-[family-name:var(--font-display)] text-xl font-semibold">Eligibility snapshot</h2>
            <ul className="mt-3 space-y-2 text-sm">
              {data.assessment.results.slice(0, 4).map((r) => (
                <li key={r.product_id} className="flex justify-between gap-3 border-b border-[color:var(--line)] pb-2">
                  <span>{r.product_name}</span>
                  <span className="text-muted">{outcomeLabel(r.outcome)}</span>
                </li>
              ))}
            </ul>
          </section>
        )}

        <section className="mt-8">
          <h2 className="font-[family-name:var(--font-display)] text-xl font-semibold">Documents</h2>
          <ul className="mt-4 divide-y divide-[color:var(--line)] border-y border-[color:var(--line)]">
            {(data.documents || []).map((d) => (
              <li key={d.document_type_code} className="flex flex-wrap items-center justify-between gap-3 py-3">
                <div>
                  <p className="font-semibold">{d.label}</p>
                  <p className="text-xs text-muted">{documentStatusLabel(d.status)}</p>
                </div>
                {d.document_id && (
                  <button type="button" onClick={() => openDoc(d)} className="text-sm font-semibold text-[#1f4d6b]">
                    Open
                  </button>
                )}
              </li>
            ))}
          </ul>
        </section>

        <section className="mt-8">
          <h2 className="font-[family-name:var(--font-display)] text-xl font-semibold">Notes</h2>
          <form onSubmit={addNote} className="mt-4 space-y-3">
            <textarea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              required
              rows={3}
              className="w-full rounded-sm border border-[color:var(--line)] bg-white px-3 py-2 text-sm"
              placeholder="What you told HomeGauge or the buyer…"
            />
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={shareBuyer} onChange={(e) => setShareBuyer(e.target.checked)} />
              Visible to the buyer
            </label>
            <button type="submit" disabled={busy === "note"} className="rounded-sm bg-[#1f4d6b] px-4 py-2 text-sm font-semibold text-white">
              Add note
            </button>
          </form>
          <ol className="mt-6 space-y-3 text-sm">
            {(data.notes || []).map((n) => (
              <li key={n.id}>
                <p className="text-xs text-muted">
                  {n.visibility === "customer" ? "Buyer can see" : "Lender / advisor"} · {new Date(n.created_at).toLocaleString("en-NG")}
                </p>
                <p className="mt-1">{n.body}</p>
              </li>
            ))}
          </ol>
        </section>
      </div>
    </div>
  );
}
