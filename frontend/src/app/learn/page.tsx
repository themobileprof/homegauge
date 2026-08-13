import Link from "next/link";

export default function LearnPage() {
  return (
    <div className="mx-auto max-w-3xl px-5 py-12">
      <Link href="/" className="font-[family-name:var(--font-display)] text-2xl font-semibold">
        Home<span className="text-leaf">Gauge</span>
      </Link>
      <h1 className="mt-8 font-[family-name:var(--font-display)] text-4xl font-semibold">Learn</h1>
      <p className="mt-3 text-muted">
        Guides on NHF, deposits, salary-account readiness, and what happens after a lender decision will live here.
      </p>
    </div>
  );
}
