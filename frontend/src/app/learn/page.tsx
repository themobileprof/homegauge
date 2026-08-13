import Link from "next/link";

const guides = [
  {
    title: "What a salary account means for mortgages",
    body: "Lenders typically look for several consecutive months of clear salary credits on one account — similar amounts, usually near payday. HomeGauge’s eligibility check follows that pattern in each active market.",
  },
  {
    title: "Scheme loans vs commercial mortgages",
    body: "Some countries offer housing-scheme or development-fund products with different rates and contribution rules. Commercial bank mortgages trade different pricing and equity expectations. Compare terms for your market — never treat any rate as an approval.",
  },
  {
    title: "Deposit, tenor, and installment-to-income",
    body: "Your equity contribution, loan tenor, and installment-to-income ratio (often around one-third of take-home pay) drive affordability. Use the calculator to see how changing deposit or years moves the monthly payment.",
  },
  {
    title: "Documents lenders usually ask for",
    body: "Expect several months of salary statements, valid ID, recent payslips, an employment letter, and any scheme-specific evidence for your market. Upload once in HomeGauge; advisors review before you approach a lender.",
  },
];

export default function LearnPage() {
  return (
    <div className="mx-auto max-w-3xl px-5 py-12">
      <Link href="/" className="font-[family-name:var(--font-display)] text-2xl font-semibold">
        Home<span className="text-leaf">Gauge</span>
      </Link>
      <h1 className="mt-8 font-[family-name:var(--font-display)] text-4xl font-semibold">Learn</h1>
      <p className="mt-3 text-muted">
        Short guides so you can talk to lenders with clearer expectations. Educational only — not advice or an offer of credit.
      </p>
      <div className="mt-10 space-y-8">
        {guides.map((g) => (
          <article key={g.title}>
            <h2 className="font-[family-name:var(--font-display)] text-2xl font-semibold">{g.title}</h2>
            <p className="mt-2 text-[color:var(--ink-soft)] leading-relaxed">{g.body}</p>
          </article>
        ))}
      </div>
      <div className="mt-12 flex flex-wrap gap-3">
        <Link href="/app/assessment" className="rounded-md bg-leaf px-4 py-2.5 text-sm font-semibold text-white">
          Check My Mortgage Eligibility
        </Link>
        <Link href="/calculator" className="rounded-md border border-[color:var(--line)] px-4 py-2.5 text-sm font-semibold">
          Affordability calculator
        </Link>
      </div>
    </div>
  );
}
