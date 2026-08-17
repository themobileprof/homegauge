export type RateFields = {
  interest_rate?: number | null;
  interest_rate_min?: number | null;
  interest_rate_max?: number | null;
  interest_rate_type?: string | null;
};

export function hasRateBand(p: RateFields | null | undefined): boolean {
  if (!p) return false;
  return p.interest_rate_min != null && p.interest_rate_max != null && p.interest_rate_min < p.interest_rate_max;
}

export function isNegotiableRate(p: RateFields | null | undefined): boolean {
  if (!p) return false;
  return p.interest_rate_type === "negotiable" || hasRateBand(p);
}

export function headlineRate(p: RateFields | null | undefined): number | null {
  if (!p) return null;
  if (p.interest_rate != null) return p.interest_rate;
  if (hasRateBand(p) && p.interest_rate_min != null && p.interest_rate_max != null) {
    return Math.round(((p.interest_rate_min + p.interest_rate_max) / 2) * 1000) / 1000;
  }
  return p.interest_rate_min ?? p.interest_rate_max ?? null;
}

export function clampRate(n: number, p: RateFields | null | undefined): number {
  if (!p || Number.isNaN(n)) return n;
  let v = n;
  if (p.interest_rate_min != null) v = Math.max(v, p.interest_rate_min);
  if (p.interest_rate_max != null) v = Math.min(v, p.interest_rate_max);
  return v;
}

function fmtPct(n: number) {
  return Number.isInteger(n) ? String(n) : String(n);
}

function typeLabel(t?: string | null) {
  if (t === "negotiable") return "negotiable";
  if (t === "variable") return "variable";
  if (t === "fixed") return "fixed";
  return t || "";
}

/** Catalog/compare copy: band first, never a lone figure that looks like an offer. */
export function formatRate(p: RateFields | null | undefined): string {
  if (!p) return "—";
  const kind = typeLabel(p.interest_rate_type);
  if (hasRateBand(p) && p.interest_rate_min != null && p.interest_rate_max != null) {
    const band = `from ${fmtPct(p.interest_rate_min)}–${fmtPct(p.interest_rate_max)}%`;
    return kind ? `${band} ${kind}` : band;
  }
  if (p.interest_rate != null) {
    return kind ? `${fmtPct(p.interest_rate)}% ${kind}` : `${fmtPct(p.interest_rate)}%`;
  }
  return "—";
}
