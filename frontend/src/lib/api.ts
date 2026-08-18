export function outcomeLabel(outcome: string) {
  switch (outcome) {
    case "likely_eligible":
      return "Likely eligible";
    case "potentially_eligible":
      return "Potentially eligible";
    case "may_require_review":
      return "May require additional review";
    case "unlikely":
      return "Unlikely to qualify";
    case "more_info_required":
      return "More information required";
    default:
      return outcome;
  }
}

/** @deprecated Prefer useCountry().money — kept for gradual migration */
export function naira(n: number | null | undefined) {
  if (n == null || Number.isNaN(n)) return "—";
  return new Intl.NumberFormat("en-NG", {
    style: "currency",
    currency: "NGN",
    maximumFractionDigits: 0,
  }).format(n);
}

// Same-origin by default so the session cookie is first-party (required for advisor/admin auth).
// Set NEXT_PUBLIC_API_URL only when the browser must call the API on another host.
export const apiBase = process.env.NEXT_PUBLIC_API_URL || "";

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${apiBase}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init.headers || {}),
    },
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || `Request failed (${res.status})`);
  }
  return data as T;
}
