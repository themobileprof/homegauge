"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { api, outcomeLabel } from "@/lib/api";
import {
  STAGES,
  actionLabel,
  deriveFile,
  isWorkingStatus,
  type CaseFile,
  type DocItem,
} from "@/lib/advisor-file";
import { ADVISOR_STATUSES, documentStatusLabel, fileRef, statusLabel } from "@/lib/cases";
import { useCountry } from "@/lib/country";
import { formatRate } from "@/lib/rates";

export default function AdvisorCasePage() {
  const { money } = useCountry();
  const { id } = useParams<{ id: string }>();
  const [data, setData] = useState<CaseFile | null>(null);
  const [status, setStatus] = useState("");
  const [nextAction, setNextAction] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState("");
  const [sendBackId, setSendBackId] = useState<string | null>(null);
  const [sendBackNote, setSendBackNote] = useState("");
  const [noteVis, setNoteVis] = useState<"internal" | "customer">("internal");
  const [extraAsk, setExtraAsk] = useState("");
  const [stage, setStage] = useState<(typeof STAGES)[number]["id"]>("situation");

  async function load() {
    const d = await api<CaseFile>(`/api/v1/advisor/cases/${id}`);
    const suggestions = (d.suggestions || []).map((s) => ({
      ...s,
      payload: (typeof s.payload === "string" ? JSON.parse(s.payload) : s.payload) || {},
    }));
    setData({ ...d, documents: d.documents || [], notes: d.notes || [], suggestions });
    setStatus(d.case.status);
    setNextAction(d.case.next_action_text || "");
  }

  useEffect(() => {
    load().catch((e) => setError(e.message));
  }, [id]);

  const file = useMemo(() => (data ? deriveFile(data) : null), [data]);
  const working = data ? isWorkingStatus(data.case.status) : false;
  const buyer = data?.case.customer_name || data?.case.customer_email || "Buyer";

  function flash(ok: string) {
    setMessage(ok);
    setError("");
  }

  async function patchStatus(nextStatus: string, nextText: string) {
    await api(`/api/v1/advisor/cases/${id}/status`, {
      method: "PATCH",
      body: JSON.stringify({ status: nextStatus, next_action_text: nextText }),
    });
  }

  async function saveAdvance(e: FormEvent) {
    e.preventDefault();
    setBusy("advance");
    try {
      await patchStatus(status, nextAction);
      flash("File status saved.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not update status.");
    } finally {
      setBusy("");
    }
  }

  async function markReady() {
    if (!file?.canMarkReady) return;
    setBusy("ready");
    try {
      await patchStatus("READY_FOR_SUBMISSION", file.readyCopy);
      flash("Handed to admin. They will record the top-level outcome.");
      await load();
      setStage("advance");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not mark ready.");
    } finally {
      setBusy("");
    }
  }

  async function addNote(e: FormEvent) {
    e.preventDefault();
    setBusy("note");
    try {
      await api(`/api/v1/advisor/cases/${id}/notes`, {
        method: "POST",
        body: JSON.stringify({ body: note, visibility: noteVis }),
      });
      setNote("");
      flash("Note added.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not add note.");
    } finally {
      setBusy("");
    }
  }

  async function resolveSuggestion(sid: string, decision: "approved" | "rejected") {
    setBusy(sid);
    try {
      await api(`/api/v1/advisor/suggestions/${sid}/resolve`, {
        method: "POST",
        body: JSON.stringify({ status: decision }),
      });
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not update work item.");
    } finally {
      setBusy("");
    }
  }

  async function setProduct(productId: string) {
    setBusy("product");
    try {
      await api(`/api/v1/advisor/cases/${id}/product`, {
        method: "PATCH",
        body: JSON.stringify({ product_id: productId }),
      });
      flash("Product set on this file.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save product.");
    } finally {
      setBusy("");
    }
  }

  async function recordLender(nextStatus: string, nextText: string, customerNote?: string) {
    setBusy("lender");
    try {
      await patchStatus(nextStatus, nextText);
      if (customerNote) {
        await api(`/api/v1/advisor/cases/${id}/notes`, {
          method: "POST",
          body: JSON.stringify({ body: customerNote, visibility: "customer" }),
        });
      }
      flash("Lender update recorded.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not record lender update.");
    } finally {
      setBusy("");
    }
  }

  async function openDoc(doc: DocItem) {
    if (!doc.document_id) return;
    setBusy(doc.document_id);
    try {
      const d = await api<{ url: string }>(`/api/v1/advisor/documents/${doc.document_id}/download-url`);
      window.open(d.url, "_blank", "noopener,noreferrer");
      if (doc.status === "uploaded") {
        await api(`/api/v1/advisor/documents/${doc.document_id}/review`, {
          method: "POST",
          body: JSON.stringify({ decision: "under_review" }),
        });
        if (working && data?.case.status === "DOCUMENTS_PENDING") {
          await patchStatus("DOCUMENTS_UNDER_REVIEW", data.case.next_action_text || "Review uploaded documents.");
        }
        await load();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not open document.");
    } finally {
      setBusy("");
    }
  }

  async function reviewDoc(doc: DocItem, decision: "accepted" | "requires_replacement", notes?: string) {
    if (!doc.document_id) return;
    setBusy(doc.document_id + decision);
    try {
      await api(`/api/v1/advisor/documents/${doc.document_id}/review`, {
        method: "POST",
        body: JSON.stringify({ decision, notes: notes || "" }),
      });
      if (decision === "requires_replacement" && working) {
        await patchStatus(
          "ADDITIONAL_INFORMATION_REQUIRED",
          `Buyer to replace: ${doc.label}.`,
        );
      }
      setSendBackId(null);
      setSendBackNote("");
      flash(decision === "accepted" ? `${doc.label} accepted.` : `${doc.label} sent back.`);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not review document.");
    } finally {
      setBusy("");
    }
  }

  if (!data || !file) {
    return (
      <div className="advisor-desk min-h-screen px-5 py-10">
        <div className="mx-auto max-w-5xl">
          <Link href="/advisor" className="text-sm font-semibold text-[#8a6d28]">
            ← Working files
          </Link>
          {error ? <p className="mt-6 text-sm text-[color:var(--danger)]">{error}</p> : <p className="mt-8 text-sm text-muted">Opening file…</p>}
        </div>
      </div>
    );
  }

  const in_ = file.input;
  const currentStage = STAGES.find((s) => s.id === stage) || STAGES[0];

  return (
    <div className="advisor-desk min-h-[calc(100vh-3.5rem)] pb-28">
      <div className="mx-auto max-w-6xl px-5 py-8 md:px-8">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Link href="/advisor" className="text-sm font-semibold text-[#8a6d28]">
            ← Working files
          </Link>
          <p className="font-[family-name:var(--font-display)] text-sm tracking-wide text-[#8a6d28]">
            {fileRef(data.case.id)}
          </p>
        </div>

        <header className="file-jacket mt-5 overflow-hidden rounded-sm border border-[#c4a35a]/40">
          <div className="h-1.5 bg-gradient-to-r from-[#c4a35a] via-[#8a6d28] to-[#1f6b45]" />
          <div className="grid gap-6 px-6 py-6 md:grid-cols-[1fr_auto] md:px-8">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.18em] text-[#8a6d28]">Buyer file</p>
              <h1 className="mt-1 font-[family-name:var(--font-display)] text-3xl font-semibold tracking-tight md:text-4xl">
                {buyer}
              </h1>
              <p className="mt-1 text-sm text-muted">{data.case.customer_email}</p>
              <p className="mt-4 max-w-2xl text-sm leading-relaxed">
                <span className="font-semibold">Next: </span>
                {data.case.next_action_text || "Set the next action at the end of this file."}
              </p>
              {!working && (
                <p className="mt-3 text-sm text-muted">
                  Current status is <strong>{statusLabel(data.case.status)}</strong>. Admin records approved, rejected, completed, or cancelled — this is not a bank decision.
                </p>
              )}
            </div>
            <div className="flex flex-col items-start gap-3 md:items-end">
              <span className="rounded-sm bg-[#8a6d28] px-3 py-1 text-xs font-semibold uppercase tracking-wide text-white">
                {statusLabel(data.case.status)}
              </span>
              {file.assessment?.readiness && (
                <p className="font-[family-name:var(--font-display)] text-4xl font-semibold leading-none">
                  {file.assessment.readiness.total}
                  <span className="text-lg text-muted">/100</span>
                </p>
              )}
              {working && (
                <button
                  type="button"
                  disabled={!file.canMarkReady || busy === "ready"}
                  onClick={markReady}
                  className="rounded-sm bg-ink px-4 py-2.5 text-sm font-semibold text-paper disabled:opacity-40"
                >
                  {busy === "ready" ? "Handing up…" : "Ready for admin"}
                </button>
              )}
            </div>
          </div>
        </header>

        {(error || message) && (
          <p className={`mt-4 text-sm ${error ? "text-[color:var(--danger)]" : "text-leaf"}`}>{error || message}</p>
        )}

        <div className="mt-8 grid gap-8 lg:grid-cols-[13.5rem_1fr]">
          <nav className="lg:sticky lg:top-20 lg:self-start" aria-label="File stages">
            <ol className="flex gap-2 overflow-x-auto pb-2 lg:flex-col lg:gap-0 lg:overflow-visible lg:border-l lg:border-[#c4a35a]/40 lg:pb-0">
              {STAGES.map((s) => {
                const done = file.stageDone[s.id as keyof typeof file.stageDone];
                const active = stage === s.id;
                return (
                  <li key={s.id}>
                    <button
                      type="button"
                      onClick={() => setStage(s.id)}
                      className={`flex min-w-[9.5rem] items-baseline gap-3 px-3 py-2.5 text-left lg:min-w-0 lg:-ml-px lg:border-l-2 ${
                        active ? "border-[#8a6d28] bg-white/60 text-ink" : "border-transparent text-muted hover:text-ink"
                      }`}
                    >
                      <span className="font-[family-name:var(--font-display)] text-xs text-[#8a6d28]">{s.n}</span>
                      <span className="text-sm font-semibold">{s.title}</span>
                      {done && <span className="ml-auto text-[10px] font-semibold uppercase tracking-wide text-leaf">Done</span>}
                    </button>
                  </li>
                );
              })}
            </ol>
          </nav>

          <div className="file-jacket rounded-sm border border-[#c4a35a]/30 px-5 py-6 md:px-8 md:py-8">
            <p className="text-xs font-semibold uppercase tracking-[0.16em] text-[#8a6d28]">
              Stage {currentStage.n}
            </p>
            <h2 className="mt-1 font-[family-name:var(--font-display)] text-2xl font-semibold">{currentStage.title}</h2>

            {stage === "situation" && (
              <Situation money={money} file={file} buyer={buyer} />
            )}

            {stage === "documents" && (
              <Documents
                docs={data.documents || []}
                file={file}
                busy={busy}
                sendBackId={sendBackId}
                sendBackNote={sendBackNote}
                onOpen={openDoc}
                onAccept={(d) => reviewDoc(d, "accepted")}
                onSendBackStart={(d) => {
                  setSendBackId(d.document_id || null);
                  setSendBackNote("");
                }}
                onSendBackNote={setSendBackNote}
                onSendBackConfirm={(d) => reviewDoc(d, "requires_replacement", sendBackNote)}
                onSendBackCancel={() => setSendBackId(null)}
              />
            )}

            {stage === "products" && <Products money={money} file={file} />}

            {stage === "lender" && (
              <LenderDesk
                data={data}
                file={file}
                working={working}
                busy={busy}
                extraAsk={extraAsk}
                onExtraAsk={setExtraAsk}
                onChoose={setProduct}
                onRecord={recordLender}
              />
            )}

            {stage === "work" && (
              <WorkList
                suggestions={data.suggestions || []}
                busy={busy}
                onResolve={resolveSuggestion}
              />
            )}

            {stage === "notes" && (
              <Notes
                notes={data.notes || []}
                note={note}
                visibility={noteVis}
                busy={busy === "note"}
                onChange={setNote}
                onVisibility={setNoteVis}
                onSubmit={addNote}
              />
            )}

            {stage === "advance" && (
              <Advance
                working={working}
                status={status}
                nextAction={nextAction}
                suggested={file.suggestedStatus}
                blockers={file.blockers}
                canMarkReady={file.canMarkReady}
                busy={busy}
                current={data.case.status}
                onStatus={setStatus}
                onNextAction={setNextAction}
                onSubmit={saveAdvance}
                onMarkReady={markReady}
              />
            )}
          </div>
        </div>
      </div>

      {working && stage !== "advance" && (
        <div className="fixed inset-x-0 bottom-0 z-30 border-t border-[#c4a35a]/30 bg-[#f7f4ea]/95 px-5 py-3 backdrop-blur md:hidden">
          <button
            type="button"
            disabled={!file.canMarkReady || busy === "ready"}
            onClick={markReady}
            className="w-full rounded-sm bg-ink px-4 py-2.5 text-sm font-semibold text-paper disabled:opacity-40"
          >
            Ready for admin
          </button>
        </div>
      )}
    </div>
  );
}

function Situation({
  money,
  file,
  buyer,
}: {
  money: (n: number | null | undefined) => string;
  file: NonNullable<ReturnType<typeof deriveFile>>;
  buyer: string;
}) {
  const in_ = file.input;
  if (!file.assessment) {
    return (
      <div className="mt-6">
        <p className="text-sm leading-relaxed text-muted">
          {buyer} has not completed eligibility yet. Ask them to finish Check Eligibility so you can see income, deposit, and product fit on this file.
        </p>
      </div>
    );
  }

  const facts = [
    { k: "Employer", v: in_?.employer_name || in_?.employment_type || "—" },
    { k: "Years in role", v: in_?.years_employed != null ? String(in_.years_employed) : "—" },
    { k: "Monthly net income", v: money(in_?.monthly_net_income) },
    { k: "Other income", v: money(in_?.other_monthly_income) },
    { k: "Expenses", v: money(in_?.monthly_expenses) },
    { k: "Existing debt", v: money(in_?.existing_debt_repayments) },
    { k: "Deposit", v: money(in_?.available_deposit) },
    { k: "Property in mind", v: money(in_?.desired_property_price) },
    { k: "Loan sought", v: money(in_?.desired_loan_amount) },
    { k: "Tenor", v: in_?.preferred_tenor_years ? `${in_.preferred_tenor_years} years` : "—" },
    { k: "Salary months", v: in_?.salary_months != null ? `${in_.salary_months}` : "—" },
    { k: "Salary domicile", v: in_?.willing_to_domicile_salary ? "Willing" : "Not confirmed" },
  ];

  return (
    <div className="mt-6">
      <p className="text-sm leading-relaxed text-muted">
        Snapshot from the buyer’s eligibility answers. Use it to brief the conversation — it is not a credit score or a lender decision.
      </p>
      {in_?.salary_months != null && in_.salary_months < 6 && (
        <p className="mt-4 rounded-sm border border-[#c4a35a]/40 bg-[#c4a35a]/10 px-4 py-3 text-sm">
          Only {in_.salary_months} salary month{in_.salary_months === 1 ? "" : "s"} declared. HomeGauge underwriting is built around ~6 months of salary-account evidence.
        </p>
      )}
      <dl className="mt-6 grid gap-px overflow-hidden rounded-sm border border-[color:var(--line)] bg-[color:var(--line)] sm:grid-cols-2">
        {facts.map((f) => (
          <div key={f.k} className="bg-[#fffcf4] px-4 py-3">
            <dt className="text-[11px] font-semibold uppercase tracking-wide text-muted">{f.k}</dt>
            <dd className="mt-0.5 text-sm font-medium">{f.v}</dd>
          </div>
        ))}
      </dl>
      {file.assessment.readiness && (
        <div className="mt-8">
          <p className="text-sm font-semibold">Mortgage readiness</p>
          <p className="mt-2 text-sm leading-relaxed text-muted">{file.assessment.readiness.narrative}</p>
          <ul className="mt-4 space-y-3">
            {file.assessment.readiness.components.map((c) => (
              <li key={c.key}>
                <div className="mb-1 flex justify-between text-sm">
                  <span>{c.label}</span>
                  <span className="text-muted">
                    {c.score}/{c.max}
                  </span>
                </div>
                <div className="h-1.5 overflow-hidden rounded-full bg-paper-2">
                  <div className="h-full bg-[#8a6d28]" style={{ width: `${(c.score / c.max) * 100}%` }} />
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

function Documents({
  docs,
  file,
  busy,
  sendBackId,
  sendBackNote,
  onOpen,
  onAccept,
  onSendBackStart,
  onSendBackNote,
  onSendBackConfirm,
  onSendBackCancel,
}: {
  docs: DocItem[];
  file: NonNullable<ReturnType<typeof deriveFile>>;
  busy: string;
  sendBackId: string | null;
  sendBackNote: string;
  onOpen: (d: DocItem) => void;
  onAccept: (d: DocItem) => void;
  onSendBackStart: (d: DocItem) => void;
  onSendBackNote: (v: string) => void;
  onSendBackConfirm: (d: DocItem) => void;
  onSendBackCancel: () => void;
}) {
  return (
    <div className="mt-6">
      <p className="text-sm leading-relaxed text-muted">
        Accept what is clear. Send back anything incomplete with a reason the buyer can act on. Opening a new upload marks it in review.
      </p>
      <p className="mt-3 text-sm font-semibold text-[#8a6d28]">
        {file.acceptedRequired.length} of {file.required.length || docs.length} required accepted
        {file.pendingReview.length ? ` · ${file.pendingReview.length} to review` : ""}
      </p>
      <ul className="mt-5 divide-y divide-[color:var(--line)] border-y border-[color:var(--line)]">
        {docs.map((doc) => {
          const canAct = Boolean(doc.document_id) && (doc.status === "uploaded" || doc.status === "under_review");
          return (
            <li key={doc.document_type_code} className="py-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">{doc.category}</p>
                  <p className="font-semibold">
                    {doc.label}
                    {!doc.required && <span className="ml-2 text-xs font-normal text-muted">optional</span>}
                  </p>
                  {doc.instructions && <p className="mt-1 text-sm text-muted">{doc.instructions}</p>}
                  {doc.review_notes && (doc.status === "requires_replacement" || doc.status === "rejected") && (
                    <p className="mt-2 text-sm">{doc.review_notes}</p>
                  )}
                </div>
                <span className={`rounded-sm px-2 py-1 text-[11px] font-semibold uppercase tracking-wide ${docTone(doc.status)}`}>
                  {documentStatusLabel(doc.status)}
                </span>
              </div>
              {doc.document_id && (
                <div className="mt-3 flex flex-wrap gap-2">
                  <button
                    type="button"
                    onClick={() => onOpen(doc)}
                    disabled={busy === doc.document_id}
                    className="rounded-sm border border-[color:var(--line)] px-3 py-1.5 text-xs font-semibold"
                  >
                    Open
                  </button>
                  {canAct && (
                    <>
                      <button
                        type="button"
                        onClick={() => onAccept(doc)}
                        disabled={busy.startsWith(doc.document_id)}
                        className="rounded-sm bg-leaf px-3 py-1.5 text-xs font-semibold text-white"
                      >
                        Accept
                      </button>
                      <button
                        type="button"
                        onClick={() => onSendBackStart(doc)}
                        className="rounded-sm bg-paper-2 px-3 py-1.5 text-xs font-semibold"
                      >
                        Send back
                      </button>
                    </>
                  )}
                </div>
              )}
              {sendBackId === doc.document_id && (
                <div className="mt-3 rounded-sm border border-[color:var(--line)] bg-white/70 p-3">
                  <label className="text-xs font-semibold">What should the buyer fix?</label>
                  <textarea
                    value={sendBackNote}
                    onChange={(e) => onSendBackNote(e.target.value)}
                    rows={2}
                    className="mt-1 w-full rounded-sm border border-[color:var(--line)] px-3 py-2 text-sm"
                    placeholder="e.g. Last two months missing from the salary statements."
                  />
                  <div className="mt-2 flex gap-2">
                    <button
                      type="button"
                      onClick={() => onSendBackConfirm(doc)}
                      className="rounded-sm bg-ink px-3 py-1.5 text-xs font-semibold text-paper"
                    >
                      Send back
                    </button>
                    <button type="button" onClick={onSendBackCancel} className="text-xs font-semibold text-muted">
                      Cancel
                    </button>
                  </div>
                </div>
              )}
            </li>
          );
        })}
      </ul>
      {docs.length === 0 && <p className="mt-4 text-sm text-muted">No checklist on this file yet.</p>}
    </div>
  );
}

function docTone(status: string) {
  if (status === "accepted") return "bg-leaf/10 text-leaf-deep";
  if (status === "uploaded" || status === "under_review") return "bg-[#c4a35a]/15 text-[#8a6d28]";
  if (status === "rejected" || status === "requires_replacement") return "bg-[color:var(--danger)]/10 text-[color:var(--danger)]";
  return "bg-paper-2 text-muted";
}

function Products({
  money,
  file,
}: {
  money: (n: number | null | undefined) => string;
  file: NonNullable<ReturnType<typeof deriveFile>>;
}) {
  if (!file.assessment) {
    return <p className="mt-6 text-sm text-muted">Complete eligibility first — product fit is generated from that assessment.</p>;
  }
  const ranked = [...file.results].sort((a, b) => rankOutcome(a.outcome) - rankOutcome(b.outcome));
  return (
    <div className="mt-6">
      <p className="text-sm leading-relaxed text-muted">
        How stated product rules look against this buyer. A “likely” result is still not an approval — lenders decide.
      </p>
      {file.assessment.best_fit_why && (
        <p className="mt-4 rounded-sm border border-[#c4a35a]/35 bg-[#c4a35a]/10 px-4 py-3 text-sm leading-relaxed">
          <span className="font-semibold">Best fit. </span>
          {file.assessment.best_fit_why}
        </p>
      )}
      <ul className="mt-5 space-y-3">
        {ranked.map((r) => (
          <li
            key={r.product_id}
            className={`rounded-sm border px-4 py-4 ${
              r.product_id === file.assessment?.best_fit_product_id ? "border-[#8a6d28] bg-white/80" : "border-[color:var(--line)] bg-white/50"
            }`}
          >
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <Link href={`/mortgages/${r.product_id}`} className="font-semibold text-leaf hover:underline">
                  {r.product_name}
                </Link>
                <p className="text-sm text-muted">{r.lender_name}</p>
              </div>
              <span className="text-xs font-semibold uppercase tracking-wide text-[#8a6d28]">{outcomeLabel(r.outcome)}</span>
            </div>
            <p className="mt-2 text-sm leading-relaxed text-muted">{r.explanation}</p>
            <p className="mt-2 text-xs text-muted">
              {formatRate(r)}
              {r.estimated_monthly_repayment != null ? ` · est. ${money(r.estimated_monthly_repayment)} / month` : ""}
              {r.min_equity_pct != null ? ` · min equity ${r.min_equity_pct}%` : ""}
            </p>
            <Link href={`/mortgages/${r.product_id}`} className="mt-3 inline-block text-sm font-semibold text-leaf hover:underline">
              View product details →
            </Link>
          </li>
        ))}
      </ul>
      {ranked.length === 0 && <p className="mt-4 text-sm text-muted">No product outcomes on this assessment.</p>}
    </div>
  );
}

function rankOutcome(o: string) {
  if (o === "likely_eligible") return 0;
  if (o === "potentially_eligible") return 1;
  if (o === "may_require_review") return 2;
  if (o === "more_info_required") return 3;
  return 4;
}

function WorkList({
  suggestions,
  busy,
  onResolve,
}: {
  suggestions: CaseFile["suggestions"];
  busy: string;
  onResolve: (id: string, d: "approved" | "rejected") => void;
}) {
  const pending = suggestions.filter((s) => s.status === "pending");
  const done = suggestions.filter((s) => s.status !== "pending");
  return (
    <div className="mt-6">
      <p className="text-sm leading-relaxed text-muted">
        Opening checklist for this file — built from assessment, readiness, and documents, not a chatbot. Tick what you have done; skip what does not apply.
      </p>
      {pending.length === 0 && <p className="mt-4 text-sm text-muted">Nothing outstanding on the work list.</p>}
      <ul className="mt-5 space-y-4">
        {pending.map((s) => (
          <li key={s.id} className="rounded-sm border border-[color:var(--line)] bg-white/70 p-4">
            {s.payload?.priority && (
              <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a6d28]">{s.payload.priority} priority</p>
            )}
            <p className="mt-1 text-sm leading-relaxed">{s.payload?.message || "Review this file."}</p>
            {(s.payload?.actions || []).length > 0 && (
              <ol className="mt-3 list-decimal space-y-1 pl-5 text-sm">
                {s.payload.actions!.map((a) => (
                  <li key={a}>{actionLabel(a)}</li>
                ))}
              </ol>
            )}
            <div className="mt-4 flex gap-2">
              <button
                type="button"
                disabled={busy === s.id}
                onClick={() => onResolve(s.id, "approved")}
                className="rounded-sm bg-leaf px-3 py-1.5 text-xs font-semibold text-white"
              >
                Done
              </button>
              <button
                type="button"
                disabled={busy === s.id}
                onClick={() => onResolve(s.id, "rejected")}
                className="rounded-sm bg-paper-2 px-3 py-1.5 text-xs font-semibold"
              >
                Skip
              </button>
            </div>
          </li>
        ))}
      </ul>
      {done.length > 0 && (
        <p className="mt-6 text-xs text-muted">
          {done.length} earlier item{done.length === 1 ? "" : "s"} already closed.
        </p>
      )}
    </div>
  );
}

function Notes({
  notes,
  note,
  visibility,
  busy,
  onChange,
  onVisibility,
  onSubmit,
}: {
  notes: CaseFile["notes"];
  note: string;
  visibility: "internal" | "customer";
  busy: boolean;
  onChange: (v: string) => void;
  onVisibility: (v: "internal" | "customer") => void;
  onSubmit: (e: FormEvent) => void;
}) {
  return (
    <div className="mt-6">
      <p className="text-sm leading-relaxed text-muted">
        Internal notes stay on the desk. Buyer-visible notes appear on their journey — use them for what they must do next.
      </p>
      <form onSubmit={onSubmit} className="mt-4 space-y-3">
        <textarea
          value={note}
          onChange={(e) => onChange(e.target.value)}
          required
          rows={4}
          className="w-full rounded-sm border border-[color:var(--line)] bg-white px-3 py-2 text-sm"
          placeholder="What happened on this file today…"
        />
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={visibility === "customer"}
            onChange={(e) => onVisibility(e.target.checked ? "customer" : "internal")}
          />
          Visible to the buyer
        </label>
        <button type="submit" disabled={busy} className="rounded-sm bg-ink px-4 py-2 text-sm font-semibold text-paper">
          {busy ? "Saving…" : "Add note"}
        </button>
      </form>
      <ol className="mt-6 space-y-4 border-l border-[#c4a35a]/40 pl-5">
        {notes.map((n) => (
          <li key={n.id}>
            <p className="text-xs text-muted">
              {n.visibility === "customer" ? "Buyer can see" : "Internal"} · {n.author_email} · {new Date(n.created_at).toLocaleString("en-NG")}
            </p>
            <p className="mt-1 text-sm leading-relaxed">{n.body}</p>
          </li>
        ))}
      </ol>
      {notes.length === 0 && <p className="mt-4 text-sm text-muted">No notes yet.</p>}
    </div>
  );
}

function Advance({
  working,
  status,
  nextAction,
  suggested,
  blockers,
  canMarkReady,
  busy,
  current,
  onStatus,
  onNextAction,
  onSubmit,
  onMarkReady,
}: {
  working: boolean;
  status: string;
  nextAction: string;
  suggested: string;
  blockers: string[];
  canMarkReady: boolean;
  busy: string;
  current: string;
  onStatus: (v: string) => void;
  onNextAction: (v: string) => void;
  onSubmit: (e: FormEvent) => void;
  onMarkReady: () => void;
}) {
  if (!working) {
    return (
      <div className="mt-6">
        <p className="text-sm leading-relaxed text-muted">
          This file is at <strong>{statusLabel(current)}</strong>. Admin owns approved, rejected, completed, and cancelled. Never treat those as a bank approval.
        </p>
      </div>
    );
  }

  return (
    <div className="mt-6">
      <p className="text-sm leading-relaxed text-muted">
        Move the working status so the next person knows what to do. Handing up to admin means the file is document-ready and product-fit — not that a lender has approved it.
      </p>

      {blockers.length > 0 && (
        <div className="mt-4 rounded-sm border border-[color:var(--line)] bg-white/70 px-4 py-3">
          <p className="text-xs font-semibold uppercase tracking-wide text-[#8a6d28]">Before you hand this up</p>
          <ul className="mt-2 list-disc space-y-1 pl-5 text-sm">
            {blockers.map((b) => (
              <li key={b}>{b}</li>
            ))}
          </ul>
        </div>
      )}

      <form onSubmit={onSubmit} className="mt-6 space-y-4">
        <label className="block text-sm">
          <span className="mb-1 block font-medium">Working status</span>
          <select
            value={status}
            onChange={(e) => onStatus(e.target.value)}
            className="w-full max-w-sm rounded-sm border border-[color:var(--line)] bg-white px-3 py-2"
          >
            {ADVISOR_STATUSES.map((s) => (
              <option key={s} value={s}>
                {statusLabel(s)}
                {s === suggested ? " — suggested" : ""}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm">
          <span className="mb-1 block font-medium">Next action</span>
          <textarea
            value={nextAction}
            onChange={(e) => onNextAction(e.target.value)}
            rows={3}
            className="w-full rounded-sm border border-[color:var(--line)] bg-white px-3 py-2 text-sm"
            placeholder="One sentence the next person can act on."
          />
        </label>
        <div className="flex flex-wrap gap-3">
          <button type="submit" disabled={busy === "advance"} className="rounded-sm border border-[color:var(--line)] px-4 py-2 text-sm font-semibold">
            {busy === "advance" ? "Saving…" : "Save status"}
          </button>
          <button
            type="button"
            disabled={!canMarkReady || busy === "ready"}
            onClick={onMarkReady}
            className="rounded-sm bg-ink px-4 py-2 text-sm font-semibold text-paper disabled:opacity-40"
          >
            {busy === "ready" ? "Handing up…" : "Ready for admin"}
          </button>
        </div>
      </form>
    </div>
  );
}

function LenderDesk({
  data,
  file,
  working,
  busy,
  extraAsk,
  onExtraAsk,
  onChoose,
  onRecord,
}: {
  data: CaseFile;
  file: NonNullable<ReturnType<typeof deriveFile>>;
  working: boolean;
  busy: string;
  extraAsk: string;
  onExtraAsk: (v: string) => void;
  onChoose: (id: string) => void;
  onRecord: (status: string, next: string, note?: string) => void;
}) {
  const c = data.case;
  const hasProduct = Boolean(c.preferred_product_id);
  const hasPortal = Boolean(c.lender_has_account);
  const options = file.likely.length ? file.likely : file.results;

  return (
    <div className="mt-6">
      <p className="text-sm leading-relaxed text-muted">
        Point this file at a product. If that lender has a HomeGauge account, they update their own pipeline after you submit.
        If they do not, you are the liaison — record what they tell you here so the buyer can see it.
      </p>

      {hasProduct ? (
        <div className="mt-4 rounded-sm border border-[#c4a35a]/35 bg-[#c4a35a]/10 px-4 py-3 text-sm">
          <p className="font-semibold">{c.preferred_product_name}</p>
          <p className="mt-1 text-muted">{c.lender_name}</p>
          <p className="mt-2">
            {hasPortal
              ? "This lender has a portal. After you submit, they can mark review or ask for more information themselves. You can still record a phone or email update."
              : "This lender has not taken up a HomeGauge account. You record their updates on this file."}
          </p>
        </div>
      ) : (
        <p className="mt-4 text-sm">No product on this file yet. Choose from the buyer’s likely fits:</p>
      )}

      <ul className="mt-4 space-y-2">
        {options.map((r) => (
          <li key={r.product_id} className="flex flex-wrap items-center justify-between gap-2 rounded-sm border border-[color:var(--line)] bg-white/70 px-4 py-3">
            <div>
              <p className="font-semibold">{r.product_name}</p>
              <p className="text-sm text-muted">{r.lender_name}</p>
            </div>
            <button
              type="button"
              disabled={busy === "product" || c.preferred_product_id === r.product_id}
              onClick={() => onChoose(r.product_id)}
              className="text-sm font-semibold text-[#8a6d28] disabled:opacity-40"
            >
              {c.preferred_product_id === r.product_id ? "Selected" : "Use this product"}
            </button>
          </li>
        ))}
      </ul>
      {options.length === 0 && <p className="mt-3 text-sm text-muted">No product outcomes yet — finish eligibility first.</p>}

      {working && hasProduct && (
        <div className="mt-8 space-y-3">
          <p className="text-xs font-semibold uppercase tracking-wide text-[#8a6d28]">Record a lender update</p>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              disabled={busy === "lender"}
              onClick={() =>
                onRecord(
                  "SUBMITTED_TO_LENDER",
                  hasPortal
                    ? `File submitted to ${c.lender_name}. Waiting for their portal review.`
                    : `File submitted to ${c.lender_name} (no portal — advisor is the liaison).`,
                )
              }
              className="rounded-sm bg-ink px-4 py-2 text-sm font-semibold text-paper"
            >
              Submitted to lender
            </button>
            <button
              type="button"
              disabled={busy === "lender"}
              onClick={() => onRecord("LENDER_REVIEW", `${c.lender_name} is reviewing the file.`)}
              className="rounded-sm border border-[color:var(--line)] px-4 py-2 text-sm font-semibold"
            >
              Lender is reviewing
            </button>
          </div>
          <form
            className="space-y-3"
            onSubmit={(e) => {
              e.preventDefault();
              onRecord("ADDITIONAL_INFORMATION_REQUIRED", extraAsk, extraAsk);
            }}
          >
            <label className="block text-sm">
              <span className="mb-1 block font-medium">Lender asked for more (buyer will see this)</span>
              <textarea
                value={extraAsk}
                onChange={(e) => onExtraAsk(e.target.value)}
                rows={3}
                required
                className="w-full rounded-sm border border-[color:var(--line)] bg-white px-3 py-2 text-sm"
                placeholder="e.g. Stanbic asked for a more recent employment letter."
              />
            </label>
            <button type="submit" disabled={busy === "lender"} className="rounded-sm bg-paper-2 px-4 py-2 text-sm font-semibold">
              Record extra information needed
            </button>
          </form>
        </div>
      )}
    </div>
  );
}
