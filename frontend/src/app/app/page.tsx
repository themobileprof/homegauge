import Link from "next/link";

export default function AppHome() {
  return (
    <div className="mx-auto max-w-3xl px-5 py-12">
      <Link href="/" className="font-[family-name:var(--font-display)] text-2xl font-semibold">
        Home<span className="text-leaf">Gauge</span>
      </Link>
      <h1 className="mt-8 font-[family-name:var(--font-display)] text-4xl font-semibold">Your mortgage journey</h1>
      <p className="mt-3 text-muted">
        Dashboard modules (readiness, documents, advisor) ship next. You are signed in to HomeGauge.
      </p>
      <ol className="mt-10 space-y-3 text-sm">
        <li>✓ Account</li>
        <li className="text-leaf">● Eligibility assessment — coming in the next slice</li>
        <li className="text-muted">○ Documents</li>
        <li className="text-muted">○ Application</li>
      </ol>
      <Link href="/calculator" className="mt-8 inline-flex text-sm font-semibold text-leaf">
        Open affordability calculator →
      </Link>
    </div>
  );
}
