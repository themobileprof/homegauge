"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { advisorName, statusLabel } from "@/lib/cases";

type Summary = {
  unassigned_open: number;
  open_cases: number;
  by_status: Record<string, number>;
};

type AdvisorRow = {
  id: string;
  email: string;
  full_name: string;
  open_cases: number;
  assigned_total: number;
  completed: number;
  approved: number;
  rejected: number;
};

type BuyerRow = {
  user_id: string;
  email: string;
  full_name: string;
  case_id?: string | null;
  status?: string;
  assigned_advisor_name?: string;
  assigned_advisor_email?: string;
  updated_at?: string | null;
};

export default function AdminReportsPage() {
  const [advisors, setAdvisors] = useState<AdvisorRow[]>([]);
  const [buyers, setBuyers] = useState<BuyerRow[]>([]);
  const [summary, setSummary] = useState<Summary | null>(null);
  const [error, setError] = useState("");
  const [tab, setTab] = useState<"advisors" | "buyers">("advisors");

  useEffect(() => {
    Promise.all([
      api<{ advisors: AdvisorRow[]; summary: Summary }>("/api/v1/admin/reports/advisors"),
      api<{ buyers: BuyerRow[]; summary: Summary }>("/api/v1/admin/reports/buyers"),
    ])
      .then(([a, b]) => {
        setAdvisors(a.advisors || []);
        setBuyers(b.buyers || []);
        setSummary(a.summary || b.summary);
      })
      .catch((e) => setError(e.message));
  }, []);

  return (
    <div className="mx-auto max-w-6xl px-5 py-10 md:px-8">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-gold">Admin console</p>
      <h1 className="mt-2 font-[family-name:var(--font-display)] text-3xl font-semibold">Reports</h1>
      <p className="mt-2 max-w-2xl text-sm text-muted">
        Workload by advisor and the latest file for each homebuyer. This is an operations view — not a lender decision.
      </p>
      {error && <p className="mt-4 text-sm text-[color:var(--danger)]">{error}</p>}

      <div className="mt-8 grid gap-4 sm:grid-cols-3">
        <Card label="Open cases" value={summary?.open_cases} href="/admin/cases" />
        <Card label="Unassigned" value={summary?.unassigned_open} href="/admin/cases" />
        <Card label="Ready for approval" value={summary?.by_status?.READY_FOR_SUBMISSION} href="/admin/approvals" />
      </div>

      <div className="mt-8 flex gap-2">
        <button
          type="button"
          onClick={() => setTab("advisors")}
          className={`rounded-md px-4 py-2 text-sm font-semibold ${tab === "advisors" ? "bg-ink text-paper" : "bg-paper-2"}`}
        >
          Advisors
        </button>
        <button
          type="button"
          onClick={() => setTab("buyers")}
          className={`rounded-md px-4 py-2 text-sm font-semibold ${tab === "buyers" ? "bg-ink text-paper" : "bg-paper-2"}`}
        >
          Homebuyers
        </button>
      </div>

      {tab === "advisors" && (
        <div className="mt-4 overflow-x-auto rounded-xl border border-[color:var(--line)] bg-white/80">
          <table className="w-full min-w-[720px] text-left text-sm">
            <thead className="border-b border-[color:var(--line)] bg-paper-2/60">
              <tr>
                <th className="px-4 py-3 font-semibold">Advisor</th>
                <th className="px-4 py-3 font-semibold">Open</th>
                <th className="px-4 py-3 font-semibold">Assigned (all time)</th>
                <th className="px-4 py-3 font-semibold">Approved</th>
                <th className="px-4 py-3 font-semibold">Completed</th>
                <th className="px-4 py-3 font-semibold">Rejected</th>
              </tr>
            </thead>
            <tbody>
              {advisors.map((a) => (
                <tr key={a.id} className="border-b border-[color:var(--line)] last:border-0">
                  <td className="px-4 py-3">
                    <div className="font-medium">{a.full_name || a.email}</div>
                    <div className="text-xs text-muted">{a.email}</div>
                  </td>
                  <td className="px-4 py-3 font-semibold">{a.open_cases}</td>
                  <td className="px-4 py-3">{a.assigned_total}</td>
                  <td className="px-4 py-3">{a.approved}</td>
                  <td className="px-4 py-3">{a.completed}</td>
                  <td className="px-4 py-3">{a.rejected}</td>
                </tr>
              ))}
              {advisors.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-muted">
                    No advisors yet.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {tab === "buyers" && (
        <div className="mt-4 overflow-x-auto rounded-xl border border-[color:var(--line)] bg-white/80">
          <table className="w-full min-w-[720px] text-left text-sm">
            <thead className="border-b border-[color:var(--line)] bg-paper-2/60">
              <tr>
                <th className="px-4 py-3 font-semibold">Homebuyer</th>
                <th className="px-4 py-3 font-semibold">Latest case</th>
                <th className="px-4 py-3 font-semibold">Advisor</th>
                <th className="px-4 py-3 font-semibold">Updated</th>
              </tr>
            </thead>
            <tbody>
              {buyers.map((b) => (
                <tr key={b.user_id} className="border-b border-[color:var(--line)] last:border-0">
                  <td className="px-4 py-3">
                    <div className="font-medium">{b.full_name || b.email}</div>
                    <div className="text-xs text-muted">{b.email}</div>
                  </td>
                  <td className="px-4 py-3">{b.case_id ? statusLabel(b.status) : "No case"}</td>
                  <td className="px-4 py-3">
                    {b.case_id ? advisorName({ assigned_advisor_name: b.assigned_advisor_name, assigned_advisor_email: b.assigned_advisor_email }) : "—"}
                  </td>
                  <td className="px-4 py-3 text-muted">{b.updated_at ? new Date(b.updated_at).toLocaleDateString("en-NG") : "—"}</td>
                </tr>
              ))}
              {buyers.length === 0 && (
                <tr>
                  <td colSpan={4} className="px-4 py-8 text-center text-muted">
                    No homebuyers yet.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function Card({ label, value, href }: { label: string; value?: number; href: string }) {
  return (
    <Link href={href} className="rounded-xl border border-[color:var(--line)] bg-white/80 p-5 hover:border-leaf">
      <p className="text-xs font-semibold uppercase tracking-wide text-muted">{label}</p>
      <p className="mt-2 font-[family-name:var(--font-display)] text-3xl font-semibold">{value ?? "—"}</p>
    </Link>
  );
}
