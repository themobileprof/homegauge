import Link from "next/link";

export default function MortgagesPage() {
  return (
    <div className="mx-auto max-w-4xl px-5 py-12">
      <Link href="/" className="font-[family-name:var(--font-display)] text-2xl font-semibold">
        Home<span className="text-leaf">Gauge</span>
      </Link>
      <h1 className="mt-8 font-[family-name:var(--font-display)] text-4xl font-semibold">Mortgage options</h1>
      <p className="mt-3 max-w-2xl text-muted">
        Product browse and comparison will load from the live database (NHF, Stanbic MREIF, commercial indicative). Each card will show last verified date.
      </p>
    </div>
  );
}
