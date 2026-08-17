"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useCountry } from "@/lib/country";

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
  interest_rate_type: string;
  processing_fee: number | null;
  valuation_fee: number | null;
  legal_fee: number | null;
  status: string;
  source?: string | null;
  source_url?: string | null;
  verification_status: string;
  last_verified_at?: string | null;
};

type Lender = {
  id: string;
  country_code: string;
  name: string;
  status: string;
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
  lender_id: "",
  country_code: "NG",
  mortgage_type: "commercial",
  interest_rate: "",
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
  verification_status: "needs_verification",
  source: "",
  source_url: "",
};

const fieldClass =
  "w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2";

function numOrNull(v: string): number | null {
  const t = v.trim();
  if (!t) return null;
  const n = Number(t);
  return Number.isFinite(n) ? n : null;
}

function strNum(v: number | null | undefined) {
  return v == null ? "" : String(v);
}

export default function AdminProductsPage() {
  const { countries, money } = useCountry();
  const [products, setProducts] = useState<Product[]>([]);
  const [lenders, setLenders] = useState<Lender[]>([]);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [countryFilter, setCountryFilter] = useState("all");
  const [editingId, setEditingId] = useState<string | "new" | null>(null);
  const [form, setForm] = useState(emptyForm);
  const [newLenderName, setNewLenderName] = useState("");
  const [addingLender, setAddingLender] = useState(false);

  const editing = editingId && editingId !== "new" ? products.find((p) => p.id === editingId) : null;
  const creating = editingId === "new";

  async function load() {
    const [p, l] = await Promise.all([
      api<{ products: Product[] }>("/api/v1/admin/products"),
      api<{ lenders: Lender[] }>("/api/v1/admin/lenders"),
    ]);
    setProducts(p.products || []);
    setLenders(l.lenders || []);
  }

  useEffect(() => {
    load()
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const lendersForCountry = useMemo(
    () => lenders.filter((l) => l.country_code === form.country_code),
    [lenders, form.country_code],
  );

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return products.filter((p) => {
      if (statusFilter !== "all" && p.status !== statusFilter) return false;
      if (countryFilter !== "all" && p.country_code !== countryFilter) return false;
      if (!q) return true;
      return (
        p.name.toLowerCase().includes(q) ||
        p.lender_name.toLowerCase().includes(q) ||
        p.mortgage_type.toLowerCase().includes(q)
      );
    });
  }, [products, query, statusFilter, countryFilter]);

  function openCreate() {
    setError("");
    setMessage("");
    setNewLenderName("");
    setForm({ ...emptyForm, country_code: countries.find((c) => c.status === "active")?.code || "NG" });
    setEditingId("new");
  }

  function openEdit(p: Product) {
    setError("");
    setMessage("");
    setNewLenderName("");
    setForm({
      name: p.name,
      description: p.description || "",
      lender_id: p.lender_id,
      country_code: p.country_code,
      mortgage_type: TYPES.some((t) => t.value === p.mortgage_type) ? p.mortgage_type : "other",
      interest_rate: strNum(p.interest_rate),
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
      status: p.status || "active",
      verification_status: p.verification_status || "needs_verification",
      source: p.source || "",
      source_url: p.source_url || "",
    });
    setEditingId(p.id);
  }

  function closePanel() {
    setEditingId(null);
    setForm(emptyForm);
  }

  function bodyFromForm() {
    const maxAge = numOrNull(form.max_age);
    const tenor = numOrNull(form.max_tenor_years);
    return {
      name: form.name.trim(),
      description: form.description.trim(),
      lender_id: form.lender_id,
      country_code: form.country_code,
      mortgage_type: form.mortgage_type,
      interest_rate: numOrNull(form.interest_rate),
      interest_rate_type: form.interest_rate_type,
      min_loan_amount: numOrNull(form.min_loan_amount),
      max_loan_amount: numOrNull(form.max_loan_amount),
      min_income: numOrNull(form.min_income),
      min_equity_pct: numOrNull(form.min_equity_pct),
      max_tenor_years: tenor == null ? null : Math.round(tenor),
      max_age: maxAge == null ? null : Math.round(maxAge),
      processing_fee: numOrNull(form.processing_fee),
      valuation_fee: numOrNull(form.valuation_fee),
      legal_fee: numOrNull(form.legal_fee),
      status: form.status,
      verification_status: form.verification_status,
      source: form.source.trim(),
      source_url: form.source_url.trim(),
      sync_rules: true,
    };
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setMessage("");
    if (!form.lender_id) {
      setError("Choose a lender, or add one first.");
      return;
    }
    setBusy(true);
    try {
      if (creating) {
        const d = await api<{ product: Product }>("/api/v1/admin/products", {
          method: "POST",
          body: JSON.stringify(bodyFromForm()),
        });
        setProducts((prev) => [d.product, ...prev.filter((p) => p.id !== d.product.id)]);
        setMessage(`${d.product.name} is on the catalog.`);
        closePanel();
      } else if (editing) {
        const d = await api<{ product: Product }>(`/api/v1/admin/products/${editing.id}`, {
          method: "PATCH",
          body: JSON.stringify(bodyFromForm()),
        });
        setProducts((prev) => prev.map((p) => (p.id === d.product.id ? d.product : p)));
        setMessage("Saved. Eligibility rules were updated from these numbers.");
        closePanel();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save product.");
    } finally {
      setBusy(false);
    }
  }

  async function onAddLender() {
    const name = newLenderName.trim();
    if (name.length < 2) {
      setError("Lender name must be at least 2 characters.");
      return;
    }
    setAddingLender(true);
    setError("");
    try {
      const d = await api<{ lender: Lender }>("/api/v1/admin/lenders", {
        method: "POST",
        body: JSON.stringify({ name, country_code: form.country_code }),
      });
      setLenders((prev) => [...prev, d.lender].sort((a, b) => a.name.localeCompare(b.name)));
      setForm((f) => ({ ...f, lender_id: d.lender.id }));
      setNewLenderName("");
      setMessage(`${d.lender.name} added as a lender.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not add lender.");
    } finally {
      setAddingLender(false);
    }
  }

  async function onRemove(p: Product) {
    if (!window.confirm(`Remove ${p.name}? It will disappear from the public catalog.`)) return;
    setError("");
    setMessage("");
    setBusy(true);
    try {
      await api(`/api/v1/admin/products/${p.id}`, { method: "DELETE" });
      setProducts((prev) => prev.filter((x) => x.id !== p.id));
      if (editingId === p.id) closePanel();
      setMessage(`${p.name} removed.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not remove product.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-6xl px-5 py-10 md:px-8">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.16em] text-gold">Admin console</p>
          <h1 className="mt-2 font-[family-name:var(--font-display)] text-3xl font-semibold">Products</h1>
          <p className="mt-2 max-w-xl text-sm text-muted">
            Publish mortgage products by market. Active, verified products appear on Compare Mortgage Options.
            Saving also updates the matching eligibility rules (income, age, equity, loan size).
          </p>
        </div>
        <button
          type="button"
          onClick={openCreate}
          className="rounded-md bg-ink px-4 py-2.5 text-sm font-semibold text-paper hover:bg-ink-soft"
        >
          Add product
        </button>
      </div>

      {error && <p className="mt-4 text-sm text-[color:var(--danger)]">{error}</p>}
      {message && <p className="mt-4 text-sm text-leaf">{message}</p>}

      {editingId && (
        <form onSubmit={onSubmit} className="mt-8 rounded-xl border border-[color:var(--line)] bg-white/80 p-6">
          <div className="flex items-start justify-between gap-3">
            <h2 className="text-lg font-semibold">{creating ? "New product" : `Edit ${editing?.name}`}</h2>
            <button type="button" onClick={closePanel} className="text-sm font-semibold text-muted hover:text-ink">
              Close
            </button>
          </div>
          <div className="mt-5 grid gap-4 sm:grid-cols-2">
            <label className="block text-sm sm:col-span-2">
              <span className="mb-1.5 block font-medium">Product name</span>
              <input required minLength={2} value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} className={fieldClass} />
            </label>
            <label className="block text-sm">
              <span className="mb-1.5 block font-medium">Country</span>
              <select
                value={form.country_code}
                onChange={(e) => setForm((f) => ({ ...f, country_code: e.target.value, lender_id: "" }))}
                className={fieldClass}
              >
                {countries.map((c) => (
                  <option key={c.code} value={c.code}>
                    {c.name}
                    {c.status !== "active" ? " (soon)" : ""}
                  </option>
                ))}
              </select>
            </label>
            <label className="block text-sm">
              <span className="mb-1.5 block font-medium">Lender</span>
              <select
                required
                value={form.lender_id}
                onChange={(e) => setForm((f) => ({ ...f, lender_id: e.target.value }))}
                className={fieldClass}
              >
                <option value="">Select a lender</option>
                {lendersForCountry.map((l) => (
                  <option key={l.id} value={l.id}>
                    {l.name}
                  </option>
                ))}
              </select>
            </label>
            <div className="sm:col-span-2 flex flex-wrap items-end gap-2">
              <label className="block min-w-[16rem] flex-1 text-sm">
                <span className="mb-1.5 block font-medium">Or add a lender in this country</span>
                <input
                  value={newLenderName}
                  onChange={(e) => setNewLenderName(e.target.value)}
                  placeholder="Lender name"
                  className={fieldClass}
                />
              </label>
              <button
                type="button"
                onClick={onAddLender}
                disabled={addingLender}
                className="rounded-md border border-[color:var(--line)] px-3 py-2.5 text-sm font-semibold hover:bg-white disabled:opacity-60"
              >
                {addingLender ? "Adding…" : "Add lender"}
              </button>
            </div>
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
              <select value={form.mortgage_type} onChange={(e) => setForm((f) => ({ ...f, mortgage_type: e.target.value }))} className={fieldClass}>
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
              </select>
            </label>
            <NumField label="Interest rate % p.a." value={form.interest_rate} onChange={(v) => setForm((f) => ({ ...f, interest_rate: v }))} />
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
              <span className="mb-1.5 block font-medium">Verification</span>
              <select
                value={form.verification_status}
                onChange={(e) => setForm((f) => ({ ...f, verification_status: e.target.value }))}
                className={fieldClass}
              >
                <option value="needs_verification">Needs verification</option>
                <option value="verified">Verified</option>
                <option value="expired">Expired</option>
              </select>
            </label>
            <label className="block text-sm">
              <span className="mb-1.5 block font-medium">Source</span>
              <input value={form.source} onChange={(e) => setForm((f) => ({ ...f, source: e.target.value }))} className={fieldClass} />
            </label>
            <label className="block text-sm">
              <span className="mb-1.5 block font-medium">Source URL</span>
              <input value={form.source_url} onChange={(e) => setForm((f) => ({ ...f, source_url: e.target.value }))} className={fieldClass} />
            </label>
          </div>
          <div className="mt-6 flex flex-wrap items-center gap-3">
            <button
              type="submit"
              disabled={busy}
              className="rounded-md bg-leaf px-4 py-2.5 text-sm font-semibold text-white hover:bg-leaf-deep disabled:opacity-60"
            >
              {busy ? "Saving…" : creating ? "Create product" : "Save changes"}
            </button>
            {!creating && editing && (
              <button
                type="button"
                disabled={busy}
                onClick={() => onRemove(editing)}
                className="rounded-md border border-[color:var(--danger)]/30 px-4 py-2.5 text-sm font-semibold text-[color:var(--danger)] hover:bg-red-50 disabled:opacity-60"
              >
                Remove
              </button>
            )}
          </div>
        </form>
      )}

      <div className="mt-8 flex flex-wrap gap-3">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search product or lender"
          className="min-w-[16rem] flex-1 rounded-md border border-[color:var(--line)] bg-white px-3 py-2 text-sm outline-none ring-leaf focus:ring-2"
        />
        <select value={countryFilter} onChange={(e) => setCountryFilter(e.target.value)} className="rounded-md border border-[color:var(--line)] bg-white px-3 py-2 text-sm">
          <option value="all">All countries</option>
          {countries.map((c) => (
            <option key={c.code} value={c.code}>
              {c.name}
            </option>
          ))}
        </select>
        <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className="rounded-md border border-[color:var(--line)] bg-white px-3 py-2 text-sm">
          <option value="all">All statuses</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
        </select>
      </div>

      <div className="mt-4 overflow-x-auto rounded-xl border border-[color:var(--line)] bg-white/80">
        <table className="w-full min-w-[860px] text-left text-sm">
          <thead className="border-b border-[color:var(--line)] bg-paper-2/60">
            <tr>
              <th className="px-4 py-3 font-semibold">Product</th>
              <th className="px-4 py-3 font-semibold">Lender</th>
              <th className="px-4 py-3 font-semibold">Rate</th>
              <th className="px-4 py-3 font-semibold">Status</th>
              <th className="px-4 py-3 font-semibold">Verified</th>
              <th className="px-4 py-3 font-semibold"> </th>
            </tr>
          </thead>
          <tbody>
            {visible.map((p) => (
              <tr key={p.id} className="border-b border-[color:var(--line)] last:border-0">
                <td className="px-4 py-3">
                  <div className="font-medium">{p.name}</div>
                  <div className="text-xs text-muted">
                    {p.country_code} · {p.mortgage_type}
                  </div>
                </td>
                <td className="px-4 py-3">{p.lender_name}</td>
                <td className="px-4 py-3">
                  {p.interest_rate != null ? `${p.interest_rate}% ${p.interest_rate_type}` : "—"}
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
                  <button type="button" onClick={() => openEdit(p)} className="font-semibold text-leaf hover:underline">
                    Edit
                  </button>
                </td>
              </tr>
            ))}
            {!loading && visible.length === 0 && (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-muted">
                  No products match that filter.
                </td>
              </tr>
            )}
            {loading && (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-muted">
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

function NumField({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <label className="block text-sm">
      <span className="mb-1.5 block font-medium">{label}</span>
      <input inputMode="decimal" value={value} onChange={(e) => onChange(e.target.value)} className={fieldClass} />
    </label>
  );
}
