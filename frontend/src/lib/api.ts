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

// Same-origin by default so the session cookie is first-party.
// Set NEXT_PUBLIC_API_URL only when the API lives on a different host.
function resolveApiBasePath(): string {
  if (process.env.NEXT_PUBLIC_API_URL) return process.env.NEXT_PUBLIC_API_URL;

  // Keep product boundaries: only prefix when this app runs under /mortgage.
  if (typeof window !== "undefined") {
    const p = window.location.pathname;
    if (p === "/mortgage" || p.startsWith("/mortgage/")) return "/mortgage";
  }

  return "";
}

export const apiBase = resolveApiBasePath();

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
