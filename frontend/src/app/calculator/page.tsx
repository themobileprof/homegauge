"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { CountrySwitcher, useCountry } from "@/lib/country";

const apiBase = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

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

export default function CalculatorPage() {
  const { money, country } = useCountry();
  const currencyHint = country?.currency_code || "local currency";
  const [propertyPrice, setPropertyPrice] = useState(35000000);
  const [deposit, setDeposit] = useState(3500000);
  const [rate, setRate] = useState(9.75);
  const [tenor, setTenor] = useState(20);
  const [income, setIncome] = useState(900000);
  const [debt, setDebt] = useState(50000);
  const [result, setResult] = useState<Result | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const loanPreview = useMemo(
    () => Math.max(0, propertyPrice - deposit),
    [propertyPrice, deposit],
  );

  async function calculate() {
    setLoading(true);
    setError("");
    try {
      const res = await fetch(`${apiBase}/api/v1/calculator/affordability`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
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
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Could not calculate");
      setResult(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not calculate");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen">
      <header className="mx-auto flex w-full max-w-5xl items-center justify-between px-5 py-5">
        <Link href="/" className="font-[family-name:var(--font-display)] text-2xl font-semibold">
          Home<span className="text-leaf">Gauge</span>
        </Link>
        <div className="flex items-center gap-4">
          <CountrySwitcher />
          <Link href="/register" className="text-sm font-semibold text-leaf">
            Check eligibility
          </Link>
        </div>
      </header>

      <main className="mx-auto grid max-w-5xl gap-8 px-5 pb-20 md:grid-cols-2">
        <section>
          <h1 className="font-[family-name:var(--font-display)] text-4xl font-semibold">Affordability calculator</h1>
          <p className="mt-3 text-muted">
            Estimates only. Changing price, deposit, rate, or years updates what a lender might expect you to repay each month.
          </p>

          <div className="mt-8 space-y-4">
            <Field label={`Property price (${currencyHint})`} value={propertyPrice} onChange={setPropertyPrice} />
            <Field label={`Your deposit / equity (${currencyHint})`} value={deposit} onChange={setDeposit} />
            <p className="text-sm text-muted">Estimated loan: <strong className="text-ink">{money(loanPreview)}</strong></p>
            <Field label="Interest rate (% per year)" value={rate} onChange={setRate} step={0.25} />
            <Field label="Years to repay (tenor)" value={tenor} onChange={setTenor} step={1} />
            <Field label={`Monthly take-home pay (${currencyHint})`} value={income} onChange={setIncome} />
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
          {!result && <p className="mt-4 text-sm text-muted">Run the calculator to see monthly repayment, total interest, and how much of your income the payment would use.</p>}
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
}: {
  label: string;
  value: number;
  onChange: (n: number) => void;
  step?: number;
}) {
  return (
    <label className="block text-sm">
      <span className="mb-1.5 block font-medium">{label}</span>
      <input
        type="number"
        step={step}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2"
      />
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
