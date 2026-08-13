"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { api } from "@/lib/api";

export type Country = {
  code: string;
  name: string;
  currency_code: string;
  locale: string;
  region_label: string;
  regions: string[];
  default_iti_pct: number;
  status: "active" | "coming_soon" | "inactive";
  sort_order: number;
};

type CountryContextValue = {
  countries: Country[];
  country: Country | null;
  countryCode: string;
  setCountryCode: (code: string) => void;
  money: (n: number | null | undefined) => string;
  loading: boolean;
};

const STORAGE_KEY = "homegauge_country";
const CountryContext = createContext<CountryContextValue | null>(null);

function parseRegions(raw: unknown): string[] {
  if (Array.isArray(raw)) return raw.map(String);
  if (typeof raw === "string") {
    try {
      const parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed.map(String) : [];
    } catch {
      return [];
    }
  }
  return [];
}

export function formatMoney(
  n: number | null | undefined,
  currency = "NGN",
  locale = "en",
) {
  if (n == null || Number.isNaN(n)) return "—";
  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency,
    maximumFractionDigits: 0,
  }).format(n);
}

export function CountryProvider({ children }: { children: ReactNode }) {
  const [countries, setCountries] = useState<Country[]>([]);
  const [countryCode, setCountryCodeState] = useState("NG");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const saved = typeof window !== "undefined" ? localStorage.getItem(STORAGE_KEY) : null;
    api<{ countries: Array<Country & { regions?: unknown }>; default_country_code: string }>(
      "/api/v1/countries?include=coming_soon",
    )
      .then((data) => {
        const list = (data.countries || []).map((c) => ({
          ...c,
          regions: parseRegions(c.regions),
        }));
        setCountries(list);
        const preferred =
          (saved && list.find((c) => c.code === saved && c.status === "active")?.code) ||
          list.find((c) => c.code === data.default_country_code && c.status === "active")?.code ||
          list.find((c) => c.status === "active")?.code ||
          "NG";
        setCountryCodeState(preferred);
      })
      .catch(() => {
        setCountries([
          {
            code: "NG",
            name: "Nigeria",
            currency_code: "NGN",
            locale: "en-NG",
            region_label: "State",
            regions: [],
            default_iti_pct: 35,
            status: "active",
            sort_order: 10,
          },
        ]);
      })
      .finally(() => setLoading(false));
  }, []);

  const setCountryCode = useCallback((code: string) => {
    setCountryCodeState(code);
    if (typeof window !== "undefined") localStorage.setItem(STORAGE_KEY, code);
  }, []);

  const country = useMemo(
    () => countries.find((c) => c.code === countryCode) || countries.find((c) => c.status === "active") || null,
    [countries, countryCode],
  );

  const money = useCallback(
    (n: number | null | undefined) =>
      formatMoney(n, country?.currency_code || "NGN", country?.locale || "en"),
    [country],
  );

  const value = useMemo(
    () => ({ countries, country, countryCode, setCountryCode, money, loading }),
    [countries, country, countryCode, setCountryCode, money, loading],
  );

  return <CountryContext.Provider value={value}>{children}</CountryContext.Provider>;
}

export function useCountry() {
  const ctx = useContext(CountryContext);
  if (!ctx) throw new Error("useCountry must be used within CountryProvider");
  return ctx;
}

export function CountrySwitcher({ className = "" }: { className?: string }) {
  const { countries, countryCode, setCountryCode } = useCountry();
  if (countries.length === 0) return null;
  return (
    <label className={`inline-flex items-center gap-2 text-sm ${className}`}>
      <span className="sr-only">Country</span>
      <select
        value={countryCode}
        onChange={(e) => {
          const next = countries.find((c) => c.code === e.target.value);
          if (next?.status === "active") setCountryCode(next.code);
        }}
        className="rounded-md border border-[color:var(--line)] bg-white/80 px-2 py-1.5 text-sm font-medium text-ink"
      >
        {countries.map((c) => (
          <option key={c.code} value={c.code} disabled={c.status !== "active"}>
            {c.name}
            {c.status === "coming_soon" ? " (soon)" : ""}
          </option>
        ))}
      </select>
    </label>
  );
}
