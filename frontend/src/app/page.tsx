import Image from "next/image";
import Link from "next/link";

const bp = process.env.NEXT_PUBLIC_BASE_PATH ?? "";

const steps = [
  {
    title: "Share your salary picture",
    body: "Tell us about your job and upload about 6 months of salary account statements showing a clear monthly credit.",
  },
  {
    title: "See what you may qualify for",
    body: "We compare your profile to mortgage products in your selected market — without pretending a bank has approved you.",
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
    a: "The part of the home price you pay yourself (your deposit). Many mortgage products ask for about 10–30%, depending on the market and lender.",
  },
  {
    q: "Are the rates on HomeGauge final?",
    a: "We show when each product was last verified. Always confirm current terms with the lender before you commit.",
  },
];

export default function HomePage() {
  return (
    <div className="text-ink">
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
              Built for salaried homebuyers who want a clear path to home finance — without property-listing noise or false “you’re approved” promises. Pick your market and start with products available there.
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

          <div className="relative min-h-[28rem] animate-[fadeUp_0.9s_ease_both] overflow-hidden rounded-2xl border border-[color:var(--line)] shadow-[0_30px_80px_rgba(12,31,23,0.25)] md:min-h-[36rem]">
            <Image
              src={`${bp}/images/hero-couple.jpg`}
              alt="A young couple standing at the front of a house they hope to own"
              fill
              priority
              className="object-cover object-[50%_20%]"
              sizes="(min-width: 768px) 45vw, 100vw"
            />
            <div className="absolute inset-x-0 bottom-0 z-10 bg-ink p-6 text-paper md:p-8">
              <p className="text-sm uppercase tracking-[0.18em] text-gold">A new homeownership reality</p>
              <p className="mt-3 font-[family-name:var(--font-display)] text-xl font-semibold leading-tight">
                For younger buyers and couples, mortgages are the practical path to homeownership.
              </p>
              <ul className="mt-6 space-y-3 text-sm leading-relaxed text-paper/90">
                <li className="flex gap-3"><span className="text-gold">01</span> Home prices are rising faster than incomes in most markets.</li>
                <li className="flex gap-3"><span className="text-gold">02</span> Mortgages let you spread cost over time instead of waiting years to save the full price.</li>
                <li className="flex gap-3"><span className="text-gold">03</span> HomeGauge helps you understand options early, so you can plan with clarity and move with confidence.</li>
              </ul>
            </div>
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
          <div className="grid gap-10 md:grid-cols-2 md:items-center">
            <div className="relative min-h-[22rem] overflow-hidden rounded-2xl border border-[color:var(--line)] md:min-h-[28rem]">
              <Image
                src={`${bp}/images/plan-kitchen.jpg`}
                alt="A young professional reviewing mortgage paperwork at a kitchen table"
                fill
                className="object-cover object-top"
                sizes="(min-width: 768px) 50vw, 100vw"
              />
            </div>
            <div>
              <p className="text-sm font-semibold uppercase tracking-[0.16em] text-leaf">Why HomeGauge</p>
              <h2 className="mt-3 font-[family-name:var(--font-display)] text-3xl font-semibold md:text-4xl">
                Trust over theatre.
              </h2>
              <p className="mt-4 text-muted leading-relaxed">
                Mortgage terms move with policy and lender appetite in every market. We surface fees, equity needs, and verification dates — and we keep humans available when automation is not enough.
              </p>
            </div>
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

        <section className="relative overflow-hidden border-y border-[color:var(--line)] text-paper">
          <Image
            src={`${bp}/images/street-walk.jpg`}
            alt="A young couple walking a residential street, looking toward homes they might one day buy"
            fill
            className="object-cover"
            sizes="100vw"
          />
          <div className="absolute inset-0 bg-[#0c1f17]/78" />
          <div className="relative z-10 mx-auto flex max-w-6xl flex-col gap-6 px-5 py-20 md:flex-row md:items-center md:justify-between md:px-8">
            <div>
              <h2 className="font-[family-name:var(--font-display)] text-3xl font-semibold">Try the affordability calculator</h2>
              <p className="mt-3 max-w-xl text-sm text-paper/80">
                Change price, deposit, rate, and tenor. See estimated monthly repayments in your market currency — clearly marked as estimates.
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

        <footer className="border-t border-[color:var(--line)] py-10 text-center text-sm text-muted">
          HomeGauge is not a bank or lender. Educational estimates only.
        </footer>
      </main>
    </div>
  );
}
