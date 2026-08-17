"use client";

import Link from "next/link";

export default function LenderHomePage() {
  return (
    <div className="mx-auto max-w-5xl px-5 py-10 md:px-8">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-[#1f4d6b]">Lender portal</p>
      <h1 className="mt-2 font-[family-name:var(--font-display)] text-4xl font-semibold">Referral pipeline</h1>
      <p className="mt-3 max-w-2xl text-muted">
        This workspace is for lender staff reviewing HomeGauge-prepared files — not for checking personal eligibility or running the advisor queue.
      </p>
      <div className="mt-10 grid gap-4 md:grid-cols-3">
        {[
          ["Ready for review", "Cases marked ready for submission will appear here."],
          ["Product terms", "Keep your published rates and document lists current."],
          ["Decisions", "Record additional information requests after your credit process."],
        ].map(([title, body]) => (
          <div key={title} className="rounded-xl border border-[#1f4d6b]/15 bg-white/80 p-5">
            <h2 className="font-semibold">{title}</h2>
            <p className="mt-2 text-sm text-muted">{body}</p>
          </div>
        ))}
      </div>
      <Link href="/mortgages" className="mt-8 inline-block text-sm font-semibold text-[#1f4d6b]">
        View products on HomeGauge →
      </Link>
    </div>
  );
}
