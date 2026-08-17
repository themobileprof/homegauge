"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { api, apiBase } from "@/lib/api";

export type Role = "CUSTOMER" | "ADVISOR" | "ADMIN" | "LENDER_USER";

export type AuthUser = {
  id: string;
  email: string;
  role: Role;
  status?: string;
};

type AuthContextValue = {
  user: AuthUser | null;
  loading: boolean;
  refresh: () => Promise<AuthUser | null>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function homeForRole(role?: Role | null) {
  switch (role) {
    case "ADMIN":
      return "/admin";
    case "ADVISOR":
      return "/advisor";
    case "LENDER_USER":
      return "/lender";
    default:
      return "/app";
  }
}

export function roleLabel(role?: Role | null) {
  switch (role) {
    case "ADMIN":
      return "Admin";
    case "ADVISOR":
      return "Advisor";
    case "LENDER_USER":
      return "Lender";
    case "CUSTOMER":
      return "Homebuyer";
    default:
      return "";
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try {
      const data = await api<{ user: AuthUser }>("/api/v1/auth/me");
      setUser(data.user);
      return data.user;
    } catch {
      setUser(null);
      return null;
    }
  }, []);

  useEffect(() => {
    refresh().finally(() => setLoading(false));
  }, [refresh]);

  const logout = useCallback(async () => {
    await fetch(`${apiBase}/api/v1/auth/logout`, { method: "POST", credentials: "include" }).catch(() => undefined);
    setUser(null);
  }, []);

  const value = useMemo(() => ({ user, loading, refresh, logout }), [user, loading, refresh, logout]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
