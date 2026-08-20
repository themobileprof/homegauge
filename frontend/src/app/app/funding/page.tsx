"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useCountry } from "@/lib/country";
import type { FundingSnapshot } from "@/lib/journey";

export default function FundingPage() {
  const { money } = useCountry();
  const [snap, setSnap] = useState<FundingSnapshot | null>(null);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [phone, setPhone] = useState("");

  async function load() {
    const d = await api<FundingSnapshot>("/api/v1/applications/me/funding");
    setSnap(d);
  }

  useEffect(() => {
    load().catch((e) => setError(e.message));
  }, []);

  async function openAccount() {
    setBusy(true);
    setError("");
    setMessage("");
    try {
      await api("/api/v1/applications/me/funding/account", {
        method: "POST",
        body: JSON.stringify({ phone: phone.trim() }),
      });
      setMessage("Case collection account is ready. Transfer fees to the account below.");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not open collection account.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-3xl px-5 py-10">
      <Link href="/app" className="text-sm font-semibold text-leaf">
        ← Journey
      </Link>
      <p className="mt-6 text-xs font-semibold uppercase tracking-[0.16em] text-leaf">Fund & settle</p>
      <h1 className="mt-2 font-[family-name:var(--font-display)] text-3xl font-semibold">Pre-disbursement costs</h1>
      <p className="mt-3 text-sm text-muted">
        Pay valuation, legal, and processing fees into your case collection account before the lender disburses.
        This is not monthly mortgage repayment and HomeGauge is not a bank.
      </p>

      {error && <p className="mt-4 text-sm text-[color:var(--danger)]">{error}</p>}
      {message && <p className="mt-4 text-sm text-leaf">{message}</p>}

      {snap && (
        <>
          {snap.preferred_product_name && (
            <p className="mt-6 text-sm">
              Product: <strong>{snap.preferred_product_name}</strong>
            </p>
          )}

          <div className="mt-4 grid gap-3 sm:grid-cols-3">
            <Stat label="Due (collectable)" value={money(snap.total_due)} />
            <Stat label="Received" value={money(snap.total_received)} />
            <Stat label="Outstanding" value={money(snap.total_outstanding)} />
          </div>

          {!snap.enabled && (
            <p className="mt-6 rounded-md border border-[color:var(--line)] bg-paper-2/80 px-4 py-3 text-sm text-muted">
              Paystack is not configured on this environment yet. Ask an admin to set PAYSTACK_SECRET_KEY /
              PAYSTACK_PUBLIC_KEY. You can still review obligations below.
            </p>
          )}

          <section className="mt-8 rounded-xl border border-[color:var(--line)] bg-white/80 p-5">
            <h2 className="font-semibold">Case collection account</h2>
            {snap.account?.account_number ? (
              <dl className="mt-4 space-y-2 text-sm">
                <Row label="Bank" value={snap.account.bank_name} />
                <Row label="Account number" value={snap.account.account_number} />
                <Row label="Account name" value={snap.account.account_name} />
              </dl>
            ) : (
              <div className="mt-4 space-y-3">
                <p className="text-sm text-muted">
                  Open a dedicated virtual account for this file. Transfers are matched to your obligations automatically.
                </p>
                <label className="block text-sm">
                  <span className="mb-1.5 block font-medium">Phone (optional)</span>
                  <input
                    value={phone}
                    onChange={(e) => setPhone(e.target.value)}
                    className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2"
                    placeholder="080…"
                  />
                </label>
                <button
                  type="button"
                  disabled={busy || !snap.enabled}
                  onClick={openAccount}
                  className="rounded-md bg-leaf px-4 py-2.5 text-sm font-semibold text-white hover:bg-leaf-deep disabled:opacity-60"
                >
                  {busy ? "Opening…" : "Open collection account"}
                </button>
              </div>
            )}
          </section>

          <section className="mt-8">
            <h2 className="text-lg font-semibold">Obligations</h2>
            <ul className="mt-4 space-y-3">
              {snap.obligations.map((o) => (
                <li key={o.id} className="rounded-xl border border-[color:var(--line)] bg-white/70 p-4">
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <div>
                      <p className="font-semibold">{o.label}</p>
                      <p className="mt-1 text-xs uppercase tracking-wide text-muted">{o.status.replaceAll("_", " ")}</p>
                    </div>
                    <p className="text-sm font-medium">
                      {o.amount != null ? money(o.amount) : "Confirm"}
                      {o.amount_received > 0 ? ` · received ${money(o.amount_received)}` : ""}
                    </p>
                  </div>
                  {o.note && <p className="mt-2 text-sm text-muted">{o.note}</p>}
                  {!o.collectable && <p className="mt-2 text-xs text-muted">Not collected via this account — handle with advisor/lender.</p>}
                </li>
              ))}
              {snap.obligations.length === 0 && (
                <li className="text-sm text-muted">Choose a product first so we can build your fee list.</li>
              )}
            </ul>
          </section>

          {snap.movements.length > 0 && (
            <section className="mt-8">
              <h2 className="text-lg font-semibold">Recent transfers</h2>
              <ul className="mt-3 space-y-2 text-sm">
                {snap.movements.map((m) => (
                  <li key={m.id} className="flex justify-between gap-3 border-b border-[color:var(--line)] pb-2">
                    <span>{money(m.amount)}</span>
                    <span className="text-muted">{new Date(m.created_at).toLocaleString("en-NG")}</span>
                  </li>
                ))}
              </ul>
            </section>
          )}
        </>
      )}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-[color:var(--line)] bg-white/70 p-4">
      <p className="text-xs uppercase tracking-wide text-muted">{label}</p>
      <p className="mt-1 font-semibold">{value}</p>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3 border-b border-[color:var(--line)] pb-2">
      <dt className="text-muted">{label}</dt>
      <dd className="font-medium">{value}</dd>
    </div>
  );
}
