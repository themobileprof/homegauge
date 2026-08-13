import Link from "next/link";

const steps = [
  {
    title: "Share your salary picture",
    body: "Tell us about your job and upload 6 months of salary account statements showing a clear monthly credit.",
  },
  {
    title: "See what you may qualify for",
    body: "We compare your profile to real Nigerian mortgage products — NHF, MREIF, and commercial options — without pretending a bank has approved you.",
  },
  {
    title: "Prepare documents & get help",
    body: "Get a personal checklist, track your progress, and request a HomeGauge advisor when you want a human in the loop.",
  },
];

const faqs = [
  {
    q: "Is HomeGauge a bank?",
    a: "No. We help you understand products, eligibility, and paperwork. Only a licensed lender can approve a mortgage.",
  },
  {
    q: "Who can use the eligibility check today?",
    a: "Salaried workers with a clear salary credit into one account for about 6 months. Self-employed and diaspora paths come later.",
  },
  {
    q: "What is equity contribution?",
    a: "The part of the home price you pay yourself (your deposit). Many Nigerian products ask for about 10–30%.",
  },
  {
    q: "Are the rates on HomeGauge final?",
    a: "We show when each product was last verified. Always confirm current terms with the lender before you commit.",
  },
];

export default function HomePage() {
  return (
    <div className="min-h-screen text-ink">
      <header className="mx-auto flex w-full max-w-6xl items-center justify-between px-5 py-5 md:px-8">
        <Link href="/" className="font-[family-name:var(--font-display)] text-2xl font-semibold tracking-tight">
          Home<span className="text-leaf">Gauge</span>
        </Link>
        <nav className="hidden items-center gap-8 text-sm font-medium text-ink-soft md:flex">
          <a href="#how">How it works</a>
          <Link href="/calculator">Calculator</Link>
          <Link href="/mortgages">Mortgage options</Link>
          <Link href="/learn">Learn</Link>
        </nav>
        <div className="flex items-center gap-3">
          <Link href="/login" className="hidden text-sm font-semibold text-ink-soft sm:inline">
            Sign in
          </Link>
          <Link
            href="/register"
            className="rounded-md bg-leaf px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-leaf-deep"
          >
            Get started
          </Link>
        </div>
      </header>

      <main>
        <section className="relative mx-auto grid min-h-[calc(100vh-5.5rem)] w-full max-w-6xl items-center gap-10 px-5 pb-16 pt-6 md:grid-cols-[1.1fr_0.9fr] md:px-8 md:pb-24">
          <div className="animate-[fadeUp_0.7s_ease_both]">
            <p className="mb-4 font-[family-name:var(--font-display)] text-5xl font-semibold leading-[0.95] tracking-tight text-ink md:text-7xl">
              HomeGauge
            </p>
            <h1 className="max-w-xl text-2xl font-semibold leading-snug text-ink-soft md:text-3xl">
              Understand your mortgage. Know what you may qualify for. Get help getting approved.
            </h1>
            <p className="mt-5 max-w-lg text-base leading-relaxed text-muted md:text-lg">
              Built for Nigerians who earn a salary and want a clear path to home finance — without property-listing noise or false “you’re approved” promises.
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <Link
                href="/register?next=/app/assessment"
                className="rounded-md bg-ink px-5 py-3 text-sm font-semibold text-paper transition hover:bg-ink-soft"
              >
                Check My Mortgage Eligibility
              </Link>
              <Link
                href="/mortgages"
                className="rounded-md border border-[color:var(--line)] bg-white/50 px-5 py-3 text-sm font-semibold text-ink transition hover:bg-white"
              >
                Compare Mortgage Options
              </Link>
            </div>
            <p className="mt-6 max-w-md text-xs leading-relaxed text-muted">
              Eligibility results are educational estimates based on stated product criteria and your salary-account history. Not a loan offer.
            </p>
          </div>

          <div className="relative animate-[fadeUp_0.9s_ease_both] overflow-hidden rounded-2xl border border-[color:var(--line)] bg-[linear-gradient(160deg,#123526_0%,#1f6b45_48%,#0c1f17_100%)] p-6 text-paper shadow-[0_30px_80px_rgba(12,31,23,0.25)] md:p-8">
            <div className="absolute -right-10 -top-10 h-40 w-40 rounded-full bg-[radial-gradient(circle,rgba(196,163,90,0.35),transparent_70%)]" />
            <p className="text-sm uppercase tracking-[0.18em] text-gold">Salary-account review</p>
            <p className="mt-4 font-[family-name:var(--font-display)] text-3xl font-semibold leading-tight">
              6 clear salary credits. One account. Honest next steps.
            </p>
            <ul className="mt-8 space-y-4 text-sm leading-relaxed text-paper/90">
              <li className="flex gap-3"><span className="text-gold">01</span> We look for a repeating credit near month-end.</li>
              <li className="flex gap-3"><span className="text-gold">02</span> We match you to products like NHF and MREIF.</li>
              <li className="flex gap-3"><span className="text-gold">03</span> An advisor can review anything that needs a human eye.</li>
            </ul>
          </div>
        </section>

        <section id="how" className="border-t border-[color:var(--line)] bg-white/40">
          <div className="mx-auto max-w-6xl px-5 py-20 md:px-8">
            <p className="text-sm font-semibold uppercase tracking-[0.16em] text-leaf">How it works</p>
            <h2 className="mt-3 max-w-2xl font-[family-name:var(--font-display)] text-3xl font-semibold md:text-4xl">
              A calm path from confusion to readiness.
            </h2>
            <div className="mt-12 grid gap-8 md:grid-cols-3">
              {steps.map((step, i) => (
                <div key={step.title} className="border-t border-leaf/30 pt-5">
                  <p className="text-sm font-semibold text-leaf">Step {i + 1}</p>
                  <h3 className="mt-2 text-xl font-semibold">{step.title}</h3>
                  <p className="mt-3 text-sm leading-relaxed text-muted">{step.body}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="mx-auto max-w-6xl px-5 py-20 md:px-8">
          <div className="grid gap-10 md:grid-cols-[0.9fr_1.1fr] md:items-end">
            <div>
              <p className="text-sm font-semibold uppercase tracking-[0.16em] text-leaf">Why HomeGauge</p>
              <h2 className="mt-3 font-[family-name:var(--font-display)] text-3xl font-semibold md:text-4xl">
                Trust over theatre.
              </h2>
            </div>
            <p className="text-muted leading-relaxed">
              Nigerian mortgage terms move with policy and lender appetite. We surface fees, equity needs, and verification dates — and we keep humans available when automation is not enough.
            </p>
          </div>
          <div className="mt-12 grid gap-6 md:grid-cols-3">
            {[
              ["No fake approvals", "We say “likely eligible”, never “approved”, until a lender decides."],
              ["Salary-first honesty", "MVP review is built around real salary credits — the signal lenders already trust."],
              ["Advisor when it counts", "AI drafts the busywork; people handle judgement calls and lender submission."],
            ].map(([title, body]) => (
              <div key={title} className="rounded-xl bg-paper-2/70 p-6">
                <h3 className="text-lg font-semibold">{title}</h3>
                <p className="mt-3 text-sm leading-relaxed text-muted">{body}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="border-y border-[color:var(--line)] bg-[#0c1f17] text-paper">
          <div className="mx-auto flex max-w-6xl flex-col gap-6 px-5 py-16 md:flex-row md:items-center md:justify-between md:px-8">
            <div>
              <h2 className="font-[family-name:var(--font-display)] text-3xl font-semibold">Try the affordability calculator</h2>
              <p className="mt-3 max-w-xl text-sm text-paper/75">
                Change price, deposit, rate, and tenor. See estimated monthly repayments in ₦ — clearly marked as estimates.
              </p>
            </div>
            <Link href="/calculator" className="rounded-md bg-gold px-5 py-3 text-sm font-semibold text-ink hover:brightness-105">
              Open calculator
            </Link>
          </div>
        </section>

        <section className="mx-auto max-w-6xl px-5 py-20 md:px-8">
          <p className="text-sm font-semibold uppercase tracking-[0.16em] text-leaf">FAQ</p>
          <h2 className="mt-3 font-[family-name:var(--font-display)] text-3xl font-semibold">Plain answers</h2>
          <div className="mt-10 divide-y divide-[color:var(--line)]">
            {faqs.map((item) => (
              <details key={item.q} className="group py-5">
                <summary className="cursor-pointer list-none text-lg font-semibold marker:content-none">
                  {item.q}
                </summary>
                <p className="mt-3 max-w-3xl text-sm leading-relaxed text-muted">{item.a}</p>
              </details>
            ))}
          </div>
        </section>

        <section className="mx-auto max-w-6xl px-5 pb-24 md:px-8">
          <div className="rounded-2xl border border-[color:var(--line)] bg-white/70 px-6 py-10 text-center md:px-12">
            <h2 className="font-[family-name:var(--font-display)] text-3xl font-semibold">Ready to see where you stand?</h2>
            <p className="mx-auto mt-3 max-w-xl text-muted">
              Start with eligibility. Compare options. Prepare your papers. HomeGauge stays with you through the journey.
            </p>
            <Link
              href="/register"
              className="mt-8 inline-flex rounded-md bg-leaf px-5 py-3 text-sm font-semibold text-white hover:bg-leaf-deep"
            >
              Check My Mortgage Eligibility
            </Link>
          </div>
        </section>
      </main>

      <footer className="border-t border-[color:var(--line)] px-5 py-10 text-sm text-muted md:px-8">
        <div className="mx-auto flex max-w-6xl flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <p className="font-[family-name:var(--font-display)] text-lg text-ink">HomeGauge</p>
          <p>Not a bank or lender. Mortgage information should be verified with the relevant institution.</p>
        </div>
      </footer>
    </div>
  );
}
