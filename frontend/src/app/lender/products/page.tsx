"use client";

import Link from "next/link";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useCountry } from "@/lib/country";
import { formatRate } from "@/lib/rates";

type LenderMe = { lender: { id: string; name: string; country_code: string }; email: string };

type Product = {
  id: string;
  country_code: string;
  currency_code: string;
  lender_id: string;
  lender_name: string;
  name: string;
  description: string;
  mortgage_type: string;
  min_loan_amount: number | null;
  max_loan_amount: number | null;
  min_income: number | null;
  max_age: number | null;
  max_tenor_years: number | null;
  min_equity_pct: number | null;
  interest_rate: number | null;
  interest_rate_min: number | null;
  interest_rate_max: number | null;
  interest_rate_type: string;
  processing_fee: number | null;
  valuation_fee: number | null;
  legal_fee: number | null;
  status: string;
  source?: string | null;
  source_url?: string | null;
  verification_status: string;
};

const TYPES = [
  { value: "nhf", label: "NHF / housing scheme" },
  { value: "mreif", label: "MREIF" },
  { value: "commercial", label: "Commercial" },
  { value: "scheme", label: "Other scheme" },
  { value: "other", label: "Other" },
];

const emptyForm = {
  name: "",
  description: "",
  mortgage_type: "commercial",
  interest_rate: "",
  interest_rate_min: "",
  interest_rate_max: "",
  interest_rate_type: "fixed",
  min_loan_amount: "",
  max_loan_amount: "",
  min_income: "",
  min_equity_pct: "",
  max_tenor_years: "",
  max_age: "",
  processing_fee: "",
  valuation_fee: "",
  legal_fee: "",
  status: "active",
  source: "",
  source_url: "",
};

const fieldClass =
  "w-full rounded-sm border border-[#1f4d6b]/20 bg-white px-3 py-2.5 outline-none ring-[#1f4d6b] focus:ring-2";

function numOrNull(v: string): number | null {
  const t = v.trim();
  if (!t) return null;
  const n = Number(t);
  return Number.isFinite(n) ? n : null;
}

function strNum(v: number | null | undefined) {
  return v == null ? "" : String(v);
}

export default function LenderProductsPage() {
  const { money } = useCountry();
  const [me, setMe] = useState<LenderMe | null>(null);
  const [products, setProducts] = useState<Product[]>([]);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState(emptyForm);

  const creating = editingId === "new";
  const editing = editingId && editingId !== "new" ? products.find((p) => p.id === editingId) : null;

  async function load() {
    const [m, p] = await Promise.all([
      api<LenderMe>("/api/v1/lender/me"),
      api<{ products: Product[] }>("/api/v1/lender/products"),
    ]);
    setMe(m);
    setProducts(p.products || []);
  }

  useEffect(() => {
    load()
      .catch((e) => setError(e instanceof Error ? e.message : "Could not load products."))
      .finally(() => setLoading(false));
  }, []);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return products.filter((p) => {
      if (statusFilter !== "all" && p.status !== statusFilter) return false;
      if (!q) return true;
      return p.name.toLowerCase().includes(q) || p.mortgage_type.toLowerCase().includes(q);
    });
  }, [products, query, statusFilter]);

  function closePanel() {
    setEditingId(null);
    setForm(emptyForm);
  }

  function openCreate() {
    setError("");
    setMessage("");
    setForm(emptyForm);
    setEditingId("new");
  }

  function openEdit(p: Product) {
    setError("");
    setMessage("");
    setForm({
      name: p.name,
      description: p.description || "",
      mortgage_type: p.mortgage_type,
      interest_rate: strNum(p.interest_rate),
      interest_rate_min: strNum(p.interest_rate_min),
      interest_rate_max: strNum(p.interest_rate_max),
      interest_rate_type: p.interest_rate_type || "fixed",
      min_loan_amount: strNum(p.min_loan_amount),
      max_loan_amount: strNum(p.max_loan_amount),
      min_income: strNum(p.min_income),
      min_equity_pct: strNum(p.min_equity_pct),
      max_tenor_years: strNum(p.max_tenor_years),
      max_age: strNum(p.max_age),
      processing_fee: strNum(p.processing_fee),
      valuation_fee: strNum(p.valuation_fee),
      legal_fee: strNum(p.legal_fee),
      status: p.status,
      source: p.source || "",
      source_url: p.source_url || "",
    });
    setEditingId(p.id);
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setMessage("");
    setBusy(true);
    const body = {
      name: form.name,
      description: form.description,
      mortgage_type: form.mortgage_type,
      interest_rate: numOrNull(form.interest_rate),
      interest_rate_min: numOrNull(form.interest_rate_min),
      interest_rate_max: numOrNull(form.interest_rate_max),
      interest_rate_type: form.interest_rate_type,
      min_loan_amount: numOrNull(form.min_loan_amount),
      max_loan_amount: numOrNull(form.max_loan_amount),
      min_income: numOrNull(form.min_income),
      min_equity_pct: numOrNull(form.min_equity_pct),
      max_tenor_years: numOrNull(form.max_tenor_years),
      max_age: numOrNull(form.max_age),
      processing_fee: numOrNull(form.processing_fee),
      valuation_fee: numOrNull(form.valuation_fee),
      legal_fee: numOrNull(form.legal_fee),
      status: form.status,
      source: form.source,
      source_url: form.source_url,
      sync_rules: true,
    };
    try {
      if (creating) {
        const d = await api<{ product: Product }>("/api/v1/lender/products", {
          method: "POST",
          body: JSON.stringify(body),
        });
        setProducts((prev) => [d.product, ...prev.filter((x) => x.id !== d.product.id)]);
        setMessage(`${d.product.name} created. HomeGauge admin will verify before it stays fully trusted in the catalog.`);
      } else if (editing) {
        const d = await api<{ product: Product }>(`/api/v1/lender/products/${editing.id}`, {
          method: "PATCH",
          body: JSON.stringify(body),
        });
        setProducts((prev) => prev.map((x) => (x.id === d.product.id ? d.product : x)));
        setMessage(`${d.product.name} saved. Eligibility rules were refreshed; verification is pending again.`);
      }
      closePanel();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save product.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="lender-desk min-h-screen">
      <div className="mx-auto max-w-6xl px-5 py-10 md:px-8">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.18em] text-[#1f4d6b]">Lender portal</p>
            <h1 className="mt-2 font-[family-name:var(--font-display)] text-4xl font-semibold tracking-tight">
              Your products
            </h1>
            <p className="mt-3 max-w-2xl text-sm leading-relaxed text-muted">
              Only mortgage products for {me?.lender.name || "your institution"}. Edit rates, fees, and eligibility
              bands — saving updates the matching rules buyers see. New or edited products need HomeGauge verification.
            </p>
          </div>
          <button
            type="button"
            onClick={openCreate}
            className="rounded-sm bg-[#1f4d6b] px-4 py-2.5 text-sm font-semibold text-white hover:bg-[#163a51]"
          >
            Add product
          </button>
        </div>

        {error && <p className="mt-6 text-sm text-[color:var(--danger)]">{error}</p>}
        {message && <p className="mt-6 text-sm text-leaf">{message}</p>}
        {loading && <p className="mt-8 text-sm text-muted">Loading your products…</p>}

        {editingId && (
          <form onSubmit={onSubmit} className="lender-jacket mt-8 rounded-sm border border-[#1f4d6b]/20 p-6">
            <div className="flex items-start justify-between gap-3">
              <h2 className="font-[family-name:var(--font-display)] text-xl font-semibold">
                {creating ? "New product" : `Edit ${editing?.name}`}
              </h2>
              <button type="button" onClick={closePanel} className="text-sm font-semibold text-muted hover:text-ink">
                Close
              </button>
            </div>
            <p className="mt-2 text-xs text-muted">
              Locked to {me?.lender.name} · {me?.lender.country_code || "—"}
            </p>
            <div className="mt-5 grid gap-4 sm:grid-cols-2">
              <label className="block text-sm sm:col-span-2">
                <span className="mb-1.5 block font-medium">Product name</span>
                <input
                  required
                  minLength={2}
                  value={form.name}
                  onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                  className={fieldClass}
                />
              </label>
              <label className="block text-sm sm:col-span-2">
                <span className="mb-1.5 block font-medium">Description</span>
                <textarea
                  rows={3}
                  value={form.description}
                  onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                  className={fieldClass}
                />
              </label>
              <label className="block text-sm">
                <span className="mb-1.5 block font-medium">Type</span>
                <select
                  value={form.mortgage_type}
                  onChange={(e) => setForm((f) => ({ ...f, mortgage_type: e.target.value }))}
                  className={fieldClass}
                >
                  {TYPES.map((t) => (
                    <option key={t.value} value={t.value}>
                      {t.label}
                    </option>
                  ))}
                </select>
              </label>
              <label className="block text-sm">
                <span className="mb-1.5 block font-medium">Interest rate type</span>
                <select
                  value={form.interest_rate_type}
                  onChange={(e) => setForm((f) => ({ ...f, interest_rate_type: e.target.value }))}
                  className={fieldClass}
                >
                  <option value="fixed">Fixed</option>
                  <option value="variable">Variable</option>
                  <option value="negotiable">Negotiable</option>
                </select>
              </label>
              <NumField label="Indicative rate % p.a." value={form.interest_rate} onChange={(v) => setForm((f) => ({ ...f, interest_rate: v }))} />
              <NumField label="Min rate % p.a." value={form.interest_rate_min} onChange={(v) => setForm((f) => ({ ...f, interest_rate_min: v }))} />
              <NumField label="Max rate % p.a." value={form.interest_rate_max} onChange={(v) => setForm((f) => ({ ...f, interest_rate_max: v }))} />
              <NumField label="Min equity %" value={form.min_equity_pct} onChange={(v) => setForm((f) => ({ ...f, min_equity_pct: v }))} />
              <NumField label="Min loan amount" value={form.min_loan_amount} onChange={(v) => setForm((f) => ({ ...f, min_loan_amount: v }))} />
              <NumField label="Max loan amount" value={form.max_loan_amount} onChange={(v) => setForm((f) => ({ ...f, max_loan_amount: v }))} />
              <NumField label="Min monthly income" value={form.min_income} onChange={(v) => setForm((f) => ({ ...f, min_income: v }))} />
              <NumField label="Max tenor (years)" value={form.max_tenor_years} onChange={(v) => setForm((f) => ({ ...f, max_tenor_years: v }))} />
              <NumField label="Max age at maturity" value={form.max_age} onChange={(v) => setForm((f) => ({ ...f, max_age: v }))} />
              <NumField label="Processing fee" value={form.processing_fee} onChange={(v) => setForm((f) => ({ ...f, processing_fee: v }))} />
              <NumField label="Valuation fee" value={form.valuation_fee} onChange={(v) => setForm((f) => ({ ...f, valuation_fee: v }))} />
              <NumField label="Legal fee" value={form.legal_fee} onChange={(v) => setForm((f) => ({ ...f, legal_fee: v }))} />
              <label className="block text-sm">
                <span className="mb-1.5 block font-medium">Catalog status</span>
                <select value={form.status} onChange={(e) => setForm((f) => ({ ...f, status: e.target.value }))} className={fieldClass}>
                  <option value="active">Active (public)</option>
                  <option value="inactive">Inactive (hidden)</option>
                </select>
              </label>
              <label className="block text-sm">
                <span className="mb-1.5 block font-medium">Source</span>
                <input value={form.source} onChange={(e) => setForm((f) => ({ ...f, source: e.target.value }))} className={fieldClass} />
              </label>
              <label className="block text-sm sm:col-span-2">
                <span className="mb-1.5 block font-medium">Source URL</span>
                <input value={form.source_url} onChange={(e) => setForm((f) => ({ ...f, source_url: e.target.value }))} className={fieldClass} />
              </label>
            </div>
            <div className="mt-6">
              <button
                type="submit"
                disabled={busy}
                className="rounded-sm bg-[#1f4d6b] px-4 py-2.5 text-sm font-semibold text-white hover:bg-[#163a51] disabled:opacity-60"
              >
                {busy ? "Saving…" : creating ? "Create product" : "Save changes"}
              </button>
            </div>
          </form>
        )}

        {!loading && (
          <>
            <div className="mt-8 flex flex-wrap gap-3">
              <input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search products"
                className="min-w-[16rem] flex-1 rounded-sm border border-[#1f4d6b]/20 bg-white px-3 py-2 text-sm outline-none ring-[#1f4d6b] focus:ring-2"
              />
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className="rounded-sm border border-[#1f4d6b]/20 bg-white px-3 py-2 text-sm"
              >
                <option value="all">All statuses</option>
                <option value="active">Active</option>
                <option value="inactive">Inactive</option>
              </select>
            </div>

            <div className="lender-jacket mt-4 overflow-x-auto rounded-sm border border-[#1f4d6b]/15">
              <table className="w-full min-w-[720px] text-left text-sm">
                <thead className="border-b border-[#1f4d6b]/15 bg-[#1f4d6b]/5">
                  <tr>
                    <th className="px-4 py-3 font-semibold">Product</th>
                    <th className="px-4 py-3 font-semibold">Rate</th>
                    <th className="px-4 py-3 font-semibold">Status</th>
                    <th className="px-4 py-3 font-semibold">Verified</th>
                    <th className="px-4 py-3 font-semibold"> </th>
                  </tr>
                </thead>
                <tbody>
                  {visible.map((p) => (
                    <tr key={p.id} className="border-b border-[#1f4d6b]/10 last:border-0">
                      <td className="px-4 py-3">
                        <Link href={`/mortgages/${p.id}`} className="font-medium text-[#1f4d6b] hover:underline">
                          {p.name}
                        </Link>
                        <div className="text-xs text-muted">{p.mortgage_type}</div>
                      </td>
                      <td className="px-4 py-3">
                        {formatRate(p)}
                        <div className="text-xs text-muted">
                          {p.min_equity_pct != null ? `${p.min_equity_pct}% equity` : "equity n/a"}
                          {p.max_loan_amount != null ? ` · up to ${money(p.max_loan_amount)}` : ""}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span className={p.status === "active" ? "text-leaf" : "text-muted"}>{p.status}</span>
                      </td>
                      <td className="px-4 py-3 text-muted">{p.verification_status.replace(/_/g, " ")}</td>
                      <td className="px-4 py-3 text-right">
                        <button type="button" onClick={() => openEdit(p)} className="font-semibold text-[#1f4d6b] hover:underline">
                          Edit
                        </button>
                      </td>
                    </tr>
                  ))}
                  {visible.length === 0 && (
                    <tr>
                      <td colSpan={5} className="px-4 py-10 text-center text-muted">
                        {products.length === 0
                          ? "No products linked to your institution yet. Add one, or ask HomeGauge admin to attach existing catalog entries."
                          : "No products match that filter."}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

function NumField({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <label className="block text-sm">
      <span className="mb-1.5 block font-medium">{label}</span>
      <input inputMode="decimal" value={value} onChange={(e) => onChange(e.target.value)} className={fieldClass} />
    </label>
  );
}
