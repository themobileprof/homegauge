"use client";

import Link from "next/link";
import { roleLabel, useAuth, type Role } from "@/lib/auth";

type Guide = { title: string; body: string };
type LinkAction = { href: string; label: string; primary?: boolean };

type RoleGuide = {
  eyebrow: string;
  title: string;
  intro: string;
  theme: "buyer" | "advisor" | "lender" | "admin" | "public";
  guides: Guide[];
  actions: LinkAction[];
};

const publicGuide: RoleGuide = {
  eyebrow: "For first-time homebuyers",
  title: "How HomeGauge works before a bank decides",
  intro:
    "HomeGauge is for salaried first-time buyers — entirely pre-disbursement. You qualify, get documents and costs ready, then an advisor helps you submit a file. We are not a bank and do not service the loan after disbursement.",
  theme: "public",
  guides: [
    {
      title: "1. Qualify against salary-fit products",
      body: "Lenders look for clear salary credits on one account — similar amounts near payday. Our eligibility check mirrors that pattern. Results are educational estimates, not an approval.",
    },
    {
      title: "2. Get ready: documents and equity",
      body: "Expect salary statements, ID, payslips, an employment letter, and any scheme evidence. Equity contribution, tenor, and installment-to-income (often near one-third of take-home) drive affordability.",
    },
    {
      title: "3. Fund & settle known costs",
      body: "Valuation, legal, and processing fees often come due before disbursement. When a product is chosen, HomeGauge can collect those into a case account so the file is not stuck on unpaid fees.",
    },
    {
      title: "4. Submit with an advisor",
      body: "An advisor packages a lender-ready file. Only a licensed lender can approve credit. Scheme loans (e.g. NHF) and commercial mortgages price and contribute differently — compare terms, never treat a rate as an offer.",
    },
  ],
  actions: [
    { href: "/register", label: "Create a buyer account", primary: true },
    { href: "/calculator", label: "Try the calculator" },
    { href: "/mortgages", label: "Browse products" },
  ],
};

const buyerGuide: RoleGuide = {
  eyebrow: "Homebuyer guides",
  title: "Your path to a lender-ready file",
  intro:
    "Use these steps in order. HomeGauge ends when your file is submitted for underwriting — not after the bank pays out.",
  theme: "buyer",
  guides: [
    {
      title: "Qualify",
      body: "Run eligibility, then pick a salary-fit product. “Likely” means the stated rules look compatible — the lender still decides.",
    },
    {
      title: "Get ready",
      body: "Upload required documents once. Fix anything your advisor sends back. Confirm equity and known fees on your dashboard.",
    },
    {
      title: "Fund & settle",
      body: "Pay valuation, legal, and processing fees into your case collection account when that step unlocks. Your advisor can see the same balance.",
    },
    {
      title: "Submit",
      body: "When documents and costs are ready, your advisor submits to the product’s lender. Watch your dashboard for requests for more information.",
    },
  ],
  actions: [
    { href: "/app/assessment", label: "Check eligibility", primary: true },
    { href: "/app", label: "Open my journey" },
    { href: "/calculator", label: "Affordability calculator" },
  ],
};

const advisorGuide: RoleGuide = {
  eyebrow: "Advisor playbook",
  title: "Working a salary-first case file",
  intro:
    "You own the dossier until it is lender-ready. HomeGauge is pre-disbursement only — your job is a clean file and a clear handoff, not post-loan servicing.",
  theme: "advisor",
  guides: [
    {
      title: "Open the working file first",
      body: "Each case is a dossier: eligibility snapshot, product fit, documents, get-ready costs, and notes. Brief from the file — do not reinvent the buyer’s answers from chat.",
    },
    {
      title: "Point the file at one product",
      body: "Product choice drives document checklist, fee obligations, and which lender (if any) has a portal. Prefer salary-fit outcomes the buyer already saw.",
    },
    {
      title: "Documents and Fund & settle",
      body: "Accept or send back uploads with a clear reason. When fees are collectable, confirm the buyer’s funding progress before marking the file ready to submit.",
    },
    {
      title: "Liaison vs portal lenders",
      body: "If the lender has a HomeGauge account, they update review status themselves after you submit. If not, you record their phone/email updates on the file so the buyer still sees progress.",
    },
    {
      title: "Hand up only when ready",
      body: "Ready for submission means document-ready and product-fit — not that a bank has approved. Admin may still assign or track top-level case status.",
    },
  ],
  actions: [
    { href: "/advisor", label: "Open working files", primary: true },
    { href: "/mortgages", label: "Product catalog" },
  ],
};

const lenderGuide: RoleGuide = {
  eyebrow: "Lender portal guides",
  title: "Referral files and your products",
  intro:
    "You only see files submitted against your products, and you only manage products linked to your institution. A status you record here is pipeline hygiene — not a formal credit decision on HomeGauge.",
  theme: "lender",
  guides: [
    {
      title: "Pipeline buckets",
      body: "New submissions land under “New to you.” Move to review when you open the file, or ask for more information so the buyer and advisor know what is missing. Outcomes (approve / reject / complete) are recorded for the case trail; admin may still hold the platform-level status.",
    },
    {
      title: "What a HomeGauge file contains",
      body: "Eligibility answers, preferred product, advisor notes, and documents prepared for salaried first-time buyers. Treat it as a packaged referral — underwrite with your own process.",
    },
    {
      title: "Your products only",
      body: "Under Your products, edit rates, fees, equity, and income bands for products associated with your lender. Saves refresh eligibility rules. HomeGauge re-verifies catalog trust after edits; you cannot self-verify.",
    },
    {
      title: "Pre-disbursement scope",
      body: "Buyers may settle valuation and legal fees before you disburse. HomeGauge does not service the loan after payout.",
    },
  ],
  actions: [
    { href: "/lender", label: "Open pipeline", primary: true },
    { href: "/lender/products", label: "Manage your products" },
  ],
};

const adminGuide: RoleGuide = {
  eyebrow: "Admin reference",
  title: "Keep the catalog and cases trustworthy",
  intro:
    "Admins oversee users, product verification, and case assignment. Day-to-day buyer coaching and lender pipeline work live in those portals — use guides there when you need role-specific detail.",
  theme: "admin",
  guides: [
    {
      title: "Products and verification",
      body: "Publish or correct mortgage products by market. Active products appear in Compare. When a lender edits their own product, verification resets so you can re-check stated terms.",
    },
    {
      title: "Users and lender linkage",
      body: "Lender users must be linked to a lender org — that scopes their pipeline and product list. Advisors work assigned case files; buyers own their journey dashboard.",
    },
    {
      title: "Cases and reports",
      body: "Assign unassigned files, track ready-for-submission, and use reports for advisor and buyer volume. Status at admin level is platform coordination, not a bank approval.",
    },
  ],
  actions: [
    { href: "/admin", label: "Platform overview", primary: true },
    { href: "/admin/products", label: "Products" },
    { href: "/admin/cases", label: "Cases" },
  ],
};

function guideForRole(role?: Role | null): RoleGuide {
  switch (role) {
    case "CUSTOMER":
      return buyerGuide;
    case "ADVISOR":
      return advisorGuide;
    case "LENDER_USER":
      return lenderGuide;
    case "ADMIN":
      return adminGuide;
    default:
      return publicGuide;
  }
}

const themeClass: Record<RoleGuide["theme"], { accent: string; chip: string; primaryBtn: string }> = {
  public: {
    accent: "text-leaf",
    chip: "bg-leaf/10 text-leaf",
    primaryBtn: "bg-leaf text-white hover:bg-leaf-deep",
  },
  buyer: {
    accent: "text-leaf",
    chip: "bg-leaf/10 text-leaf",
    primaryBtn: "bg-leaf text-white hover:bg-leaf-deep",
  },
  advisor: {
    accent: "text-[#8a6d28]",
    chip: "bg-[#8a6d28]/12 text-[#8a6d28]",
    primaryBtn: "bg-[#8a6d28] text-white hover:bg-[#6f5720]",
  },
  lender: {
    accent: "text-[#1f4d6b]",
    chip: "bg-[#1f4d6b]/10 text-[#1f4d6b]",
    primaryBtn: "bg-[#1f4d6b] text-white hover:bg-[#163a51]",
  },
  admin: {
    accent: "text-ink",
    chip: "bg-ink/10 text-ink",
    primaryBtn: "bg-ink text-paper hover:bg-ink-soft",
  },
};

export default function LearnPage() {
  const { user, loading } = useAuth();
  const pack = guideForRole(user?.role);
  const theme = themeClass[pack.theme];
  const roleName = user ? roleLabel(user.role) : null;

  return (
    <div className="mx-auto max-w-3xl px-5 py-12">
      <p className={`text-xs font-semibold uppercase tracking-[0.16em] ${theme.accent}`}>{pack.eyebrow}</p>
      <h1 className="mt-2 font-[family-name:var(--font-display)] text-4xl font-semibold tracking-tight">{pack.title}</h1>
      {!loading && roleName && (
        <p className={`mt-3 inline-flex rounded-sm px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wide ${theme.chip}`}>
          Showing guides for {roleName}
        </p>
      )}
      <p className="mt-4 text-sm leading-relaxed text-muted">{pack.intro}</p>
      <p className="mt-2 text-xs text-muted">Educational only — not advice or an offer of credit.</p>

      <ol className="mt-10 space-y-8">
        {pack.guides.map((g, i) => (
          <li key={g.title} className="flex gap-4">
            <span
              className={`mt-1 flex h-7 w-7 shrink-0 items-center justify-center rounded-sm text-xs font-semibold ${theme.chip}`}
              aria-hidden
            >
              {i + 1}
            </span>
            <article>
              <h2 className="font-[family-name:var(--font-display)] text-2xl font-semibold leading-snug">{g.title}</h2>
              <p className="mt-2 leading-relaxed text-[color:var(--ink-soft)]">{g.body}</p>
            </article>
          </li>
        ))}
      </ol>

      <div className="mt-12 flex flex-wrap gap-3">
        {pack.actions.map((a) => (
          <Link
            key={a.href + a.label}
            href={a.href}
            className={
              a.primary
                ? `rounded-md px-4 py-2.5 text-sm font-semibold ${theme.primaryBtn}`
                : "rounded-md border border-[color:var(--line)] px-4 py-2.5 text-sm font-semibold hover:bg-white"
            }
          >
            {a.label}
          </Link>
        ))}
      </div>

      {user?.role === "ADVISOR" || user?.role === "LENDER_USER" || user?.role === "ADMIN" ? (
        <p className="mt-10 border-t border-[color:var(--line)] pt-6 text-sm text-muted">
          Helping a buyer understand the journey?{" "}
          <Link href="/calculator" className="font-semibold text-leaf hover:underline">
            Open the calculator
          </Link>{" "}
          or{" "}
          <Link href="/mortgages" className="font-semibold text-leaf hover:underline">
            product catalog
          </Link>{" "}
          — buyer-facing steps live on their dashboard after they sign in.
        </p>
      ) : null}
    </div>
  );
}
