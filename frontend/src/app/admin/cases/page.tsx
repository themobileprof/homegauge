"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { ADMIN_STATUSES, advisorName, statusLabel } from "@/lib/cases";

type CaseRow = {
  id: string;
  customer_name: string;
  customer_email: string;
  status: string;
  assigned_advisor_id?: string | null;
  assigned_advisor_name?: string;
  assigned_advisor_email?: string;
  next_action_text: string;
  updated_at: string;
};

type Advisor = { id: string; email: string; full_name: string; open_cases: number };

export default function AdminCasesPage() {
  const [cases, setCases] = useState<CaseRow[]>([]);
  const [advisors, setAdvisors] = useState<Advisor[]>([]);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [assignedFilter, setAssignedFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [advisorId, setAdvisorId] = useState("");
  const [status, setStatus] = useState("");
  const [nextAction, setNextAction] = useState("");

  async function load() {
    const [c, a] = await Promise.all([
      api<{ cases: CaseRow[] }>("/api/v1/admin/cases"),
      api<{ advisors: Advisor[] }>("/api/v1/admin/advisors"),
    ]);
    setCases(c.cases || []);
    setAdvisors(a.advisors || []);
  }

  useEffect(() => {
    load()
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return cases.filter((c) => {
      if (assignedFilter === "unassigned" && c.assigned_advisor_id) return false;
      if (assignedFilter === "assigned" && !c.assigned_advisor_id) return false;
      if (assignedFilter !== "all" && assignedFilter !== "unassigned" && assignedFilter !== "assigned" && c.assigned_advisor_id !== assignedFilter) {
        return false;
      }
      if (statusFilter !== "all" && c.status !== statusFilter) return false;
      if (!q) return true;
      return (
        (c.customer_name || "").toLowerCase().includes(q) ||
        c.customer_email.toLowerCase().includes(q) ||
        advisorName(c).toLowerCase().includes(q)
      );
    });
  }, [cases, query, assignedFilter, statusFilter]);

  const editing = editingId ? cases.find((c) => c.id === editingId) : null;

  function openEdit(c: CaseRow) {
    setError("");
    setMessage("");
    setEditingId(c.id);
    setAdvisorId(c.assigned_advisor_id || "");
    setStatus(c.status);
    setNextAction(c.next_action_text || "");
  }

  async function onAssign(e: FormEvent) {
    e.preventDefault();
    if (!editing || !advisorId) {
      setError("Choose an advisor.");
      return;
    }
    setBusy("assign");
    setError("");
    try {
      const d = await api<{ case: CaseRow }>(`/api/v1/admin/cases/${editing.id}/assign`, {
        method: "POST",
        body: JSON.stringify({ advisor_id: advisorId }),
      });
      setCases((prev) => prev.map((c) => (c.id === d.case.id ? { ...c, ...d.case } : c)));
      setMessage(`Assigned to ${advisorName(d.case)}.`);
      setEditingId(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not assign.");
    } finally {
      setBusy(null);
    }
  }

  async function onStatus(e: FormEvent) {
    e.preventDefault();
    if (!editing) return;
    setBusy("status");
    setError("");
    try {
      const d = await api<{ case: CaseRow }>(`/api/v1/admin/cases/${editing.id}/status`, {
        method: "PATCH",
        body: JSON.stringify({ status, next_action_text: nextAction }),
      });
      setCases((prev) => prev.map((c) => (c.id === d.case.id ? { ...c, ...d.case } : c)));
      setMessage("Status updated.");
      setEditingId(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not update status.");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="mx-auto max-w-6xl px-5 py-10 md:px-8">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-gold">Admin console</p>
      <h1 className="mt-2 font-[family-name:var(--font-display)] text-3xl font-semibold">Case assignment</h1>
      <p className="mt-2 max-w-2xl text-sm text-muted">
        Assign and reassign files to advisors, and set case status. Advisors handle notes, documents, and working the file — not this console.
      </p>
      {error && <p className="mt-4 text-sm text-[color:var(--danger)]">{error}</p>}
      {message && <p className="mt-4 text-sm text-leaf">{message}</p>}

      {editing && (
        <div className="mt-8 grid gap-4 rounded-xl border border-[color:var(--line)] bg-white/80 p-6 lg:grid-cols-2">
          <div className="lg:col-span-2 flex items-start justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold">{editing.customer_name || editing.customer_email}</h2>
              <p className="text-sm text-muted">{editing.customer_email}</p>
            </div>
            <button type="button" onClick={() => setEditingId(null)} className="text-sm font-semibold text-muted hover:text-ink">
              Close
            </button>
          </div>
          <form onSubmit={onAssign} className="space-y-3">
            <p className="text-sm font-semibold">Assign advisor</p>
            <select required value={advisorId} onChange={(e) => setAdvisorId(e.target.value)} className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5">
              <option value="">Select advisor</option>
              {advisors.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.full_name || a.email} ({a.open_cases} open)
                </option>
              ))}
            </select>
            <button type="submit" disabled={busy !== null} className="rounded-md bg-ink px-4 py-2.5 text-sm font-semibold text-paper disabled:opacity-60">
              {busy === "assign" ? "Assigning…" : editing.assigned_advisor_id ? "Reassign" : "Assign"}
            </button>
          </form>
          <form onSubmit={onStatus} className="space-y-3">
            <p className="text-sm font-semibold">Case status</p>
            <select value={status} onChange={(e) => setStatus(e.target.value)} className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5">
              {ADMIN_STATUSES.map((s) => (
                <option key={s} value={s}>
                  {statusLabel(s)}
                </option>
              ))}
            </select>
            <input
              value={nextAction}
              onChange={(e) => setNextAction(e.target.value)}
              placeholder="Next action (shown on the advisor queue)"
              className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 text-sm"
            />
            <button type="submit" disabled={busy !== null} className="rounded-md bg-leaf px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-60">
              {busy === "status" ? "Saving…" : "Update status"}
            </button>
          </form>
        </div>
      )}

      <div className="mt-8 flex flex-wrap gap-3">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search buyer or advisor"
          className="min-w-[16rem] flex-1 rounded-md border border-[color:var(--line)] bg-white px-3 py-2 text-sm outline-none ring-leaf focus:ring-2"
        />
        <select value={assignedFilter} onChange={(e) => setAssignedFilter(e.target.value)} className="rounded-md border border-[color:var(--line)] bg-white px-3 py-2 text-sm">
          <option value="all">All assignment</option>
          <option value="unassigned">Unassigned</option>
          <option value="assigned">Assigned</option>
          {advisors.map((a) => (
            <option key={a.id} value={a.id}>
              {a.full_name || a.email}
            </option>
          ))}
        </select>
        <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className="rounded-md border border-[color:var(--line)] bg-white px-3 py-2 text-sm">
          <option value="all">All statuses</option>
          {ADMIN_STATUSES.map((s) => (
            <option key={s} value={s}>
              {statusLabel(s)}
            </option>
          ))}
        </select>
      </div>

      <div className="mt-4 overflow-x-auto rounded-xl border border-[color:var(--line)] bg-white/80">
        <table className="w-full min-w-[860px] text-left text-sm">
          <thead className="border-b border-[color:var(--line)] bg-paper-2/60">
            <tr>
              <th className="px-4 py-3 font-semibold">Homebuyer</th>
              <th className="px-4 py-3 font-semibold">Advisor</th>
              <th className="px-4 py-3 font-semibold">Status</th>
              <th className="px-4 py-3 font-semibold">Updated</th>
              <th className="px-4 py-3 font-semibold"> </th>
            </tr>
          </thead>
          <tbody>
            {visible.map((c) => (
              <tr key={c.id} className="border-b border-[color:var(--line)] last:border-0">
                <td className="px-4 py-3">
                  <div className="font-medium">{c.customer_name || c.customer_email}</div>
                  <div className="text-xs text-muted">{c.customer_email}</div>
                </td>
                <td className="px-4 py-3">
                  {c.assigned_advisor_id ? advisorName(c) : <span className="font-semibold text-[#8a6d28]">Unassigned</span>}
                </td>
                <td className="px-4 py-3">{statusLabel(c.status)}</td>
                <td className="px-4 py-3 text-muted">{new Date(c.updated_at).toLocaleDateString("en-NG")}</td>
                <td className="px-4 py-3 text-right">
                  <button type="button" onClick={() => openEdit(c)} className="font-semibold text-leaf hover:underline">
                    Manage
                  </button>
                </td>
              </tr>
            ))}
            {!loading && visible.length === 0 && (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-muted">
                  No cases match that filter.
                </td>
              </tr>
            )}
            {loading && (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-muted">
                  Loading…
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
