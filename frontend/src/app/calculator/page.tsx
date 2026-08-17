"use client";

import { useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useCountry } from "@/lib/country";

type Result = {
  loan_amount: number;
  monthly_repayment: number;
  total_repayment: number;
  total_interest: number;
  loan_to_value_pct: number;
  debt_to_income_pct: number;
  required_equity: number;
  required_equity_pct: number;
  installment_to_income_pct: number;
  disclaimer: string;
};

type Product = {
  id: string;
  name: string;
  lender_name: string;
  interest_rate: number | null;
  interest_rate_type: string;
  max_tenor_years: number | null;
  min_equity_pct: number | null;
  min_income: number | null;
  min_loan_amount: number | null;
  max_loan_amount: number | null;
  verification_status: string;
};

export default function CalculatorPage() {
  const { money, country, countryCode } = useCountry();
  const currencyHint = country?.currency_code || "local currency";
  const [products, setProducts] = useState<Product[]>([]);
  const [productId, setProductId] = useState("");
  const [productsLoading, setProductsLoading] = useState(true);
  const [propertyPrice, setPropertyPrice] = useState(35000000);
  const [deposit, setDeposit] = useState(3500000);
  const [rate, setRate] = useState(9.75);
  const [tenor, setTenor] = useState(20);
  const [income, setIncome] = useState(900000);
  const [debt, setDebt] = useState(50000);
  const [result, setResult] = useState<Result | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const selected = useMemo(() => products.find((p) => p.id === productId) || null, [products, productId]);
  const rateLocked = selected?.interest_rate != null;
  const tenorLocked = selected?.max_tenor_years != null;
  const depositLocked = selected?.min_equity_pct != null;

  const loanPreview = useMemo(
    () => Math.max(0, propertyPrice - deposit),
    [propertyPrice, deposit],
  );

  useEffect(() => {
    setProductId("");
    setProductsLoading(true);
    api<{ products: Product[] }>(`/api/v1/mortgage-products?country=${encodeURIComponent(countryCode)}`)
      .then((d) => setProducts(d.products || []))
      .catch(() => setProducts([]))
      .finally(() => setProductsLoading(false));
  }, [countryCode]);

  useEffect(() => {
    if (!selected) return;
    if (selected.interest_rate != null) setRate(selected.interest_rate);
    if (selected.max_tenor_years != null) setTenor(selected.max_tenor_years);
    setResult(null);
  }, [selected]);

  useEffect(() => {
    if (selected?.min_equity_pct == null) return;
    setDeposit(Math.round((propertyPrice * selected.min_equity_pct) / 100));
  }, [selected, propertyPrice]);

  async function calculate() {
    setLoading(true);
    setError("");
    try {
      const data = await api<Result>("/api/v1/calculator/affordability", {
        method: "POST",
        body: JSON.stringify({
          property_price: propertyPrice,
          deposit,
          loan_amount: loanPreview,
          interest_rate: rate,
          tenor_years: tenor,
          monthly_income: income,
          existing_monthly_debt: debt,
        }),
      });
      setResult(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not calculate");
    } finally {
      setLoading(false);
    }
  }

  const loanNote =
    selected &&
    ((selected.min_loan_amount != null && loanPreview < selected.min_loan_amount) ||
      (selected.max_loan_amount != null && loanPreview > selected.max_loan_amount))
      ? `This product’s stated loan band is ${money(selected.min_loan_amount)} – ${money(selected.max_loan_amount)}.`
      : null;
  const incomeNote =
    selected?.min_income != null && income < selected.min_income
      ? `Stated minimum monthly income for this product is about ${money(selected.min_income)}.`
      : null;

  return (
    <div>
      <main className="mx-auto grid max-w-5xl gap-8 px-5 py-10 pb-20 md:grid-cols-2">
        <section>
          <h1 className="font-[family-name:var(--font-display)] text-4xl font-semibold">Affordability calculator</h1>
          <p className="mt-3 text-muted">
            Estimates only. Pick a product in {country?.name || "your market"} to lock its published rate, tenor, and minimum equity — or enter terms yourself.
          </p>

          <div className="mt-8 space-y-4">
            <label className="block text-sm">
              <span className="mb-1.5 block font-medium">Mortgage product</span>
              <select
                value={productId}
                onChange={(e) => setProductId(e.target.value)}
                disabled={productsLoading}
                className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2 disabled:bg-paper-2"
              >
                <option value="">Custom estimate (enter your own terms)</option>
                {products.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.lender_name} — {p.name}
                    {p.interest_rate != null ? ` (${p.interest_rate}%)` : ""}
                  </option>
                ))}
              </select>
              {productsLoading && <p className="mt-1 text-xs text-muted">Loading products…</p>}
              {!productsLoading && products.length === 0 && (
                <p className="mt-1 text-xs text-muted">No published products in this market yet. Enter terms yourself.</p>
              )}
              {selected && (
                <p className="mt-1 text-xs text-muted">
                  Using {selected.name}. Rate, tenor, and minimum equity come from the product
                  {selected.verification_status === "verified" ? " and were marked verified" : " and still need verification"}.
                </p>
              )}
            </label>

            <Field label={`Property price (${currencyHint})`} value={propertyPrice} onChange={setPropertyPrice} />
            <Field
              label={`Your deposit / equity (${currencyHint})`}
              value={deposit}
              onChange={setDeposit}
              readOnly={depositLocked}
              hint={
                depositLocked
                  ? `Locked to this product’s ${selected?.min_equity_pct}% minimum equity. Changes with property price.`
                  : undefined
              }
            />
            <p className="text-sm text-muted">
              Estimated loan: <strong className="text-ink">{money(loanPreview)}</strong>
            </p>
            {loanNote && <p className="text-sm text-[color:var(--danger)]">{loanNote}</p>}
            <Field
              label="Interest rate (% per year)"
              value={rate}
              onChange={setRate}
              step={0.25}
              readOnly={rateLocked}
              hint={
                rateLocked
                  ? `Locked to ${selected?.name} (${selected?.interest_rate_type || "stated"} rate).`
                  : undefined
              }
            />
            <Field
              label="Years to repay (tenor)"
              value={tenor}
              onChange={setTenor}
              step={1}
              readOnly={tenorLocked}
              hint={tenorLocked ? `Locked to this product’s maximum tenor of ${selected?.max_tenor_years} years.` : undefined}
            />
            <Field label={`Monthly take-home pay (${currencyHint})`} value={income} onChange={setIncome} />
            {incomeNote && <p className="text-sm text-[color:var(--danger)]">{incomeNote}</p>}
            <Field label={`Other monthly debt payments (${currencyHint})`} value={debt} onChange={setDebt} />
            <button
              type="button"
              onClick={calculate}
              disabled={loading}
              className="rounded-md bg-leaf px-5 py-3 text-sm font-semibold text-white hover:bg-leaf-deep disabled:opacity-60"
            >
              {loading ? "Calculating…" : "Calculate estimate"}
            </button>
            {error && <p className="text-sm text-[color:var(--danger)]">{error}</p>}
          </div>
        </section>

        <section className="rounded-2xl border border-[color:var(--line)] bg-white/70 p-6 md:p-8">
          <h2 className="text-lg font-semibold">Your estimate</h2>
          {selected && (
            <p className="mt-2 text-sm text-muted">
              Scenario for <strong className="text-ink">{selected.name}</strong> from {selected.lender_name}.
            </p>
          )}
          {!result && (
            <p className="mt-4 text-sm text-muted">
              Run the calculator to see monthly repayment, total interest, and how much of your income the payment would use.
            </p>
          )}
          {result && (
            <dl className="mt-6 space-y-4 text-sm">
              <Row label="Estimated monthly repayment" value={money(result.monthly_repayment)} />
              <Row label="Total you would repay" value={money(result.total_repayment)} />
              <Row label="Total interest" value={money(result.total_interest)} />
              <Row label="Loan as % of property price (LTV)" value={`${result.loan_to_value_pct}%`} />
              <Row label="Payment as % of income (ITI)" value={`${result.installment_to_income_pct}%`} />
              <Row label="Debts + mortgage vs income (DTI)" value={`${result.debt_to_income_pct}%`} />
              <Row label="Equity / deposit in this scenario" value={`${money(result.required_equity)} (${result.required_equity_pct}%)`} />
              <p className="pt-4 text-xs leading-relaxed text-muted">{result.disclaimer}</p>
            </dl>
          )}
        </section>
      </main>
    </div>
  );
}

function Field({
  label,
  value,
  onChange,
  step = 1000,
  readOnly = false,
  hint,
}: {
  label: string;
  value: number;
  onChange: (n: number) => void;
  step?: number;
  readOnly?: boolean;
  hint?: string;
}) {
  return (
    <label className="block text-sm">
      <span className="mb-1.5 block font-medium">{label}</span>
      <input
        type="number"
        step={step}
        value={value}
        readOnly={readOnly}
        onChange={(e) => onChange(Number(e.target.value))}
        className={`w-full rounded-md border border-[color:var(--line)] px-3 py-2.5 outline-none ring-leaf focus:ring-2 ${
          readOnly ? "cursor-not-allowed bg-paper-2 text-ink-soft" : "bg-white"
        }`}
      />
      {hint && <span className="mt-1 block text-xs text-muted">{hint}</span>}
    </label>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between gap-4 border-b border-[color:var(--line)] pb-3">
      <dt className="text-muted">{label}</dt>
      <dd className="text-right font-semibold">{value}</dd>
    </div>
  );
}
