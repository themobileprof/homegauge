"use client";

import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useCountry } from "@/lib/country";

type Assessment = { id: string };

const steps = ["About you", "Your job & salary", "Money & home goal", "Review"];

export default function AssessmentPage() {
  const router = useRouter();
  const { country, countryCode, money } = useCountry();
  const currency = country?.currency_code || "local currency";
  const regionLabel = country?.region_label || "Region";
  const regions = country?.regions?.length ? country.regions : ["Other"];
  const regionsKey = regions.join("|");
  const showSchemeMonths = countryCode === "NG";

  const [step, setStep] = useState(0);
  const [assessmentId, setAssessmentId] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({
    country_code: countryCode,
    full_name: "",
    date_of_birth: "",
    state_of_residence: "",
    residency_type: "resident",
    marital_status: "single",
    employment_type: "salaried",
    employer_name: "",
    years_employed: 1,
    monthly_net_income: 0,
    other_monthly_income: 0,
    salary_months: 6,
    nhf_contributor_months: 0,
    willing_to_domicile_salary: true,
    monthly_expenses: 0,
    existing_debt_repayments: 0,
    available_deposit: 0,
    desired_property_price: 0,
    desired_loan_amount: 0,
    preferred_tenor_years: 20,
  });

  useEffect(() => {
    setForm((f) => ({
      ...f,
      country_code: countryCode,
      state_of_residence: regions.includes(f.state_of_residence) ? f.state_of_residence : regions[0] || "",
    }));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- regionsKey captures region list changes
  }, [countryCode, regionsKey]);

  const progress = useMemo(() => Math.round(((step + 1) / steps.length) * 100), [step]);

  function set<K extends keyof typeof form>(key: K, value: (typeof form)[K]) {
    setForm((f) => ({ ...f, [key]: value }));
  }

  async function ensureAssessment() {
    if (assessmentId) return assessmentId;
    const data = await api<{ assessment: Assessment }>("/api/v1/assessments", { method: "POST" });
    setAssessmentId(data.assessment.id);
    return data.assessment.id;
  }

  async function saveAndNext(e: FormEvent) {
    e.preventDefault();
    setError("");
    setSaving(true);
    try {
      if (form.employment_type !== "salaried" && form.employment_type !== "civil_servant") {
        setError("Automated eligibility currently supports salaried workers with a salary account. You can still browse products and use the calculator.");
        setSaving(false);
        return;
      }
      const id = await ensureAssessment();
      await api(`/api/v1/assessments/${id}`, {
        method: "PATCH",
        body: JSON.stringify({ ...form, country_code: countryCode }),
      });
      if (step < steps.length - 1) {
        setStep((s) => s + 1);
      } else {
        const done = await api<{ assessment: { id: string } }>(`/api/v1/assessments/${id}/complete`, {
          method: "POST",
        });
        router.push(`/app/assessment/${done.assessment.id}/results`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="mx-auto max-w-xl px-5 py-8">
      <h1 className="font-[family-name:var(--font-display)] text-3xl font-semibold">Check eligibility</h1>
      <p className="mt-2 text-sm text-muted">
        Built around your salary account in {country?.name || "your market"}. We look for about 6 months of clear salary credits.
      </p>

      <div className="mt-6">
        <div className="mb-2 flex justify-between text-xs font-semibold text-muted">
          <span>{steps[step]}</span>
          <span>{progress}%</span>
        </div>
        <div className="h-2 overflow-hidden rounded-full bg-paper-2">
          <div className="h-full bg-leaf transition-all" style={{ width: `${progress}%` }} />
        </div>
      </div>

      <form onSubmit={saveAndNext} className="mt-8 space-y-4">
        {step === 0 && (
          <>
            <Field label="Full name" value={form.full_name} onChange={(v) => set("full_name", v)} required />
            <Field label="Date of birth" type="date" value={form.date_of_birth} onChange={(v) => set("date_of_birth", v)} required />
            <label className="block text-sm">
              <span className="mb-1.5 block font-medium">{regionLabel}</span>
              <select className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5" value={form.state_of_residence} onChange={(e) => set("state_of_residence", e.target.value)}>
                {regions.map((s) => <option key={s}>{s}</option>)}
              </select>
            </label>
            <label className="block text-sm">
              <span className="mb-1.5 block font-medium">Marital status</span>
              <select className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5" value={form.marital_status} onChange={(e) => set("marital_status", e.target.value)}>
                <option value="single">Single</option>
                <option value="married">Married</option>
                <option value="divorced">Divorced</option>
                <option value="widowed">Widowed</option>
              </select>
            </label>
          </>
        )}

        {step === 1 && (
          <>
            <label className="block text-sm">
              <span className="mb-1.5 block font-medium">Employment type</span>
              <select className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5" value={form.employment_type} onChange={(e) => set("employment_type", e.target.value)}>
                <option value="salaried">Salaried employee</option>
                <option value="civil_servant">Civil servant</option>
                <option value="self_employed">Self-employed (browse only for now)</option>
              </select>
            </label>
            <Field label="Employer" value={form.employer_name} onChange={(v) => set("employer_name", v)} required />
            <Num label="Years with current employer" value={form.years_employed} onChange={(v) => set("years_employed", v)} step={0.5} />
            <Num label={`Monthly take-home pay (${currency})`} value={form.monthly_net_income} onChange={(v) => set("monthly_net_income", v)} />
            <Num label={`Other monthly income (${currency})`} value={form.other_monthly_income} onChange={(v) => set("other_monthly_income", v)} />
            <Num label="Months of clear salary credits on one account" value={form.salary_months} onChange={(v) => set("salary_months", v)} step={1} />
            {showSchemeMonths && (
              <Num label="Months contributing to NHF (if any)" value={form.nhf_contributor_months} onChange={(v) => set("nhf_contributor_months", v)} step={1} />
            )}
            <label className="flex items-start gap-3 text-sm">
              <input type="checkbox" checked={form.willing_to_domicile_salary} onChange={(e) => set("willing_to_domicile_salary", e.target.checked)} className="mt-1" />
              <span>I am willing to move my salary to the lender’s bank if required.</span>
            </label>
          </>
        )}

        {step === 2 && (
          <>
            <Num label={`Monthly living expenses (${currency})`} value={form.monthly_expenses} onChange={(v) => set("monthly_expenses", v)} />
            <Num label={`Other monthly loan/debt payments (${currency})`} value={form.existing_debt_repayments} onChange={(v) => set("existing_debt_repayments", v)} />
            <Num label={`Deposit / equity you can put down (${currency})`} value={form.available_deposit} onChange={(v) => set("available_deposit", v)} />
            <Num label={`Desired property price (${currency})`} value={form.desired_property_price} onChange={(v) => set("desired_property_price", v)} />
            <Num label={`Desired loan amount (${currency}) — leave 0 to calculate from price − deposit`} value={form.desired_loan_amount} onChange={(v) => set("desired_loan_amount", v)} />
            <Num label="Preferred years to repay" value={form.preferred_tenor_years} onChange={(v) => set("preferred_tenor_years", v)} step={1} />
          </>
        )}

        {step === 3 && (
          <div className="space-y-3 rounded-xl border border-[color:var(--line)] bg-white/70 p-5 text-sm">
            <p><strong>{form.full_name}</strong> · {country?.name} · {form.state_of_residence}</p>
            <p>{form.employer_name} · take-home {money(form.monthly_net_income)}/month</p>
            <p>{form.salary_months} months of salary credits declared</p>
            <p>Deposit {money(form.available_deposit)} toward {money(form.desired_property_price)} property</p>
            <p className="text-xs text-muted">Submitting will compare you to products in {country?.name}. Results are not a bank approval.</p>
          </div>
        )}

        {error && <p className="text-sm text-[color:var(--danger)]">{error}</p>}

        <div className="flex gap-3 pt-2">
          {step > 0 && (
            <button type="button" onClick={() => setStep((s) => s - 1)} className="rounded-md border border-[color:var(--line)] px-4 py-3 text-sm font-semibold">
              Back
            </button>
          )}
          <button type="submit" disabled={saving} className="flex-1 rounded-md bg-leaf px-4 py-3 text-sm font-semibold text-white hover:bg-leaf-deep disabled:opacity-60">
            {saving ? "Saving…" : step === steps.length - 1 ? "See my results" : "Continue"}
          </button>
        </div>
      </form>
    </div>
  );
}

function Field({ label, value, onChange, type = "text", required }: { label: string; value: string; onChange: (v: string) => void; type?: string; required?: boolean }) {
  return (
    <label className="block text-sm">
      <span className="mb-1.5 block font-medium">{label}</span>
      <input required={required} type={type} value={value} onChange={(e) => onChange(e.target.value)} className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2" />
    </label>
  );
}

function Num({ label, value, onChange, step = 1000 }: { label: string; value: number; onChange: (v: number) => void; step?: number }) {
  return (
    <label className="block text-sm">
      <span className="mb-1.5 block font-medium">{label}</span>
      <input type="number" step={step} value={value} onChange={(e) => onChange(Number(e.target.value))} className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2" />
    </label>
  );
}
